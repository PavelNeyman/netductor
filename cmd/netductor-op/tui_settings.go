package main

import (
	"strings"

	"github.com/PavelNeyman/netductor/internal/opconfig"
)

// Operator settings: ~/.config/netductor/operator.conf (TUI + Web via opconfig).

type tuiSettings struct {
	RemoteHost     string
	RemoteUser     string
	RemoteKey      string
	RemotePassword string
	Lang           string
	SecondaryHost  string
	LastEdgeID     string
	Mode           string
}

func tuiConfigPath() string { return opconfig.Path() }

func fromOp(s opconfig.Settings) tuiSettings {
	return tuiSettings{
		RemoteHost: s.RemoteHost, RemoteUser: s.RemoteUser, RemoteKey: s.RemoteKey,
		RemotePassword: s.RemotePassword, Lang: s.Lang, SecondaryHost: s.SecondaryHost,
		LastEdgeID: s.LastEdgeID, Mode: s.Mode,
	}
}

func toOp(s tuiSettings) opconfig.Settings {
	return opconfig.Settings{
		RemoteHost: s.RemoteHost, RemoteUser: s.RemoteUser, RemoteKey: s.RemoteKey,
		RemotePassword: s.RemotePassword, Lang: s.Lang, SecondaryHost: s.SecondaryHost,
		LastEdgeID: s.LastEdgeID, Mode: s.Mode,
	}
}

func loadTUISettings() tuiSettings {
	s := fromOp(opconfig.Load())
	if s.RemoteUser == "" {
		s.RemoteUser = "root"
	}
	if s.Lang == "" {
		s.Lang = "auto"
	}
	return s
}

func saveTUISettings(s tuiSettings) error {
	return opconfig.Save(toOp(s))
}

func (m *model) applySettings(s tuiSettings) {
	m.remoteHost = s.RemoteHost
	if s.Mode != "" {
		m.mode = runMode(s.Mode)
	}
	m.remoteUser = s.RemoteUser
	if m.remoteUser == "" {
		m.remoteUser = "root"
	}
	m.remoteKey = s.RemoteKey
	m.remotePassword = s.RemotePassword
	m.langPref = s.Lang
	switch strings.ToLower(strings.TrimSpace(s.Lang)) {
	case "ru":
		m.lang = langRU
	case "en":
		m.lang = langEN
	default:
		m.lang = detectLang()
		m.langPref = "auto"
	}
}

func (m model) snapshotSettings() tuiSettings {
	lang := m.langPref
	if lang == "" {
		lang = "auto"
	}
	if lang != "auto" {
		if m.lang == langRU {
			lang = "ru"
		} else {
			lang = "en"
		}
	}
	prev := loadTUISettings()
	return tuiSettings{
		RemoteHost: m.remoteHost, RemoteUser: orDefault(m.remoteUser, "root"),
		RemoteKey: m.remoteKey, RemotePassword: m.remotePassword, Lang: lang,
		Mode: string(m.mode), SecondaryHost: prev.SecondaryHost, LastEdgeID: prev.LastEdgeID,
	}
}
