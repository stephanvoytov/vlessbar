//go:build darwin

package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// networkServices lists enabled macOS network services (skips the header line
// and disabled services, which are prefixed with '*').
func networkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, err
	}
	var svcs []string
	for i, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if i == 0 || line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		svcs = append(svcs, line)
	}
	return svcs, nil
}

// proxyOn points the system HTTP/HTTPS/SOCKS proxy at the local Xray inbounds.
func proxyOn() error {
	svcs, err := networkServices()
	if err != nil {
		return err
	}
	for _, s := range svcs {
		run_cmd("networksetup", "-setwebproxy", s, "127.0.0.1", strconv.Itoa(httpPort))
		run_cmd("networksetup", "-setsecurewebproxy", s, "127.0.0.1", strconv.Itoa(httpPort))
		run_cmd("networksetup", "-setsocksfirewallproxy", s, "127.0.0.1", strconv.Itoa(socksPort))
	}
	return nil
}

// proxyOff disables all system proxies.
func proxyOff() error {
	svcs, err := networkServices()
	if err != nil {
		return err
	}
	for _, s := range svcs {
		run_cmd("networksetup", "-setwebproxystate", s, "off")
		run_cmd("networksetup", "-setsecurewebproxystate", s, "off")
		run_cmd("networksetup", "-setsocksfirewallproxystate", s, "off")
	}
	return nil
}

func run_cmd(name string, args ...string) {
	_ = exec.Command(name, args...).Run()
}
