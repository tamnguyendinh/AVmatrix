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
	"syscall"
	"time"
)

const (
	backendHealthURL = "http://localhost:4747/api/info"
	webURL           = "http://127.0.0.1:5173"
)

type launcherPaths struct {
	exePath   string
	rootDir   string
	homeDir   string
	logDir    string
	webDist   string
	serverExe string
	stateFile string
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
		exePath:   exePath,
		rootDir:   rootDir,
		homeDir:   homeDir,
		logDir:    filepath.Join(homeDir, "logs"),
		webDist:   filepath.Join(homeDir, "web-dist"),
		serverExe: filepath.Join(homeDir, "server-bundle", "avmatrix-server.exe"),
		stateFile: stateFile,
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

	webServer := &http.Server{
		Addr:              "127.0.0.1:5173",
		Handler:           staticHandler(paths.webDist),
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

	waitForExit(paths, backend)
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			rel = "index.html"
		}
		target := filepath.Join(webDist, rel)
		if stat, err := os.Stat(target); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, target)
			return
		}
		http.ServeFile(w, r, filepath.Join(webDist, "index.html"))
	})
}

func waitForExit(paths launcherPaths, backend backendProcess) {
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	if backend.done == nil {
		<-sig
		_ = os.Remove(paths.stateFile)
		return
	}

	select {
	case err := <-backend.done:
		_ = os.Remove(paths.stateFile)
		log.Printf("backend exited: %v", err)
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
	if err := stopRuntime(paths); err != nil {
		return err
	}
	return startRuntime(paths)
}

func stopRuntime(paths launcherPaths) error {
	state, err := readState(paths)
	if err == nil {
		if state.LauncherPID != os.Getpid() {
			stopPID(state.LauncherPID)
		}
		stopPID(state.BackendPID)
	}
	waitForURLDown(webURL, 12*time.Second)
	waitForURLDown(backendHealthURL, 12*time.Second)
	_ = os.Remove(paths.stateFile)
	return nil
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
