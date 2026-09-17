package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Wizard is a multi-step flow inside alt-screen (Esc = previous step / cancel).

type wizTarget string

const (
	wizPrimary   wizTarget = "primary"
	wizSecondary wizTarget = "secondary"
	wizOpenWrt   wizTarget = "openwrt"
	wizMikroTik  wizTarget = "mikrotik"
)

type wizStep int

const (
	wizStepTarget wizStep = iota
	wizStepFields
	wizStepConfirm
	wizStepRun
)

type wizField struct {
	Key, Label, Value string
	Secret            bool
	Placeholder       string
}

func (m *model) startWizard() {
	m.screen = screenWizard
	m.wizStep = wizStepTarget
	m.wizTarget = ""
	m.wizFields = nil
	m.wizFieldIdx = 0
	m.wizInput = ""
	m.wizMsg = ""
	m.cursor = 0
}

func wizTargetEntries(lang tuiLang) []menuEntry {
	if lang == langRU {
		return []menuEntry{
			{"primary", "Primary VPS", "Зарубежный control plane", "Чистый Debian/VPS за границей: install стека, Reality SNI, bootstrap fleet, doctor. После этого обычно настраивают Secondary."},
			{"secondary", "Secondary VPS", "RU entry", "Российский VPS: SSH с primary, provision-secondary (ключ, agent, sing-box, warm services). Пароль только для первого входа."},
			{"openwrt", "OpenWrt / RPi", "Edge agent", "По LAN с Mac: edge provision (бинарь агента + bootstrap). Enroll с backoff, approve на primary."},
			{"mikrotik", "MikroTik", "ROS site", "Сайт MikroTik (+ опционально RPi OpenWrt): identity, маршруты, push скриптов через SSH TOFU."},
		}
	}
	return []menuEntry{
		{"primary", "Primary VPS", "Abroad control plane", "Clean Debian/VPS abroad: stack install, Reality SNI, fleet bootstrap, doctor. Usually followed by Secondary."},
		{"secondary", "Secondary VPS", "RU entry", "RU VPS: SSH from primary, provision-secondary (key, agent, sing-box, warm services). Password only for first login."},
		{"openwrt", "OpenWrt / RPi", "Edge agent", "From Mac over LAN: edge provision (agent binary + bootstrap). Enroll with backoff, approve on primary."},
		{"mikrotik", "MikroTik", "ROS site", "MikroTik site (+ optional RPi OpenWrt): identity, routes, script push via SSH TOFU."},
	}
}

func (m *model) wizBuildFields() {
	lang := m.lang
	ru := lang == langRU
	switch m.wizTarget {
	case wizPrimary:
		m.wizFields = []wizField{
			{Key: "sni", Label: map[bool]string{true: "Reality SNI", false: "Reality SNI"}[true], Value: "api.vk.me", Placeholder: "api.vk.me"},
			{Key: "install", Label: map[bool]string{true: "Запустить install? (yes/no)", false: "Run install? (yes/no)"}[ru], Value: "yes", Placeholder: "yes"},
		}
	case wizSecondary:
		m.wizFields = []wizField{
			{Key: "host", Label: map[bool]string{true: "IP / host secondary", false: "Secondary IP / host"}[ru], Placeholder: "92.x.x.x"},
			{Key: "user", Label: "SSH user", Value: "root"},
			{Key: "pass", Label: map[bool]string{true: "SSH пароль (первый вход)", false: "SSH password (first login)"}[ru], Secret: true},
			{Key: "sni", Label: "Reality SNI", Value: "api.vk.me", Placeholder: "api.vk.me"},
		}
	case wizOpenWrt:
		m.wizFields = []wizField{
			{Key: "host", Label: map[bool]string{true: "IP роутера (LAN)", false: "Router LAN IP"}[ru], Placeholder: "192.168.1.1"},
			{Key: "user", Label: "SSH user", Value: "root"},
			{Key: "pass", Label: map[bool]string{true: "SSH пароль (если нет ключа)", false: "SSH password (if no key)"}[ru], Secret: true},
			{Key: "id", Label: map[bool]string{true: "Device ID", false: "Device ID"}[ru], Value: "home-owrt-1", Placeholder: "home-owrt-1"},
			{Key: "server", Label: map[bool]string{true: "Primary API URL", false: "Primary API URL"}[ru], Value: "http://2.27.118.70:8787", Placeholder: "http://PRIMARY:8787"},
			{Key: "agent", Label: map[bool]string{true: "Путь к agent binary", false: "Path to agent binary"}[ru], Placeholder: "./netductor-agent-linux-arm64"},
		}
	case wizMikroTik:
		m.wizFields = []wizField{
			{Key: "host", Label: "MikroTik IP", Placeholder: "192.168.88.1"},
			{Key: "user", Label: "SSH user", Value: "admin"},
			{Key: "pass", Label: map[bool]string{true: "Пароль", false: "Password"}[ru], Secret: true},
			{Key: "name", Label: map[bool]string{true: "Имя сайта", false: "Site name"}[ru], Value: "home", Placeholder: "home"},
		}
	}
	m.wizFieldIdx = 0
	if len(m.wizFields) > 0 {
		m.wizInput = m.wizFields[0].Value
	}
}

