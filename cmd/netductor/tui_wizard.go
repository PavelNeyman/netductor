package main

import (
	"fmt"
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
	wizNVR       wizTarget = "nvr"
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
			{"secondary", "Secondary VPS", "RU entry", "Российский VPS: SSH с primary, provision-secondary (ключ, agent, sing-box, VPN entry). Пароль только для первого входа."},
			{"openwrt", "OpenWrt / RPi", "Edge agent", "По LAN с Mac: edge provision (бинарь агента + bootstrap). Enroll с backoff, approve на primary."},
			{"mikrotik", "MikroTik", "ROS site", "Сайт MikroTik (+ опционально RPi OpenWrt): identity, маршруты, push скриптов через SSH TOFU."},
			{"nvr", "Cameras / NVR", "Камеры", "NVR: leases, add camera, probe, record — через primary после edge."},
			{"addons", "Дополнения", "Lampac", "Опционально: Lampac (Docker) на primary по SSH."},
		}
	}
	return []menuEntry{
		{"primary", "Primary VPS", "Abroad control plane", "Clean Debian/VPS abroad: stack install, Reality SNI, fleet bootstrap, doctor. Usually followed by Secondary."},
		{"secondary", "Secondary VPS", "RU entry", "RU VPS: SSH from primary, provision-secondary (key, agent, sing-box, VPN entry). Password only for first login."},
		{"openwrt", "OpenWrt / RPi", "Edge agent", "From Mac over LAN: edge provision (agent binary + bootstrap). Enroll with backoff, approve on primary."},
		{"mikrotik", "MikroTik", "ROS site", "MikroTik site (+ optional RPi OpenWrt): identity, routes, script push via SSH TOFU."},
		{"nvr", "Cameras / NVR", "Cameras", "NVR: leases, add camera, probe, record — via primary after edge."},
		{"addons", "Add-ons", "Lampac", "Optional: Lampac (Docker) on primary over SSH."},
	}
}

// wizBuildFields removed: deploy targets use huh wizards in tui_deploy_wizards.go (single path).

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
		// Single path: confirm → tui_deploy_wizards (huh), not inline fields
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
		case "tab":
			m.screen = screenMenu
			m.tab = tabTools
			m.cursor = 0
			m.wizStep = wizStepTarget
			return m, nil
		case "esc":
			switch m.wizStep {
			case wizStepTarget:
				m.screen = screenMenu
				m.tab = tabWizard
				m.cursor = 0
				return m, nil
			case wizStepFields:
				if string(m.wizTarget) == "remote" || string(m.wizTarget) == "form" || m.formAction != "" {
					m.formAction = ""
					m.screen = screenMenu
					m.tab = tabSettings
					m.cursor = 0
					return m, nil
				}
				m.screen = screenMenu
				m.tab = tabWizard
				m.wizStep = wizStepTarget
				m.cursor = 0
				return m, nil
			case wizStepConfirm:
		// Single path: confirm → tui_deploy_wizards (huh), not inline fields
				m.wizStep = wizStepFields
				m.wizFieldIdx = 0
				if len(m.wizFields) > 0 {
					m.wizInput = m.wizFields[0].Value
				}
				return m, nil
			case wizStepRun:
				m.screen = screenMenu
				m.tab = tabWizard
				m.wizStep = wizStepTarget
				return m, nil
			}
		case "up", "k":
			if m.wizStep == wizStepFields && m.wizFieldIdx > 0 {
				m.wizFields[m.wizFieldIdx].Value = m.wizInput
				m.wizFieldIdx--
				m.wizInput = m.wizFields[m.wizFieldIdx].Value
			}
			return m, nil
		case "down", "j":
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
		// Exit alt-screen; run huh outside Bubble Tea (see runTUI switch).
		if string(m.wizTarget) == "remote" {
			m.remoteHost = m.fieldVal("host")
			m.remoteUser = orDefault(m.fieldVal("user"), "root")
			m.output = "remote = " + m.remoteLabel() + "\n(SSH key in agent; BatchMode=yes)"
			m.screen = screenOutput
			m.tab = tabTools
			m.wizStep = wizStepTarget
			return m, nil
		}
		m.output = m.runWizardApplyInTUI()
		m.screen = screenOutput
		m.tab = tabWizard
		m.wizStep = wizStepTarget
		return m, nil
	case wizStepRun:
		m.screen = screenMenu
		m.tab = tabWizard
		m.wizStep = wizStepTarget
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
	return m.runWizardApplyInTUI()
}
