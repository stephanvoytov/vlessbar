package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// launchGUI is called when the engine is started with no arguments (i.e. when
// the user double-clicks VLessBar.app). It runs the bundled AppleScript UI,
// passing it the Resources dir, the engine path and the bundle path.
//
// Any failure is reported to ~/.vlessbar/gui-error.log and shown on screen, so
// the app never "does nothing" silently.
func launchGUI() error {
	exe, err := os.Executable()
	if err != nil {
		return reportGUIFailure(fmt.Errorf("cannot locate executable: %w", err))
	}
	resDir := filepath.Join(filepath.Dir(exe), "..", "Resources")
	script := filepath.Join(resDir, "ui.applescript")
	if _, err := os.Stat(script); err != nil {
		return reportGUIFailure(fmt.Errorf("ui.applescript not found at %s (not running from .app?): %w", script, err))
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))

	cmd := exec.Command("osascript", script, resDir, exe, bundle)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return reportGUIFailure(fmt.Errorf("osascript failed: %w\n--- osascript output ---\n%s", err, out.String()))
	}
	return nil
}

// reportGUIFailure records the error to a log file and tries to show it in a
// dialog so the user sees what went wrong instead of a dead click.
func reportGUIFailure(err error) error {
	msg := err.Error()
	if dir, e := dataDir(); e == nil {
		_ = os.WriteFile(filepath.Join(dir, "gui-error.log"), []byte(msg+"\n"), 0o600)
	}
	dialogText := "VLessBar не смог открыть интерфейс.\n\n" + msg
	_ = exec.Command("osascript", "-e",
		`tell application "System Events" to display dialog `+appleScriptString(dialogText)+
			` with title "VLessBar" buttons {"OK"} default button 1 with icon caution`).Run()
	return err
}

// appleScriptString renders s as an AppleScript string literal.
func appleScriptString(s string) string {
	out := make([]rune, 0, len(s)+2)
	out = append(out, '"')
	for _, r := range s {
		switch r {
		case '\\':
			out = append(out, '\\', '\\')
		case '"':
			out = append(out, '\\', '"')
		default:
			out = append(out, r)
		}
	}
	out = append(out, '"')
	return string(out)
}

// cmdGuiState prints a one-line status for the AppleScript UI.
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
