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
	Short             string // one-line hint under label (menu style)
	Detail            string // right pane full description
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
	header := m.renderHeader()
	helpChips := []helpChip{
		{"Esc", map[bool]string{true: "назад", false: "back"}[ru]},
		{"↵", map[bool]string{true: "далее", false: "next"}[ru]},
		{"↑↓", map[bool]string{true: "поле", false: "field"}[ru]},
		{"L", map[bool]string{true: "язык", false: "lang"}[ru]},
		{"^C", map[bool]string{true: "выход", false: "quit"}[ru]},
	}
	var chips strings.Builder
	for _, c := range helpChips {
		chips.WriteString(renderChip(c))
	}
	help := stHelpBG.Width(w).Render(chips.String())
	helpH := lipgloss.Height(help)
	headH := lipgloss.Height(header)
	bodyH := h - headH - helpH - 1
	if bodyH < 5 {
		bodyH = 5
	}

	var body string
	switch m.wizStep {
	case wizStepTarget:
		// same split as main menu
		ents := wizTargetEntries(m.lang)
		body = m.renderWizardSplitEntries(ents, bodyH, w)
	case wizStepFields:
		body = m.renderWizardSplitFields(bodyH, w)
	case wizStepConfirm:
		body = m.renderWizardConfirm(bodyH, w)
	case wizStepRun:
		body = lipgloss.NewStyle().Width(w - 2).Height(bodyH).MaxHeight(bodyH).Render(m.wizMsg)
	default:
		body = ""
	}

	used := headH + lipgloss.Height(body) + helpH
	gap := h - used
	if gap < 0 {
		gap = 0
	}
	return header + "\n" + body + strings.Repeat("\n", gap) + help
}

func (m *model) renderWizardSplitEntries(entries []menuEntry, bodyH, w int) string {
	leftW := w * 2 / 5
	if leftW < 28 {
		leftW = 28
	}
	if leftW > 48 {
		leftW = 48
	}
	rightW := w - leftW - 3
	if rightW < 20 {
		rightW = 20
		leftW = w - rightW - 3
	}
	var leftLines []string
	for i, e := range entries {
		if i == m.cursor {
			leftLines = append(leftLines, stSel.Width(leftW).Render(e.Title))
			leftLines = append(leftLines, stSel.Width(leftW).Render("  "+e.Short))
		} else {
			leftLines = append(leftLines, stNorm.Width(leftW).Render(e.Title))
			leftLines = append(leftLines, stMuted.Width(leftW).Render("  "+e.Short))
		}
	}
	leftBody := lipgloss.NewStyle().Width(leftW).Height(bodyH).MaxHeight(bodyH).Render(strings.Join(leftLines, "\n"))
	detail := ""
	if len(entries) > 0 {
		e := entries[m.cursor]
		if m.cursor >= len(entries) {
			e = entries[len(entries)-1]
		}
		detail = stDetail.Width(rightW - 2).Render(stTitle.Render(e.Title) + "\n\n" + e.Detail)
	}
	detail = stBorder.Width(rightW).Height(bodyH).MaxHeight(bodyH).Render(
		lipgloss.NewStyle().Width(rightW - 2).Height(bodyH - 2).Render(detail),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBody, " ", detail)
}

