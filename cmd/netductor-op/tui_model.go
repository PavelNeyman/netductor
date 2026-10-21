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
	if m.screen == screenCatalog {
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			m.width, m.height = ws.Width, ws.Height
			return m, nil
		}
		return m.updateCatalog(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.helpY = msg.Height - 2
		return m, nil
	case tea.MouseMsg:
		// Wheel scroll log inside wizard run
		if m.screen == screenWizard && m.wizStep == wizStepRun && m.wizViewMode != "steps" {
			if msg.Button == tea.MouseButtonWheelUp {
				m.wizLogOffset++
				m.clampWizLogOffset(20)
				return m, nil
			}
			if msg.Button == tea.MouseButtonWheelDown {
				if m.wizLogOffset > 0 {
					m.wizLogOffset--
				}
				return m, nil
			}
		}
		if m.screen == screenOutput {
			if msg.Button == tea.MouseButtonWheelUp {
				m.outOffset++
				return m, nil
			}
			if msg.Button == tea.MouseButtonWheelDown {
				if m.outOffset > 0 {
					m.outOffset--
				}
				return m, nil
			}
		}
		if m.screen == screenCatalog {
			if msg.Button == tea.MouseButtonWheelUp {
				if m.catLevel == 0 {
					if m.catGroupIdx > 0 {
						m.catGroupIdx--
					}
				} else if m.catActionIdx > 0 {
					m.catActionIdx--
				}
				return m, nil
			}
			if msg.Button == tea.MouseButtonWheelDown {
				if m.catLevel == 0 {
					if m.catGroupIdx < len(m.catalogGroups())-1 {
						m.catGroupIdx++
					}
				} else {
					acts := m.catalogActions()
					if m.catActionIdx < len(acts)-1 {
						m.catActionIdx++
					}
				}
				return m, nil
			}
		}
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
		case "pgup", "ctrl+u":
			if m.screen == screenOutput {
				m.outOffset += 10
				return m, nil
			}
		case "pgdown", "ctrl+d":
			if m.screen == screenOutput {
				m.outOffset -= 10
				if m.outOffset < 0 {
					m.outOffset = 0
				}
				return m, nil
			}
		case "home":
			if m.screen == screenOutput {
				// jump to top of log
				lines := len(m.output)
				if lines > 0 {
					m.outOffset = 999999 // clamped in render
				}
				return m, nil
			}
		case "end":
			if m.screen == screenOutput {
				m.outOffset = 0
				return m, nil
			}
		case "esc", "q":
			if m.screen == screenOutput {
				m.screen = screenMenu
				m.output = ""
				m.outOffset = 0
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
		// Persist for next launch
		s := m.snapshotSettings()
		s.Mode = string(m.mode)
		_ = saveTUISettings(s)
		mt, md := describeModeLang(m.mode, m.lang)
		if m.lang == langRU {
			m.output = "✓ Режим: " + mt + "\n" + md + "\n\nДальше: вкладки Мастер / Инструменты / Операции для этого режима."
		} else {
			m.output = "✓ Mode: " + mt + "\n" + md + "\n\nNext: Wizard / Tools / Ops tabs apply to this mode."
		}
		m.tab = tabTools
		m.screen = screenOutput
		m.outOffset = 0
		m.cursor = 0
		return m, nil
	}
	if m.tab == tabWizard {
		// In-TUI multi-step fields (no huh / no alt-screen exit)
		m.wizTarget = wizTarget(id)
		m.wizFields = wizBuildFields(id, &m)
		m.wizFieldIdx = 0
		m.wizInput = ""
		if len(m.wizFields) > 0 {
			m.wizInput = m.wizFields[0].Value
		}
		m.wizMsg = ""
		m.wizStep = wizStepFields
		if len(m.wizFields) == 0 {
			m.wizStep = wizStepConfirm
		}
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
		m.outOffset = 0
		return m, nil
	case "fleet-status":
		m.showCmd("fleet", "status")
	case "jobs", "jobs-list":
		m.output = listOpJobsText()
		m.screen = screenOutput
		m.outOffset = 0
		return m, nil
	case "jobs-running":
		m.output = listOpJobsTextRunning()
		m.screen = screenOutput
		m.outOffset = 0
		return m, nil
	case "jobs-cancel":
		m.tab = tabWizard
		m.wizTarget = "jobs-cancel"
		m.wizFields = []wizField{{Key: "job_id", Label: "Job ID", Value: "", Placeholder: "id from Jobs list"}}
		m.wizFieldIdx = 0
		m.wizInput = ""
		m.wizStep = wizStepFields
		m.screen = screenWizard
		return m, nil
	case "secondary-sync", "fleet-sync", "relay-sync":
		m.showCmd("secondary", "sync")
	case "apply-lampac":
		// Open multi-select add-ons instead of single lampac install
		m.tab = tabWizard
		m.wizTarget = "addons"
		m.wizFields = wizBuildFields("addons", &m)
		m.wizFieldIdx = 0
		m.wizInput = ""
		if len(m.wizFields) > 0 {
			m.wizInput = m.wizFields[0].Value
		}
		m.wizStep = wizStepFields
		m.screen = screenWizard
	case "disable-legacy":
		m.showCmd("fleet", "disable-legacy")
	case "catalog":
		m.openCatalog()
		return m, nil
	case "status":
		if m.hasRemote() {
			m.showCmd("status")
		} else {
			m.output = m.runNetductor("status")
			m.screen = screenOutput
		m.outOffset = 0
		}
	case "mtls-list":
		m.showCmd("mtls", "list")
	case "mtls-rollover":
		m.showCmd("mtls", "rollover", "status")
	case "doctor":
		if m.hasRemote() {
			m.showCmd("doctor")
		} else {
			m.output = m.runNetductor("doctor")
			m.screen = screenOutput
		m.outOffset = 0
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
	case "edge-storage-list", "edge-storage-backup", "edge-storage-restore", "edge-storage-mute", "edge-storage-pack":
		m.startActionForm(id)
	case "edge-register":
		m.startActionForm("edge-register")
	case "probe":
		if m.hasRemote() {
			m.showCmd("probe")
		} else {
			m.output = m.runNetductor("probe")
			m.screen = screenOutput
		m.outOffset = 0
		}
	case "nvr-status":
		m.showCmd("nvr", "status")
	case "nvr-go2rtc":
		m.showCmd("nvr", "go2rtc")
	case "nvr-cameras":
		m.showCmd("nvr", "cameras", "list")
	case "nvr-storage":
		m.showCmd("nvr", "storage")
	case "git-repos":
		m.showCmd("git", "list")
	case "git-pipelines":
		m.showCmd("git", "pipelines")
	case "registry-status":
		m.showCmd("registry", "status")
	case "domain-show":
		m.showCmd("domain", "show")
	case "addons-status":
		m.showCmd("addons", "list")
	case "vpn-refresh":
		m.showCmd("vpn", "refresh-links")
	case "dns-list":
		m.showCmd("dns", "list")
	case "dns-reload":
		m.showCmd("dns", "reload")
	case "dns-on":
		m.startActionForm("dns-on")
		return m, nil
	case "dns-off":
		m.startActionForm("dns-off")
		return m, nil
	case "backup-now":
		m.showCmd("backup", "now")
	case "backup-list":
		m.showCmd("backup", "list")
	case "backup-schedule":
		m.showCmd("backup", "schedule")
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
		m.outOffset = 0
		return m, nil
	case "cfg-clear-remote":
		m.remoteHost, m.remoteKey, m.remotePassword = "", "", ""
		m.showSecrets = false
		m.remoteUser = "root"
		_ = saveTUISettings(m.snapshotSettings())
		m.output = "remote cleared"
		m.screen = screenOutput
		m.outOffset = 0
		return m, nil
	case "wizard", "addons":
		m.tab = tabWizard
		m.screen = screenMenu
		m.cursor = 0
		if id == "addons" {
			m.wizTarget = "addons"
			m.wizFields = wizBuildFields("addons", &m)
			m.wizFieldIdx = 0
			m.wizInput = ""
			if len(m.wizFields) > 0 {
				m.wizInput = m.wizFields[0].Value
			}
			m.wizStep = wizStepFields
			m.screen = screenWizard
		}
		return m, nil
	case "site-wizard", "mt-manage":
		m.wizTarget = wizMikroTik
		m.wizFields = wizBuildFields("mikrotik", &m)
		m.wizFieldIdx = 0
		m.wizInput = ""
		if len(m.wizFields) > 0 {
			m.wizInput = m.wizFields[0].Value
		}
		m.wizStep = wizStepFields
		m.screen = screenWizard
		return m, nil
	case "hostname", "sni-live", "ssh-hosts", "session", "backup-peer", "vpn-sub", "vpn-rename", "sites-list", "rooms-list":
		m.startActionForm(id)
		return m, nil
	default:
		m.output = "unknown action: " + id
		m.screen = screenOutput
		m.outOffset = 0
	}
	return m, nil
}
