//go:build !darwin && !windows

package main

import "errors"

// launchGUI has no implementation outside macOS/Windows.
func launchGUI() error { return errors.New("GUI is only available on macOS and Windows; use the CLI") }