func (m *model) renderWizard() string {
	w := max(40, m.width)
	h := max(8, m.height)
	ru := m.lang == langRU
	title := "Setup wizard"
	if ru {
		title = "Мастер настройки"
	}
	head := stTitle.Width(w).Render(title + "  ·  " + strings.ToUpper(string(map[bool]string{true: "ru", false: "en"}[ru])))
	// language badge time in header via shared
	header := m.renderHeader()
	_ = head

	var body string
	switch m.wizStep {
	case wizStepTarget:
		hint := "↑↓ choose target · Enter · Esc cancel"
		if ru {
			hint = "↑↓ цель · Enter · Esc отмена"
		}
		body = stMuted.Render(hint) + "\n\n"
		for i, e := range wizTargetEntries(m.lang) {
			line := fmt.Sprintf("%d. %s — %s", i+1, e.Title, e.Short)
			if i == m.cursor {
				body += stSel.Width(w-2).Render(line) + "\n"
				body += stDetail.Width(w-4).Render(e.Detail) + "\n"
			} else {
				body += stNorm.Width(w-2).Render(line) + "\n"
			}
		}
	case wizStepFields:
		hint := "Enter = next field · Esc = back"
		if ru {
			hint = "Enter = следующее поле · Esc = назад"
		}
		body = stMuted.Render(hint) + "\n\n"
		for i, f := range m.wizFields {
			val := f.Value
			if i == m.wizFieldIdx {
				val = m.wizInput
				if f.Secret && val != "" {
					val = strings.Repeat("•", len(val))
				}
				body += stSel.Width(w-2).Render(fmt.Sprintf("> %s: %s▌", f.Label, val)) + "\n"
			} else {
				show := f.Value
				if f.Secret && show != "" {
					show = "••••"
				}
				body += stNorm.Width(w-2).Render(fmt.Sprintf("  %s: %s", f.Label, show)) + "\n"
			}
		}
	case wizStepConfirm:
		msg := "Enter = run · Esc = back to fields"
		if ru {
			msg = "Enter = выполнить · Esc = к полям"
		}
		body = stMuted.Render(msg) + "\n\n" + stTitle.Render(string(m.wizTarget)) + "\n"
		for _, f := range m.wizFields {
			show := f.Value
			if f.Secret && show != "" {
				show = "••••"
			}
			body += stNorm.Render(fmt.Sprintf("  %s = %s", f.Label, show)) + "\n"
		}
	case wizStepRun:
		body = lipgloss.NewStyle().Width(w - 2).MaxHeight(h - 6).Render(m.wizMsg)
	}

	helpChips := []helpChip{{"Esc", map[bool]string{true: "назад", false: "back"}[ru]}, {"↵", map[bool]string{true: "далее", false: "next"}[ru]}, {"L", map[bool]string{true: "язык", false: "lang"}[ru]}, {"^C", map[bool]string{true: "выход", false: "quit"}[ru]}}
	var chips strings.Builder
	for _, c := range helpChips {
		chips.WriteString(renderChip(c))
	}
	help := stHelpBG.Width(w).Render(chips.String())
	used := lipgloss.Height(header) + lipgloss.Height(body) + lipgloss.Height(help)
	gap := h - used
	if gap < 0 {
		gap = 0
	}
	return header + "\n" + body + strings.Repeat("\n", gap) + help
}

