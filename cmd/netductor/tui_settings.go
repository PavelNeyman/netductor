package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// TUI settings persisted under ~/.config/netductor/tui.json

type tuiSettings struct {
	RemoteHost     string `json:"remote_host"`
	RemoteUser     string `json:"remote_user"`
	RemoteKey      string `json:"remote_key"`      // path to private key
	RemotePassword string `json:"remote_password"` // optional; prefer key
	Lang           string `json:"lang"`            // ru|en|auto
}

func tuiConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "netductor", "tui.json")
}

func loadTUISettings() tuiSettings {
	var s tuiSettings
	s.RemoteUser = "root"
	path := tuiConfigPath()
	if path == "" {
		return s
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	if s.RemoteUser == "" {
		s.RemoteUser = "root"
	}
	return s
}

func saveTUISettings(s tuiSettings) error {
	path := tuiConfigPath()
	if path == "" {
		return os.ErrInvalid
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func (m *model) applySettings(s tuiSettings) {
	m.remoteHost = s.RemoteHost
	m.remoteUser = s.RemoteUser
	if m.remoteUser == "" {
		m.remoteUser = "root"
	}
	m.remoteKey = s.RemoteKey
	m.remotePassword = s.RemotePassword
	switch s.Lang {
	case "ru":
		m.lang = langRU
	case "en":
		m.lang = langEN
	}
}

func (m model) snapshotSettings() tuiSettings {
	lang := "auto"
	if m.lang == langRU {
		lang = "ru"
	} else if m.lang == langEN {
		lang = "en"
	}
	return tuiSettings{
		RemoteHost: m.remoteHost, RemoteUser: orDefault(m.remoteUser, "root"),
		RemoteKey: m.remoteKey, RemotePassword: m.remotePassword, Lang: lang,
	}
}
