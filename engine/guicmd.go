package main

import "fmt"

// cmdGuiState prints a one-line status for the GUI (used by both the macOS
// AppleScript UI and the Windows Win32 UI).
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
	fmt.Printf("%s | %s | servers: %d | lan: %s\n", state, name, len(s.Servers), onOff(s.AllowLAN))
	return nil
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// cmdGuiServers prints "index<TAB>name" lines for the GUI.
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

// cmdVersions prints "app=..." and "core=..." (used by the macOS menu UI).
func cmdVersions() error {
	fmt.Println("app=" + version)
	fmt.Println("core=" + xrayVersion())
	return nil
}