func (m model) updateWizard(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			m.result = tuiResult{action: "quit", mode: m.mode}
			return m, tea.Quit
		case "l", "L":
			return m.toggleLang()
		case "esc":
			switch m.wizStep {
			case wizStepTarget:
				m.screen = screenMenu
				m.tab = tabTools
				m.cursor = 0
				return m, nil
			case wizStepFields:
				m.wizStep = wizStepTarget
				m.cursor = 0
				return m, nil
			case wizStepConfirm:
				m.wizStep = wizStepFields
				m.wizFieldIdx = 0
				if len(m.wizFields) > 0 {
					m.wizInput = m.wizFields[0].Value
				}
				return m, nil
			case wizStepRun:
				m.screen = screenMenu
				m.tab = tabTools
				return m, nil
			}
		case "up", "k":
			if m.wizStep == wizStepTarget && m.cursor > 0 {
				m.cursor--
			}
			if m.wizStep == wizStepFields && m.wizFieldIdx > 0 {
				m.wizFields[m.wizFieldIdx].Value = m.wizInput
				m.wizFieldIdx--
				m.wizInput = m.wizFields[m.wizFieldIdx].Value
			}
			return m, nil
		case "down", "j":
			if m.wizStep == wizStepTarget && m.cursor < len(wizTargetEntries(m.lang))-1 {
				m.cursor++
			}
			if m.wizStep == wizStepFields && m.wizFieldIdx < len(m.wizFields)-1 {
				m.wizFields[m.wizFieldIdx].Value = m.wizInput
				m.wizFieldIdx++
				m.wizInput = m.wizFields[m.wizFieldIdx].Value
			}
			return m, nil
		case "enter":
			return m.wizardEnter()
		case "backspace":
			if m.wizStep == wizStepFields && len(m.wizInput) > 0 {
				m.wizInput = m.wizInput[:len(m.wizInput)-1]
			}
			return m, nil
		default:
			if m.wizStep == wizStepFields && len(msg.String()) == 1 {
				m.wizInput += msg.String()
			}
			return m, nil
		}
	}
	return m, nil
}

func (m model) wizardEnter() (tea.Model, tea.Cmd) {
	switch m.wizStep {
	case wizStepTarget:
		ents := wizTargetEntries(m.lang)
		if m.cursor < 0 || m.cursor >= len(ents) {
			return m, nil
		}
		m.wizTarget = wizTarget(ents[m.cursor].ID)
		m.wizBuildFields()
		m.wizStep = wizStepFields
		return m, nil
	case wizStepFields:
		if len(m.wizFields) == 0 {
			m.wizStep = wizStepConfirm
			return m, nil
		}
		m.wizFields[m.wizFieldIdx].Value = m.wizInput
		if m.wizFieldIdx < len(m.wizFields)-1 {
			m.wizFieldIdx++
			m.wizInput = m.wizFields[m.wizFieldIdx].Value
			return m, nil
		}
		m.wizStep = wizStepConfirm
		return m, nil
	case wizStepConfirm:
		m.wizStep = wizStepRun
		m.wizMsg = m.runWizardApply()
		return m, nil
	case wizStepRun:
		m.screen = screenMenu
		m.tab = tabTools
		return m, nil
	}
	return m, nil
}

