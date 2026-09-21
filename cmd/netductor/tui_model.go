package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.screen == screenWizard {
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			m.width, m.height = ws.Width, ws.Height
			return m, nil
		}
		return m.updateWizard(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.helpY = msg.Height - 2
		return m, nil
	case tea.MouseMsg:
		if msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		if msg.Action != tea.MouseActionPress && msg.Action != tea.MouseActionRelease {
			return m, nil
		}
		return m.handleMouse(msg.X, msg.Y)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			m.result = tuiResult{action: "quit", mode: m.mode}
			return m, tea.Quit
		case "l", "L", "ctrl+l":
			return m.toggleLang()
		case "tab":
			order := []string{tabWizard, tabTools, tabOps, tabSettings, tabMode}
			for i, tname := range order {
				if tname == m.tab {
					m.tab = order[(i+1)%len(order)]
					break
				}
			}
			m.cursor = 0
			if m.tab == tabWizard {
				m.screen = screenMenu
				m.wizStep = wizStepTarget
				return m, nil
			}
			if m.tab == tabMode {
				m.screen = screenMode
			} else {
				m.screen = screenMenu
			}
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.currentEntries())-1 {
				m.cursor++
			}
			return m, nil
		case "esc", "q":
			if m.screen == screenOutput {
				m.screen = screenMenu
				m.output = ""
				return m, nil
			}
			if m.tab != tabMode && m.screen == screenMenu {
				m.tab = tabMode
				m.screen = screenMode
				m.cursor = 0
				return m, nil
			}
			m.tab = tabTools
			m.screen = screenMenu
			m.cursor = 0
			return m, nil
		case "enter":
			if m.screen == screenOutput {
				m.screen = screenMenu
				m.output = ""
				return m, nil
			}
			return m.activateCursor()
		}
	}
	return m, nil
}

func (m model) toggleLang() (tea.Model, tea.Cmd) {
	if m.lang == langRU {
		m.lang = langEN
		m.langPref = "en"
	} else {
		m.lang = langRU
		m.langPref = "ru"
	}
	return m, nil
}

func (m model) activateCursor() (tea.Model, tea.Cmd) {
	ents := m.currentEntries()
	if m.cursor < 0 || m.cursor >= len(ents) {
		return m, nil
	}
	id := ents[m.cursor].ID
	if m.screen == screenMode || m.tab == tabMode {
		m.mode = runMode(id)
		m.tab = tabTools
		m.screen = screenMenu
		m.cursor = 0
		return m, nil
	}
	if m.tab == tabWizard {
		m.wizTarget = wizTarget(id)
		m.wizBuildFields()
		m.wizStep = wizStepFields
		m.screen = screenWizard
		return m, nil
	}
	return m.handleAction(id)
}

func (m model) handleAction(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "quit":
		m.quitting = true
		m.result = tuiResult{action: "quit", mode: m.mode}
		return m, tea.Quit
	case "change-mode":
		m.tab = tabMode
		m.screen = screenMode
		m.cursor = 0
		return m, nil
	case "remote-set":
		m.wizTarget = "remote"
		m.wizFields = []wizField{
			{Key: "host", Label: "VPS host / IP", Value: m.remoteHost, Placeholder: "vps.example.com"},
			{Key: "user", Label: "SSH user", Value: orDefault(m.remoteUser, "root")},
		}
		m.wizFieldIdx = 0
		m.wizInput = m.wizFields[0].Value
		m.wizStep = wizStepFields
		m.screen = screenWizard
		return m, nil
	case "remote-clear":
		m.remoteHost, m.remoteUser = "", "root"
		m.output = "remote cleared — commands run locally"
		m.screen = screenOutput
		return m, nil
	case "fleet-status":
		m.showCmd("fleet", "status")
	case "secondary-sync", "fleet-sync", "relay-sync":
		m.showCmd("secondary", "sync")
	case "apply-lampac":
		m.showCmd("install", "lampac")
	case "disable-legacy":
		m.showCmd("fleet", "disable-legacy")
	case "status":
		if m.hasRemote() {
			m.showCmd("status")
		} else {
			m.output = capture(func() { runStatus() })
			m.screen = screenOutput
		}
	case "mtls-list":
		m.showCmd("mtls", "list")
	case "doctor":
		if m.hasRemote() {
			m.showCmd("doctor")
		} else {
			m.output = capture(func() { _ = runDoctorNative() })
			m.screen = screenOutput
		}
	case "vpn-list":
		m.showCmd("vpn", "list")
	case "secondary-status":
		m.showCmd("secondary", "status")
	case "nodes-list":
		m.showCmd("nodes", "list")
	case "edge-list":
		m.showCmd("edge", "list")
	case "edge-pending":
		m.showCmd("edge", "pending")
	case "edge-recovery":
		m.showCmd("edge", "recovery")
	case "edge-register":
		m.startActionForm("edge-register")
	case "probe":
		if m.hasRemote() {
			m.showCmd("probe")
		} else {
			m.output = capture(func() { runProbe(nil) })
			m.screen = screenOutput
		}
	case "nvr-status":
		m.showCmd("nvr", "status")
	case "nvr-go2rtc":
		m.showCmd("nvr", "go2rtc")
	case "backup-now":
		m.showCmd("backup", "now")
	case "audit-tail":
		m.showCmd("audit", "tail")
	case "install":
		m.showCmd("install")
		return m, nil
	case "prepare":
		m.showCmd("install", "--prepare")
		return m, nil
	case "vpn-add":
		m.startActionForm("vpn-add")
		return m, nil
	case "build":
		m.startActionForm("build")
		return m, nil
	case "owrt-install":
		m.startActionForm("owrt-install")
		return m, nil
	case "cfg-remote":
		m.startActionForm("cfg-remote")
		return m, nil
	case "cfg-lang":
		return m.toggleLang()
	case "cfg-save":
		if err := saveTUISettings(m.snapshotSettings()); err != nil {
			m.output = "save failed: " + err.Error()
		} else {
			m.output = "saved " + tuiConfigPath()
		}
		m.screen = screenOutput
		return m, nil
	case "cfg-clear-remote":
		m.remoteHost, m.remoteKey, m.remotePassword = "", "", ""
		m.remoteUser = "root"
		_ = saveTUISettings(m.snapshotSettings())
		m.output = "remote cleared"
		m.screen = screenOutput
		return m, nil
	case "hostname", "site-wizard", "mt-manage", "sites-list", "sni-live", "ssh-hosts", "session", "backup-peer", "vpn-sub", "vpn-rename":
		m.output = "CLI: netductor " + id + " — or use Wizard tab"
		m.screen = screenOutput
		return m, nil
	default:
		m.output = "unknown action: " + id
		m.screen = screenOutput
	}
	return m, nil
}
