package main

import "runtime"

// startActionForm opens in-TUI fields for an action (no tea.Quit / huh).
func (m *model) startActionForm(action string) {
	m.formAction = action
	ru := m.lang == langRU
	switch action {
	case "cfg-remote":
		m.wizFields = []wizField{
			{Key: "host", Label: map[bool]string{true: "Host / IP", false: "Host / IP"}[true], Value: m.remoteHost, Placeholder: "vps.example.com"},
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
	case "edge-register":
		m.wizFields = []wizField{
			{Key: "id", Label: map[bool]string{true: "Device ID", false: "Device ID"}[ru], Placeholder: "cudy-home-1"},
			{Key: "site", Label: map[bool]string{true: "Location / site id", false: "Location / site id"}[ru], Placeholder: "home-msk"},
			{Key: "note", Label: map[bool]string{true: "Заметка", false: "Note"}[ru]},
		}
	case "hostname":
		m.wizFields = []wizField{
			{Key: "name", Label: "Hostname", Placeholder: "nd-core-nl01"},
		}
	case "sni-live":
		m.wizFields = []wizField{
			{Key: "sni", Label: "Reality SNI", Value: "api.vk.me"},
		}
	case "vpn-rename":
		m.wizFields = []wizField{
			{Key: "old", Label: "Old name"},
			{Key: "new", Label: "New name"},
		}
	case "vpn-sub":
		m.wizFields = []wizField{
			{Key: "name", Label: "VPN user"},
		}
	case "dns-on", "dns-off":
		m.wizFields = []wizField{
			{Key: "id", Label: map[bool]string{true: "ID списка", false: "List id"}[ru], Placeholder: "adguard"},
		}
	case "backup-peer":
		m.wizFields = []wizField{
			{Key: "peer", Label: "Peer URL / id"},
		}
	case "session":
		m.wizFields = []wizField{
			{Key: "user", Label: "VPN user"},
		}
	case "ssh-hosts", "sites-list":
		m.wizFields = []wizField{
			{Key: "note", Label: "Enter to run", Value: "ok"},
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
	case "edge-register":
		id := m.fieldVal("id")
		if id == "" {
			return "device_id required"
		}
		args := []string{"edge", "register", id}
		if s := m.fieldVal("site"); s != "" {
			args = append(args, "--site", s)
		}
		if n := m.fieldVal("note"); n != "" {
			args = append(args, "--note", n)
		}
		return m.runNetductor(args...)
	case "hostname":
		name := m.fieldVal("name")
		if name == "" {
			return "hostname required"
		}
		return m.runNetductor("hostname", "set", name)
	case "sni-live":
		return m.runNetductor("vpn", "set-sni", orDefault(m.fieldVal("sni"), "api.vk.me"))
	case "vpn-rename":
		return m.runNetductor("vpn", "rename", m.fieldVal("old"), m.fieldVal("new"))
	case "vpn-sub":
		return m.runNetductor("vpn", "sub", m.fieldVal("name"))
	case "dns-on":
		return m.runNetductor("dns", "on", m.fieldVal("id"))
	case "dns-off":
		return m.runNetductor("dns", "off", m.fieldVal("id"))
	case "backup-peer":
		return m.runNetductor("backup", "peer", m.fieldVal("peer"))
	case "session":
		return m.runNetductor("vpn", "session", m.fieldVal("user"))
	case "ssh-hosts":
		return m.runNetductor("ssh", "hosts")
	case "sites-list":
		return m.runNetductor("sites", "list")
	default:
		return "unknown form"
	}
}
