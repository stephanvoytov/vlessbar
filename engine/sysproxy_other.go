//go:build !darwin && !windows

package main

// System-proxy integration is implemented per-OS; other platforms are no-ops.
func proxyOn() error  { return nil }
func proxyOff() error { return nil }
