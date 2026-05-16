package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	backendHealthURL = "http://127.0.0.1:4848/api/info"
	webURL           = "http://127.0.0.1:5228"

	launcherHeartbeatPath = "/__avmatrix_launcher/heartbeat"
	launcherClosedPath    = "/__avmatrix_launcher/closed"
	launcherUITimeout     = 15 * time.Second
	launcherUICloseGrace  = 2 * time.Second
)

type launcherPaths struct {
	exePath    string
	rootDir    string
	homeDir    string
	logDir     string
	webDist    string
	serverExe  string
	backendExe string
	stateFile  string
}

type launcherState struct {
	RootDir     string    `json:"rootDir"`
	LauncherPID int       `json:"launcherPid"`
	BackendPID  int       `json:"backendPid"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Status      string    `json:"status"`
}

type backendProcess struct {
	pid  int
	done <-chan error
}

func main() {
	paths, err := resolvePaths()
	if err != nil {
		log.Fatalf("resolve paths: %v", err)
	}
	initLog(paths)

	action := parseAction(os.Args[1:])
	var runErr error
	switch action {
	case "register":
		runErr = registerProtocol(paths)
	case "reset":
		runErr = resetRuntime(paths)
	case "stop":
		runErr = stopRuntime(paths)
	default:
		runErr = startRuntime(paths)
	}
	if runErr != nil {
		writeState(paths, "error", 0)
		log.Fatalf("%s failed: %v", action, runErr)
	}
}

func resolvePaths() (launcherPaths, error) {
	exePath, err := os.Executable()
	if err != nil {
		return launcherPaths{}, err
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return launcherPaths{}, err
	}

	homeDir := filepath.Dir(exePath)
	rootDir := filepath.Dir(homeDir)
	stateFile := filepath.Join(os.TempDir(), "avmatrix-launcher-"+shortHash(rootDir)+".json")
	return launcherPaths{
		exePath:    exePath,
		rootDir:    rootDir,
		homeDir:    homeDir,
		logDir:     filepath.Join(homeDir, "logs"),
		webDist:    filepath.Join(homeDir, "web-dist"),
		serverExe:  filepath.Join(homeDir, "server-bundle", "avmatrix-server.exe"),
		backendExe: filepath.Join(homeDir, "server-bundle", "avmatrix.exe"),
		stateFile:  stateFile,
	}, nil
}

func shortHash(value string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(filepath.Clean(value))))
	return strconv.FormatUint(uint64(h.Sum32()), 16)
}

func parseAction(args []string) string {
	if len(args) == 0 {
		return "start"
	}
	raw := strings.ToLower(strings.Join(args, " "))
	switch {
	case strings.Contains(raw, "register"):
		return "register"
	case strings.Contains(raw, "reset"):
		return "reset"
	case strings.Contains(raw, "stop"):
		return "stop"
	default:
		return "start"
	}
}

func initLog(paths launcherPaths) {
	if err := os.MkdirAll(paths.logDir, 0o755); err != nil {
		return
	}
	logFile, err := os.OpenFile(filepath.Join(paths.logDir, "launcher.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
}

func startRuntime(paths launcherPaths) error {
	log.Printf("start root=%s", paths.rootDir)
	if state, err := readState(paths); err == nil && state.LauncherPID != os.Getpid() && processAlive(state.LauncherPID) {
		if waitForURL(backendHealthURL, 4*time.Second) && waitForURL(webURL, 4*time.Second) {
			log.Printf("reusing existing launcher pid=%d", state.LauncherPID)
			return openBrowser(webURL)
		}
		log.Printf("stopping stale launcher pid=%d backend=%d", state.LauncherPID, state.BackendPID)
		stopPID(state.BackendPID)
		stopPID(state.LauncherPID)
		_ = os.Remove(paths.stateFile)
	}

	backend, err := ensureBackend(paths)
	if err != nil {
		return err
	}
	defer stopPID(backend.pid)

	lifecycle := newWebLifecycleMonitor(launcherUITimeout, launcherUICloseGrace)
	webServer := &http.Server{
		Addr:              "127.0.0.1:5228",
		Handler:           staticHandlerWithLifecycle(paths.webDist, lifecycle),
		ReadHeaderTimeout: 10 * time.Second,
	}

	webStarted := false
	if !urlReady(webURL) {
		if err := verifyWebDist(paths.webDist); err != nil {
			return err
		}
		webStarted = true
		go func() {
			err := webServer.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("web server failed: %v", err)
			}
		}()
	}
	if webStarted {
		lifecycle.start()
		defer lifecycle.stop()
		defer shutdownWeb(webServer)
	}

	writeState(paths, "starting", backend.pid)
	if !waitForURL(backendHealthURL, 90*time.Second) {
		writeState(paths, "error", backend.pid)
		return errors.New("backend did not become ready")
	}
	if !waitForURL(webURL, 90*time.Second) {
		writeState(paths, "error", backend.pid)
		return errors.New("web ui did not become ready")
	}
	writeState(paths, "ready", backend.pid)

	if err := openBrowser(webURL); err != nil {
		return err
	}

	waitForExit(paths, backend, lifecycleDone(webStarted, lifecycle))
	return nil
}

func ensureBackend(paths launcherPaths) (backendProcess, error) {
	if urlReady(backendHealthURL) {
		log.Printf("backend already ready at %s", backendHealthURL)
		return backendProcess{}, nil
	}
	if _, err := os.Stat(paths.serverExe); err != nil {
		return backendProcess{}, fmt.Errorf("packaged backend missing: %s", paths.serverExe)
	}

	cmd := exec.Command(paths.serverExe)
	cmd.Dir = filepath.Dir(paths.serverExe)
	cmd.SysProcAttr = hiddenProcAttr()
	attachLog(paths, cmd, "backend.log")
	if err := cmd.Start(); err != nil {
		return backendProcess{}, err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	log.Printf("backend pid=%d", cmd.Process.Pid)
	return backendProcess{pid: cmd.Process.Pid, done: done}, nil
}

func verifyWebDist(webDist string) error {
	if stat, err := os.Stat(webDist); err != nil || !stat.IsDir() {
		return fmt.Errorf("web-dist missing: %s", webDist)
	}
	if _, err := os.Stat(filepath.Join(webDist, "index.html")); err != nil {
		return fmt.Errorf("web-dist index missing: %s", filepath.Join(webDist, "index.html"))
	}
	return nil
}

func staticHandler(webDist string) http.Handler {
	return staticHandlerWithLifecycle(webDist, nil)
}

func staticHandlerWithLifecycle(webDist string, lifecycle *webLifecycleMonitor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if lifecycle != nil && lifecycle.handle(w, r) {
			return
		}
		rel := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			rel = "index.html"
		}
		target := filepath.Join(webDist, rel)
		if stat, err := os.Stat(target); err == nil && !stat.IsDir() {
			serveStaticFile(w, r, target, lifecycle)
			return
		}
		serveStaticFile(w, r, filepath.Join(webDist, "index.html"), lifecycle)
	})
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, target string, lifecycle *webLifecycleMonitor) {
	if lifecycle == nil || filepath.Base(target) != "index.html" {
		http.ServeFile(w, r, target)
		return
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		http.ServeFile(w, r, target)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(injectLauncherLifecycle(raw))
}

func injectLauncherLifecycle(raw []byte) []byte {
	const marker = "</body>"
	html := string(raw)
	index := strings.LastIndex(strings.ToLower(html), marker)
	if index < 0 {
		return append(append([]byte{}, raw...), []byte(launcherLifecycleScript)...)
	}
	result := make([]byte, 0, len(raw)+len(launcherLifecycleScript))
	result = append(result, raw[:index]...)
	result = append(result, launcherLifecycleScript...)
	result = append(result, raw[index:]...)
	return result
}

const launcherLifecycleScript = `<script data-avmatrix-launcher-lifecycle>
(() => {
  const heartbeat = "/__avmatrix_launcher/heartbeat";
  const closed = "/__avmatrix_launcher/closed";
  const ping = () => fetch(heartbeat, { method: "POST", cache: "no-store", keepalive: true }).catch(() => {});
  ping();
  const timer = setInterval(ping, 5000);
  window.addEventListener("pagehide", () => {
    clearInterval(timer);
    try {
      if (navigator.sendBeacon) {
        navigator.sendBeacon(closed, "");
      } else {
        fetch(closed, { method: "POST", cache: "no-store", keepalive: true }).catch(() => {});
      }
    } catch (_) {}
  });
})();
</script>`

type webLifecycleMonitor struct {
	mu            sync.Mutex
	timeout       time.Duration
	closeGrace    time.Duration
	checkInterval time.Duration
	seen          bool
	lastSeen      time.Time
	done          chan struct{}
	stopCh        chan struct{}
	doneOnce      sync.Once
	stopOnce      sync.Once
}

func newWebLifecycleMonitor(timeout time.Duration, closeGrace time.Duration) *webLifecycleMonitor {
	if timeout <= 0 {
		timeout = launcherUITimeout
	}
	if closeGrace <= 0 || closeGrace >= timeout {
		closeGrace = timeout / 3
	}
	return &webLifecycleMonitor{
		timeout:       timeout,
		closeGrace:    closeGrace,
		checkInterval: lifecycleCheckInterval(timeout),
		done:          make(chan struct{}),
		stopCh:        make(chan struct{}),
	}
}

func lifecycleCheckInterval(timeout time.Duration) time.Duration {
	interval := timeout / 4
	if interval < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	if interval > 2*time.Second {
		return 2 * time.Second
	}
	return interval
}

func (m *webLifecycleMonitor) start() {
	if m == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(m.checkInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if m.expired(time.Now()) {
					m.finish()
					return
				}
			case <-m.stopCh:
				return
			}
		}
	}()
}

func (m *webLifecycleMonitor) stop() {
	if m == nil {
		return
	}
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
}

func (m *webLifecycleMonitor) Done() <-chan struct{} {
	if m == nil {
		return nil
	}
	return m.done
}

func (m *webLifecycleMonitor) handle(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case launcherHeartbeatPath:
		if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		m.recordHeartbeat(time.Now())
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
		return true
	case launcherClosedPath:
		if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		m.recordClosed(time.Now())
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
		return true
	default:
		return false
	}
}

func (m *webLifecycleMonitor) recordHeartbeat(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seen = true
	m.lastSeen = now
}

func (m *webLifecycleMonitor) recordClosed(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.seen {
		return
	}
	m.lastSeen = now.Add(-(m.timeout - m.closeGrace))
}

func (m *webLifecycleMonitor) expired(now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.seen && now.Sub(m.lastSeen) > m.timeout
}

func (m *webLifecycleMonitor) finish() {
	m.doneOnce.Do(func() {
		close(m.done)
	})
}

func lifecycleDone(webStarted bool, lifecycle *webLifecycleMonitor) <-chan struct{} {
	if !webStarted || lifecycle == nil {
		return nil
	}
	return lifecycle.Done()
}

func waitForExit(paths launcherPaths, backend backendProcess, uiDone <-chan struct{}) {
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	if backend.done == nil {
		select {
		case <-uiDone:
			log.Printf("web ui session closed")
		case <-sig:
		}
		_ = os.Remove(paths.stateFile)
		return
	}

	select {
	case err := <-backend.done:
		_ = os.Remove(paths.stateFile)
		log.Printf("backend exited: %v", err)
	case <-uiDone:
		_ = os.Remove(paths.stateFile)
		log.Printf("web ui session closed")
	case <-sig:
		_ = os.Remove(paths.stateFile)
	}
}

func shutdownWeb(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func resetRuntime(paths launcherPaths) error {
	log.Printf("reset root=%s", paths.rootDir)
	return stopRuntime(paths)
}

func stopRuntime(paths launcherPaths) error {
	state, err := readState(paths)
	if err == nil {
		if state.LauncherPID != os.Getpid() {
			stopPID(state.LauncherPID)
		}
		stopPID(state.BackendPID)
	}
	if err := stopRuntimeProcessesByPath(paths); err != nil {
		log.Printf("runtime process sweep failed: %v", err)
	}
	waitForURLDown(webURL, 12*time.Second)
	waitForURLDown(backendHealthURL, 12*time.Second)
	_ = os.Remove(paths.stateFile)
	return nil
}

func stopRuntimeProcessesByPath(paths launcherPaths) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	bundleDir := filepath.Join(paths.homeDir, "server-bundle")
	script := fmt.Sprintf(`
$ErrorActionPreference = 'SilentlyContinue'
$currentPid = %d
$launcherPath = [System.IO.Path]::GetFullPath(%s).ToLowerInvariant()
$serverPath = [System.IO.Path]::GetFullPath(%s).ToLowerInvariant()
$backendPath = [System.IO.Path]::GetFullPath(%s).ToLowerInvariant()
$bundleDir = [System.IO.Path]::GetFullPath(%s).TrimEnd([char]92).ToLowerInvariant()
Get-CimInstance Win32_Process | Where-Object {
  if ($_.ProcessId -eq $currentPid) { return $false }
  $exe = if ($_.ExecutablePath) { $_.ExecutablePath.ToLowerInvariant() } else { '' }
  $cmd = if ($_.CommandLine) { $_.CommandLine.ToLowerInvariant() } else { '' }
  (
    ($_.Name -ieq 'AVmatrixLauncher.exe' -and $exe -eq $launcherPath) -or
    ($_.Name -ieq 'avmatrix-server.exe' -and $exe -eq $serverPath) -or
    ($_.Name -ieq 'avmatrix.exe' -and (
      $exe -eq $backendPath -or
      $exe.StartsWith($bundleDir) -or
      $cmd.Contains($bundleDir)
    ))
  )
} | ForEach-Object {
  Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
}
`, os.Getpid(), psQuote(paths.exePath), psQuote(paths.serverExe), psQuote(paths.backendExe), psQuote(bundleDir))
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = hiddenProcAttr()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("powershell process sweep: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func psQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func readState(paths launcherPaths) (launcherState, error) {
	var state launcherState
	data, err := os.ReadFile(paths.stateFile)
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	return state, nil
}

func writeState(paths launcherPaths, status string, backendPID int) {
	state := launcherState{
		RootDir:     paths.rootDir,
		LauncherPID: os.Getpid(),
		BackendPID:  backendPID,
		UpdatedAt:   time.Now(),
		Status:      status,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Printf("marshal state: %v", err)
		return
	}
	if err := os.WriteFile(paths.stateFile, data, 0o644); err != nil {
		log.Printf("write state: %v", err)
	}
}

func urlReady(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

func waitForURL(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if urlReady(url) {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func waitForURLDown(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !urlReady(url) {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return !urlReady(url)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
		return err == nil && strings.Contains(string(out), fmt.Sprintf("\"%d\"", pid))
	}
	proc, err := os.FindProcess(pid)
	return err == nil && proc.Signal(syscall.Signal(0)) == nil
}

func stopPID(pid int) {
	if pid <= 0 || pid == os.Getpid() || !processAlive(pid) {
		return
	}
	if runtime.GOOS == "windows" {
		soft := exec.Command("taskkill", "/PID", fmt.Sprint(pid), "/T")
		soft.SysProcAttr = hiddenProcAttr()
		_ = soft.Run()
		if waitForPIDExit(pid, 8*time.Second) {
			return
		}
		force := exec.Command("taskkill", "/PID", fmt.Sprint(pid), "/T", "/F")
		force.SysProcAttr = hiddenProcAttr()
		_ = force.Run()
		waitForPIDExit(pid, 5*time.Second)
		return
	}
	proc, err := os.FindProcess(pid)
	if err == nil {
		_ = proc.Signal(os.Interrupt)
	}
	if waitForPIDExit(pid, 8*time.Second) {
		return
	}
	if err == nil {
		_ = proc.Kill()
	}
	waitForPIDExit(pid, 5*time.Second)
}

func waitForPIDExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return !processAlive(pid)
}

func registerProtocol(paths launcherPaths) error {
	if runtime.GOOS != "windows" {
		return errors.New("protocol registration is Windows-only")
	}
	command := fmt.Sprintf(`"%s" "%%1"`, paths.exePath)
	key := `HKCU\Software\Classes\avmatrix`
	commands := [][]string{
		{"add", key, "/ve", "/d", "URL:AVmatrix Launcher", "/f"},
		{"add", key, "/v", "URL Protocol", "/d", "", "/f"},
		{"add", key + `\shell\open\command`, "/ve", "/d", command, "/f"},
	}
	for _, args := range commands {
		cmd := exec.Command("reg", args...)
		cmd.SysProcAttr = hiddenProcAttr()
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("reg %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func openBrowser(url string) error {
	if os.Getenv("AVMATRIX_LAUNCHER_NO_BROWSER") == "1" {
		log.Printf("browser open suppressed by AVMATRIX_LAUNCHER_NO_BROWSER")
		return nil
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		cmd.SysProcAttr = hiddenProcAttr()
		return cmd.Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}

func attachLog(paths launcherPaths, cmd *exec.Cmd, fileName string) {
	if err := os.MkdirAll(paths.logDir, 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(paths.logDir, fileName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	cmd.Stdout = file
	cmd.Stderr = file
}

func hiddenProcAttr() *syscall.SysProcAttr {
	if runtime.GOOS != "windows" {
		return &syscall.SysProcAttr{}
	}
	const createNoWindow = 0x08000000
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
