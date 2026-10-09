package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// launchGUI is called when the engine is started with no arguments (i.e. when
// the user double-clicks VLessBar.app). It runs the bundled AppleScript UI,
// passing it the Resources dir and the engine path.
func launchGUI() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	resDir := filepath.Join(filepath.Dir(exe), "..", "Resources")
	script := filepath.Join(resDir, "ui.applescript")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("gui not available (not running from .app)")
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	cmd := exec.Command("osascript", script, resDir, exe, bundle)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// cmdGuiState prints a one-line ASCII status for the AppleScript UI.
func cmdGuiState() error {
	s, err := loadState()
	if err != nil {
		return err
	}
	state := "OFF"
	if xrayRunning() {
		state = "ON"
	}
	name := "(no server)"
	if s.Selected >= 0 && s.Selected < len(s.Servers) {
		name = s.Servers[s.Selected].Name
	}
	fmt.Printf("%s | %s | servers: %d\n", state, name, len(s.Servers))
	return nil
}

// cmdGuiServers prints "index<TAB>name" lines for the AppleScript UI.
func cmdGuiServers() error {
	s, err := loadState()
	if err != nil {
		return err
	}
	for i, srv := range s.Servers {
		fmt.Printf("%d\t%s\n", i, srv.Name)
	}
	return nil
}
