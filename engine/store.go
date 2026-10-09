package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Server is one parsed share link. Manual servers are added by the user
// (e.g. a legacy vless:// link) and are kept across subscription refreshes.
type Server struct {
	Name    string `json:"name"`
	URI     string `json:"uri"`
	Address string `json:"address"`
	Port    int    `json:"port"`
	Network string `json:"network"`
	Manual  bool   `json:"manual,omitempty"`
}

// State is the persisted client state.
type State struct {
	Hwid        string   `json:"hwid"`
	SubURL      string   `json:"sub_url"`
	AllowLAN    bool     `json:"allow_lan"`
	Servers     []Server `json:"servers"`
	Selected    int      `json:"selected"`
	LastUpdate  string   `json:"last_update"`
	UserInfo    string   `json:"user_info"`
	Announce    string   `json:"announce"`
	ProfileURL  string   `json:"profile_url"`
	HwidActive  bool     `json:"hwid_active"`
	HwidLimit   bool     `json:"hwid_limit"`
	HwidNotSupp bool     `json:"hwid_not_supported"`
	Warning     string   `json:"warning"`
}

func statePath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

func loadState() (*State, error) {
	p, err := statePath()
	if err != nil {
		return nil, err
	}
	s := &State{}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	return s, nil
}

func saveState(s *State) error {
	p, err := statePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func pidPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "xray.pid"), nil
}
