package main

import "runtime"

// startActionForm opens in-TUI fields for an action (no tea.Quit / huh).
func (m *model) startActionForm(action string) {
	m.formAction = action
	ru := m.lang == langRU
	switch action {
	case "cfg-remote":
		m.wizFields = []wizField{
			{Key: "host", Label: map[bool]string{true: "Host / IP", false: "Host / IP"}[true], Value: m.remoteHost, Placeholder: "2.27.118.70"},
			{Key: "user", Label: "SSH user", Value: orDefault(m.remoteUser, "root")},
			{Key: "key", Label: map[bool]string{true: "Путь к ключу (пусто = agent)", false: "Key path (empty = agent)"}[ru], Value: m.remoteKey, Placeholder: "~/.ssh/id_ed25519"},
			{Key: "pass", Label: map[bool]string{true: "Пароль (если нет ключа)", false: "Password (if no key)"}[ru], Value: m.remotePassword, Secret: true},
		}
	case "vpn-add":
		m.wizFields = []wizField{
			{Key: "name", Label: map[bool]string{true: "Имя пользователя", false: "User name"}[ru], Placeholder: "alice"},
			{Key: "note", Label: map[bool]string{true: "Заметка", false: "Note"}[ru]},
		}
	case "build":
		goos, goarch := runtime.GOOS, runtime.GOARCH
		m.wizFields = []wizField{
			{Key: "goos", Label: "GOOS", Value: "linux"},
			{Key: "goarch", Label: "GOARCH", Value: "amd64"},
			{Key: "note", Label: map[bool]string{true: "Подсказка", false: "Hint"}[ru], Value: "local=" + goos + "/" + goarch},
		}
	case "owrt-install":
		m.wizFields = []wizField{
			{Key: "host", Label: "OpenWrt host", Placeholder: "192.168.1.1"},
			{Key: "user", Label: "SSH user", Value: "root"},
			{Key: "pass", Label: "SSH password", Secret: true},
			{Key: "id", Label: "Device ID", Value: "home-owrt-1"},
			{Key: "server", Label: "Primary API", Value: "http://127.0.0.1:8787", Placeholder: "http://PRIMARY:8787"},
		}
	default:
		m.output = "unknown form " + action
		m.screen = screenOutput
		return
	}
	m.wizTarget = "form"
	m.wizFieldIdx = 0
	m.wizInput = m.wizFields[0].Value
	m.wizStep = wizStepFields
	m.screen = screenWizard
}

func (m *model) submitActionForm() string {
	switch m.formAction {
	case "cfg-remote":
		m.remoteHost = m.fieldVal("host")
		m.remoteUser = orDefault(m.fieldVal("user"), "root")
		m.remoteKey = m.fieldVal("key")
		m.remotePassword = m.fieldVal("pass")
		_ = saveTUISettings(m.snapshotSettings())
		return "remote = " + m.remoteLabel() + "\nsaved " + tuiConfigPath()
	case "vpn-add":
		name := m.fieldVal("name")
		if name == "" {
			return "name required"
		}
		args := []string{"vpn", "add", name}
		if n := m.fieldVal("note"); n != "" {
			args = append(args, "--note", n)
		}
		return m.runNetductor(args...)
	case "build":
		return m.runNetductor("build", "--goos", orDefault(m.fieldVal("goos"), "linux"), "--goarch", orDefault(m.fieldVal("goarch"), "amd64"))
	case "owrt-install":
		host := m.fieldVal("host")
		user := orDefault(m.fieldVal("user"), "root")
		if host == "" {
			return "host required"
		}
		args := []string{"edge", "provision", user + "@" + host, "--id", orDefault(m.fieldVal("id"), "home-owrt-1"), "--server", orDefault(m.fieldVal("server"), "http://127.0.0.1:8787")}
		if p := m.fieldVal("pass"); p != "" {
			args = append(args, "--password", p)
		}
		// edge provision is local SSH to router, not remote netductor
		return m.runNetductor(args...)
	default:
		return "unknown form"
	}
}
