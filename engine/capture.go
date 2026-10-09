package main

import (
	"bytes"
	"io"
	"os"
)

// capture runs fn while redirecting os.Stdout into a buffer and returns what was
// printed. The GUI uses it to show engine command output inside the window.
func capture(fn func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", fn()
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String(), runErr
}
