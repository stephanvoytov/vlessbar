//go:build darwin

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// startXray writes the config and launches xray in the background.
func startXray(v *Vless) error {
	dir, err := dataDir()
	if err != nil {
		return err
	}
	bin, err := xrayBinary()
	if err != nil {
		return err
	}
	cfgPath, err := writeXrayConfig(dir, v)
	if err != nil {
		return err
	}
	logf, err := os.OpenFile(filepath.Join(dir, "xray.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "run", "-c", cfgPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := cmd.Start(); err != nil {
		return err
	}
	p, _ := pidPath()
	_ = os.WriteFile(p, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600)
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
		_ = syscall.Kill(pid, syscall.SIGTERM)
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
	return syscall.Kill(pid, 0) == nil
}