func (m *model) fieldVal(key string) string {
	for _, f := range m.wizFields {
		if f.Key == key {
			return strings.TrimSpace(f.Value)
		}
	}
	return ""
}

func (m model) runWizardApply() string {
	var b strings.Builder
	log := func(s string) { b.WriteString(s); b.WriteString("\n") }
	switch m.wizTarget {
	case wizPrimary:
		sni := m.fieldVal("sni")
		if sni == "" {
			sni = "api.vk.me"
		}
		doInst := strings.HasPrefix(strings.ToLower(m.fieldVal("install")), "y")
		if doInst {
			log("→ netductor install")
			cmd := exec.Command("netductor", "install")
			out, err := cmd.CombinedOutput()
			b.Write(out)
			if err != nil {
				log("install: " + err.Error())
			}
		}
		log("→ vpn set-sni " + sni)
		_ = exec.Command("netductor", "vpn", "set-sni", sni).Run()
		_ = exec.Command("netductor", "fleet", "bootstrap").Run()
		out, _ := exec.Command("netductor", "doctor").CombinedOutput()
		b.Write(out)
		log("Done. Next: Secondary wizard.")
	case wizSecondary:
		host, user, pass, sni := m.fieldVal("host"), m.fieldVal("user"), m.fieldVal("pass"), m.fieldVal("sni")
		if host == "" || pass == "" {
			return "host and password required"
		}
		if user == "" {
			user = "root"
		}
		if sni == "" {
			sni = "api.vk.me"
		}
		log("→ fleet provision-secondary " + host)
		cmd := exec.Command("netductor", "fleet", "provision-secondary",
			"--host", host, "--user", user, "--password", pass, "--sni", sni)
		out, err := cmd.CombinedOutput()
		b.Write(out)
		if err != nil {
			log(err.Error())
		}
		out, _ = exec.Command("netductor", "fleet", "status").CombinedOutput()
		b.Write(out)
	case wizOpenWrt:
		host, user, id, server, agent := m.fieldVal("host"), m.fieldVal("user"), m.fieldVal("id"), m.fieldVal("server"), m.fieldVal("agent")
		if host == "" {
			return "router host required"
		}
		if user == "" {
			user = "root"
		}
		if id == "" {
			id = "home-owrt-1"
		}
		if server == "" {
			server = "http://127.0.0.1:8787"
		}
		args := []string{"edge", "provision", user + "@" + host, "--id", id, "--server", server}
		if agent != "" {
			if abs, err := filepath.Abs(agent); err == nil {
				agent = abs
			}
			args = append(args, "--agent", agent)
		}
		if pass := m.fieldVal("pass"); pass != "" {
			args = append(args, "--password", pass)
		}
		log("→ netductor " + strings.Join(args, " "))
		cmd := exec.Command("netductor", args...)
		out, err := cmd.CombinedOutput()
		b.Write(out)
		if err != nil {
			log(err.Error())
			log("Fallback: copy agent manually, set /etc/netductor-agent/config, start service.")
		} else {
			log("Approve device on primary: netductor edge approve " + id)
		}
	case wizMikroTik:
		host, user, pass, name := m.fieldVal("host"), m.fieldVal("user"), m.fieldVal("pass"), m.fieldVal("name")
		if host == "" {
			return "MikroTik host required"
		}
		log("→ site / MikroTik manage " + host)
		// Best-effort CLI if present
		cmd := exec.Command("netductor", "site", "init", "--host", host, "--user", user, "--password", pass, "--name", name)
		out, err := cmd.CombinedOutput()
		b.Write(out)
		if err != nil {
			log(err.Error())
			log("Use Tools → MikroTik manage or docs/MIKROTIK.md")
		}
	default:
		return "unknown target"
	}
	_ = os.Stdout
	return b.String()
}