func (m *model) renderWizardSplitFields(bodyH, w int) string {
	ru := m.lang == langRU
	leftW := w * 2 / 5
	if leftW < 30 {
		leftW = 30
	}
	if leftW > 52 {
		leftW = 52
	}
	rightW := w - leftW - 3
	if rightW < 22 {
		rightW = 22
		leftW = w - rightW - 3
	}
	hint := "Enter next field · ↑↓ move · Esc back"
	if ru {
		hint = "Enter — следующее · ↑↓ поле · Esc назад"
	}
	var leftLines []string
	leftLines = append(leftLines, stMuted.Width(leftW).Render(hint))
	leftLines = append(leftLines, stMuted.Width(leftW).Render(strings.ToUpper(string(m.wizTarget))))
	for i, f := range m.wizFields {
		val := f.Value
		if i == m.wizFieldIdx {
			val = m.wizInput
			if f.Secret && val != "" {
				val = strings.Repeat("•", len(val))
			} else if val == "" && f.Placeholder != "" {
				val = f.Placeholder
			}
			leftLines = append(leftLines, stSel.Width(leftW).Render(f.Label))
			leftLines = append(leftLines, stSel.Width(leftW).Render("  "+val+"▌"))
			if f.Short != "" {
				leftLines = append(leftLines, stMuted.Width(leftW).Render("  "+f.Short))
			}
		} else {
			show := f.Value
			if f.Secret && show != "" {
				show = "••••"
			}
			if show == "" {
				show = "—"
			}
			leftLines = append(leftLines, stNorm.Width(leftW).Render(f.Label))
			leftLines = append(leftLines, stMuted.Width(leftW).Render("  "+show))
		}
	}
	leftBody := lipgloss.NewStyle().Width(leftW).Height(bodyH).MaxHeight(bodyH).Render(strings.Join(leftLines, "\n"))

	detailTitle := ""
	detailBody := ""
	if len(m.wizFields) > 0 && m.wizFieldIdx < len(m.wizFields) {
		f := m.wizFields[m.wizFieldIdx]
		detailTitle = f.Label
		detailBody = f.Detail
		if detailBody == "" {
			detailBody = f.Short
		}
		if f.Placeholder != "" {
			detailBody += "\n\n" + map[bool]string{true: "Подсказка: ", false: "Placeholder: "}[ru] + f.Placeholder
		}
	}
	detail := stDetail.Width(rightW - 2).Render(stTitle.Render(detailTitle) + "\n\n" + detailBody)
	detail = stBorder.Width(rightW).Height(bodyH).MaxHeight(bodyH).Render(
		lipgloss.NewStyle().Width(rightW - 2).Height(bodyH - 2).Render(detail),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBody, " ", detail)
}

func (m *model) renderWizardConfirm(bodyH, w int) string {
	ru := m.lang == langRU
	msg := "Enter = run deploy · Esc = edit fields"
	if ru {
		msg = "Enter = запуск · Esc = править поля"
	}
	leftW := w * 2 / 5
	if leftW < 30 {
		leftW = 30
	}
	if leftW > 52 {
		leftW = 52
	}
	rightW := w - leftW - 3
	var leftLines []string
	leftLines = append(leftLines, stMuted.Width(leftW).Render(msg))
	leftLines = append(leftLines, stTitle.Width(leftW).Render(strings.ToUpper(string(m.wizTarget))))
	for _, f := range m.wizFields {
		show := f.Value
		if f.Secret && show != "" {
			show = "••••"
		}
		if show == "" {
			show = "—"
		}
		leftLines = append(leftLines, stNorm.Width(leftW).Render(f.Label))
		leftLines = append(leftLines, stMuted.Width(leftW).Render("  "+show))
	}
	leftBody := lipgloss.NewStyle().Width(leftW).Height(bodyH).MaxHeight(bodyH).Render(strings.Join(leftLines, "\n"))
	sum := map[bool]string{
		true:  "Проверьте значения слева.\n\nEnter — выполнить деплой.\nEsc — вернуться к полям.",
		false: "Review values on the left.\n\nEnter — run deploy.\nEsc — back to fields.",
	}[ru]
	detail := stDetail.Width(rightW - 2).Render(stTitle.Render(map[bool]string{true: "Подтверждение", false: "Confirm"}[ru]) + "\n\n" + sum)
	detail = stBorder.Width(rightW).Height(bodyH).MaxHeight(bodyH).Render(
		lipgloss.NewStyle().Width(rightW - 2).Height(bodyH - 2).Render(detail),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBody, " ", detail)
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
