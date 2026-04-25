package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

func main() {
	bundleDir, rootDir, logFile, err := resolveRuntime()
	if err != nil {
		fatal(err)
	}
	if logFile != nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	nodePath := filepath.Join(bundleDir, "node.exe")
	if _, err := os.Stat(nodePath); err != nil {
		fatal(fmt.Errorf("bundled node.exe missing: %s", nodePath))
	}

	cliPath := filepath.Join(rootDir, "avmatrix", "dist", "cli", "index.js")
	if _, err := os.Stat(cliPath); err != nil {
		fatal(fmt.Errorf("AVmatrix CLI build missing: %s", cliPath))
	}

	cmd := exec.Command(nodePath, cliPath, "serve")
	cmd.Dir = filepath.Join(rootDir, "avmatrix")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = hiddenProcAttr()
	log.Printf("starting backend: %s %s serve", nodePath, cliPath)
	if err := cmd.Run(); err != nil {
		fatal(err)
	}
}

func resolveRuntime() (string, string, *os.File, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", "", nil, err
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return "", "", nil, err
	}
	bundleDir := filepath.Dir(exePath)
	launcherDir := filepath.Dir(bundleDir)
	rootDir := filepath.Dir(launcherDir)
	logDir := filepath.Join(launcherDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", "", nil, err
	}
	logFile, err := os.OpenFile(filepath.Join(logDir, "server-wrapper.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", "", nil, err
	}
	return bundleDir, rootDir, logFile, nil
}

func hiddenProcAttr() *syscall.SysProcAttr {
	if runtime.GOOS != "windows" {
		return &syscall.SysProcAttr{}
	}
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}

func fatal(err error) {
	log.Printf("fatal: %v", err)
	os.Exit(1)
}
