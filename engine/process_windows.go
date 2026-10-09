//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// Win32 process helpers (stdlib syscall only, no external deps).
var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess        = kernel32.NewProc("OpenProcess")
	procGetExitCodeProcess = kernel32.NewProc("GetExitCodeProcess")
	procCloseHandle        = kernel32.NewProc("CloseHandle")
)

const (
	processQueryLimitedInformation = 0x1000
	stillActive                    = 259
	createNewProcessGroup          = 0x00000200
)

// startXray writes the config and launches xray in the background.
func startXray(v *Vless, allowLAN bool) error {
	dir, err := dataDir()
	if err != nil {
		return err
	}
	bin, err := xrayBinary()
	if err != nil {
		return err
	}
	cfgPath, err := writeXrayConfig(dir, v, allowLAN)
	if err != nil {
		return err
	}
	logf, err := os.OpenFile(filepath.Join(dir, "xray.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "run", "-c", cfgPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := cmd.Start(); err != nil {
		return err
	}
	p, _ := pidPath()
	_ = os.WriteFile(p, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600)
	// Detach: the child keeps running after this (often short-lived) process exits.
	go func() { _ = cmd.Wait() }()
	return nil
}

func readPid() (int, error) {
	p, err := pidPath()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func stopXray() {
	if pid, err := readPid(); err == nil && pid > 0 {
		// /T kills the whole process tree, /F forces termination.
		_ = exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
	}
	if p, err := pidPath(); err == nil {
		_ = os.Remove(p)
	}
}

func xrayRunning() bool {
	pid, err := readPid()
	if err != nil || pid <= 0 {
		return false
	}
	h, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return false
	}
	defer procCloseHandle.Call(h)
	var code uint32
	r, _, _ := procGetExitCodeProcess.Call(h, uintptr(unsafe.Pointer(&code)))
	if r == 0 {
		return false
	}
	return code == stillActive
}
