package main

import (
	"crypto/rand"
	"fmt"
	"regexp"
)

// hwidRe matches the Remnawave v3.0.0 HWID contract: 10..64 of [A-Za-z0-9=-].
var hwidRe = regexp.MustCompile(`^[a-zA-Z0-9=-]{10,64}$`)

// newHwid returns a random RFC-4122 v4 UUID string (36 chars, dash-separated),
// which satisfies the HWID charset/length contract.
func newHwid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ensureHwid returns the stored HWID, generating and persisting one on first run.
// It never rotates an existing valid HWID.
func ensureHwid(s *State) (string, error) {
	if s.Hwid != "" && hwidRe.MatchString(s.Hwid) {
		return s.Hwid, nil
	}
	s.Hwid = newHwid()
	return s.Hwid, saveState(s)
}
