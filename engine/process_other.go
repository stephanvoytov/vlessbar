//go:build !darwin && !windows

package main

import "errors"

// Stubs so the package (and its tests) build on non-darwin hosts. VLessBar
// only runs on macOS; these are never used at runtime.

func startXray(v *Vless, allowLAN bool) error { return errors.New("tunnel start is only supported on macOS") }
func stopXray()                              {}
func xrayRunning() bool                       { return false }
