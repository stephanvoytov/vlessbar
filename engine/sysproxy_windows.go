//go:build windows

package main

import (
	"os/exec"
	"strconv"
	"syscall"
)

// Windows system proxy via the WinINET "Internet Settings" registry key, then a
// WinINET refresh so already-running apps notice the change. Stdlib only.
const inetSettingsKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

func regSet(args ...string) {
	_ = exec.Command("reg", args...).Run()
}

// proxyOn points the system HTTP/HTTPS proxy at the local Xray HTTP inbound.
func proxyOn() error {
	server := "127.0.0.1:" + strconv.Itoa(httpPort)
	regSet("add", inetSettingsKey, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f")
	regSet("add", inetSettingsKey, "/v", "ProxyServer", "/t", "REG_SZ", "/d", server, "/f")
	regSet("add", inetSettingsKey, "/v", "ProxyOverride", "/t", "REG_SZ", "/d", "localhost;127.*;<local>", "/f")
	refreshInet()
	return nil
}

// proxyOff disables the system proxy.
func proxyOff() error {
	regSet("add", inetSettingsKey, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f")
	refreshInet()
	return nil
}

func refreshInet() {
	wininet := syscall.NewLazyDLL("wininet.dll")
	set := wininet.NewProc("InternetSetOptionW")
	const (
		internetOptionSettingsChanged = 39
		internetOptionRefresh         = 37
	)
	_, _, _ = set.Call(0, internetOptionSettingsChanged, 0, 0)
	_, _, _ = set.Call(0, internetOptionRefresh, 0, 0)
}
