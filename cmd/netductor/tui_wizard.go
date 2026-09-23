package main

import (
	"strings"

	"github.com/atotto/clipboard"

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
	Toggle            bool   // checklist item: Value yes/no, Space toggles
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
			{"addons", "Дополнения (чек-лист)", "Lampac · TG · go2rtc", "Целевая VPS + чек-лист; TG-токен/admin при выборе бота."},
		}
	}
	return []menuEntry{
		{"primary", "Primary VPS", "Abroad control plane", "Clean Debian/VPS abroad: stack install, Reality SNI, fleet bootstrap, doctor. Usually followed by Secondary."},
		{"secondary", "Secondary VPS", "RU entry", "RU VPS: SSH from primary, provision-secondary (key, agent, sing-box, VPN entry). Password only for first login."},
		{"openwrt", "OpenWrt / RPi", "Edge agent", "From Mac over LAN: edge provision (agent binary + bootstrap). Enroll with backoff, approve on primary."},
		{"mikrotik", "MikroTik", "ROS site", "MikroTik site (+ optional RPi OpenWrt): identity, routes, script push via SSH TOFU."},
		{"nvr", "Cameras / NVR", "Cameras", "NVR: leases, add camera, probe, record — via primary after edge."},
		{"addons", "Add-ons (checklist)", "any VPS", "Host/key + [✓] Lampac/TG/go2rtc. TG needs token + admin id. Not primary-only."},
	}
}

// wizBuildFields removed: deploy targets use huh wizards in tui_deploy_wizards.go (single path).

func (m *model) renderWizard() string {
	w := max(40, m.width)
	h := max(8, m.height)
	ru := m.lang == langRU
	header := m.renderHeader()
	helpChips := m.wizardHelpChips(ru)
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
		ru2 := m.lang == langRU
		title := "Running · " + string(m.wizTarget)
		if ru2 {
			title = "Выполнение · " + string(m.wizTarget)
		}
		var statusLine string
		if m.wizRunning {
			statusLine = map[bool]string{true: "Идёт работа… p=прогресс o=лог ↑↓=скролл", false: "In progress… p=progress o=log ↑↓=scroll"}[ru2]
		} else {
			statusLine = map[bool]string{
				true:  "Готово. Enter или Esc — в главное меню · p/o · ↑↓ скролл лога",
				false: "Done. Enter or Esc — main menu · p/o · ↑↓ scroll log",
			}[ru2]
		}
		head2 := stTitle.Render(title) + "\n" + stMuted.Render(statusLine) + "\n"
		var content string
		if m.wizViewMode == "steps" {
			content = m.renderWizardStepsView(w - 6)
		} else {
			content = m.scrollableLog(max(3, bodyH-6))
		}
		innerH := bodyH - 2
		if innerH < 3 {
			innerH = 3
		}
		inner := lipgloss.NewStyle().Width(w - 4).Height(innerH).MaxHeight(innerH).Render(head2 + content)
		body = stBorder.Width(w).Height(bodyH).MaxHeight(bodyH).Render(inner)
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
	hasToggle := false
	for _, f := range m.wizFields {
		if f.Toggle {
			hasToggle = true
			break
		}
	}
	if hasToggle {
		hint = "Space = toggle · ↑↓ · Enter next · Ctrl+R install"
		if ru {
			hint = "Space = вкл/выкл · ↑↓ · Enter далее · Ctrl+R установка"
		}
	}
	var leftLines []string
	leftLines = append(leftLines, stMuted.Width(leftW).Render(hint))
	leftLines = append(leftLines, stMuted.Width(leftW).Render(strings.ToUpper(string(m.wizTarget))))
	for i, f := range m.wizFields {
		val := f.Value
		if i == m.wizFieldIdx {
			val = m.wizInput
			if f.Toggle {
				mark := "[ ]"
				if yesish(val) {
					mark = "[✓]"
				}
				leftLines = append(leftLines, stSel.Width(leftW).Render(mark+" "+f.Label))
				if f.Short != "" {
					leftLines = append(leftLines, stMuted.Width(leftW).Render("  "+f.Short))
				}
			} else {
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
			}
		} else {
			if f.Toggle {
				mark := "[ ]"
				if yesish(f.Value) {
					mark = "[✓]"
				}
				leftLines = append(leftLines, stNorm.Width(leftW).Render(mark+" "+f.Label))
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
	msg := "Ctrl+R or Enter = START deploy · Esc = edit fields"
	if ru {
		msg = "Ctrl+R или Enter = ЗАПУСК · Esc = править поля"
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
		if f.Toggle {
			mark := "[ ]"
			if yesish(f.Value) {
				mark = "[✓]"
			}
			leftLines = append(leftLines, stNorm.Width(leftW).Render(mark+" "+f.Label))
			continue
		}
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
		true:  "Проверьте значения слева.\n\nCtrl+R или Enter — ЗАПУСТИТЬ деплой.\nEsc — к полям.\n\nПосле успеха: doctor / status.",
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
	case wizLogMsg:
		m.wizMsg += msg.Line
		if len(m.wizMsg) > 12000 {
			m.wizMsg = m.wizMsg[len(m.wizMsg)-10000:]
		}
		m.applyLogLine(msg.Line)
		return m, listenWizardStream()
	case wizDoneMsg:
		if msg.Text != "" {
			m.wizMsg += msg.Text
			if !strings.HasSuffix(m.wizMsg, "\n") {
				m.wizMsg += "\n"
			}
		}
		if msg.OK {
			m.wizMsg += TT(m.lang, "\n\n══════════════════════════════════════\n✓ Install/setup finished.\n  Press Enter or Esc → main menu\n  Then: Tools → doctor / status to verify\n══════════════════════════════════════\n", "\n\n══════════════════════════════════════\n✓ Установка/настройка завершена.\n  Enter или Esc → главное меню\n  Далее: Инструменты → doctor / status\n══════════════════════════════════════\n")
		} else {
			m.wizMsg += TT(m.lang, "\n✗ Failed. Enter / Esc = back\n", "\n✗ Ошибка. Enter / Esc = назад\n")
		}
		m.wizRunning = false
		if msg.OK {
			for i := range m.wizSteps {
				m.wizSteps[i].Done = true
				m.wizSteps[i].Active = false
			}
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			m.result = tuiResult{action: "quit", mode: m.mode}
			return m, tea.Quit
		case "ctrl+r", "ctrl+R":
			if m.wizStep == wizStepConfirm {
				return m.wizardEnter()
			}
			if m.wizStep == wizStepFields && string(m.wizTarget) == "addons" {
				// sync current field and jump to confirm then run
				if m.wizFieldIdx < len(m.wizFields) {
					m.wizFields[m.wizFieldIdx].Value = m.wizInput
				}
				m.wizStep = wizStepConfirm
				return m.wizardEnter()
			}
			return m, nil
		case "p":
			if m.wizStep == wizStepRun {
				m.wizViewMode = "steps"
				return m, nil
			}
		case "o":
			if m.wizStep == wizStepRun {
				m.wizViewMode = "log"
				return m, nil
			}
		case "pgup":
			if m.wizStep == wizStepRun && m.wizViewMode != "steps" {
				m.wizLogOffset += 10
				m.clampWizLogOffset(20)
				return m, nil
			}
		case "pgdown", "ctrl+d":
			if m.wizStep == wizStepRun && m.wizViewMode != "steps" {
				m.wizLogOffset -= 10
				if m.wizLogOffset < 0 {
					m.wizLogOffset = 0
				}
				return m, nil
			}
		case "home":
			if m.wizStep == wizStepRun {
				// jump to top of log
				n := strings.Count(m.wizMsg, "\n")
				m.wizLogOffset = n
				return m, nil
			}
		case "end":
			if m.wizStep == wizStepRun {
				m.wizLogOffset = 0
				return m, nil
			}
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
			if m.wizStep == wizStepRun && m.wizViewMode != "steps" {
				m.wizLogOffset++
				m.clampWizLogOffset(20)
				return m, nil
			}
			if m.wizStep == wizStepFields && m.wizFieldIdx > 0 {
				m.wizFields[m.wizFieldIdx].Value = m.wizInput
				m.wizFieldIdx--
				m.wizInput = m.wizFields[m.wizFieldIdx].Value
			}
			return m, nil
		case "down", "j":
			if m.wizStep == wizStepRun && m.wizViewMode != "steps" {
				if m.wizLogOffset > 0 {
					m.wizLogOffset--
				}
				return m, nil
			}
			if m.wizStep == wizStepFields && m.wizFieldIdx < len(m.wizFields)-1 {
				m.wizFields[m.wizFieldIdx].Value = m.wizInput
				m.wizFieldIdx++
				m.wizInput = m.wizFields[m.wizFieldIdx].Value
			}
			return m, nil
		case " ", "space":
			if m.wizStep == wizStepFields && m.wizFieldIdx < len(m.wizFields) {
				f := &m.wizFields[m.wizFieldIdx]
				if f.Toggle {
					if yesish(m.wizInput) || yesish(f.Value) {
						m.wizInput = "no"
						f.Value = "no"
					} else {
						m.wizInput = "yes"
						f.Value = "yes"
					}
					return m, nil
				}
			}
			return m, nil
		case "enter":
			return m.wizardEnter()
		case "backspace", "ctrl+h":
			if m.wizStep == wizStepFields && len(m.wizInput) > 0 {
				r := []rune(m.wizInput)
				m.wizInput = string(r[:len(r)-1])
			}
			return m, nil
		case "ctrl+u":
			if m.wizStep == wizStepRun && m.wizViewMode != "steps" {
				m.wizLogOffset += 10
				return m, nil
			}
			if m.wizStep == wizStepFields {
				m.wizInput = ""
			}
			return m, nil
		case "ctrl+v":
			if m.wizStep == wizStepFields {
				if s, err := clipboard.ReadAll(); err == nil && s != "" {
					m.wizInput = stripFieldNewlines(m.wizInput + s)
				}
			}
			return m, nil
		default:
			if m.wizStep == wizStepFields {
				if text, ok := keyTextForField(msg); ok {
					m.wizInput = stripFieldNewlines(m.wizInput + text)
				}
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
		if string(m.wizTarget) == "remote" {
			m.remoteHost = m.fieldVal("host")
			m.remoteUser = orDefault(m.fieldVal("user"), "root")
			m.wizMsg = "remote = " + m.remoteLabel() + "\n(SSH key in agent; BatchMode=yes)"
			m.wizStep = wizStepRun
			m.wizRunning = false
			m.screen = screenWizard
			return m, nil
		}
		if m.formAction != "" || string(m.wizTarget) == "form" {
			m.wizMsg = m.submitActionForm()
			m.formAction = ""
			m.wizStep = wizStepRun
			m.wizRunning = false
			m.screen = screenWizard
			return m, nil
		}
		m.wizMsg = TT(m.lang, "Working… please wait.\n", "Работаю… подождите.\n")
		m.wizSteps = wizStepsForTarget(m.wizTarget, m.lang)
		m.wizViewMode = "steps"
		m.wizStep = wizStepRun
		m.wizRunning = true
		m.screen = screenWizard
		return m, startWizardDeployCmd(m)
	case wizStepRun:
		if m.wizRunning {
			return m, nil
		}
		m.wizStep = wizStepTarget
		m.wizMsg = ""
		m.screen = screenMenu
		m.tab = tabWizard
		m.cursor = 0
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

func keyTextForField(msg tea.KeyMsg) (string, bool) {
	if msg.Paste {
		if len(msg.Runes) > 0 {
			return string(msg.Runes), true
		}
		s := msg.String()
		return s, s != ""
	}
	switch msg.Type {
	case tea.KeyRunes:
		return string(msg.Runes), len(msg.Runes) > 0
	case tea.KeySpace:
		return " ", true
	default:
		s := msg.String()
		// Some terminals deliver paste as one long key string without Paste=true
		if len(s) > 1 && !strings.HasPrefix(s, "ctrl+") && !strings.HasPrefix(s, "alt+") &&
			!strings.HasPrefix(s, "shift+") && s != "enter" && s != "tab" && s != "esc" &&
			s != "up" && s != "down" && s != "left" && s != "right" &&
			s != "backspace" && s != "delete" && s != "space" {
			return s, true
		}
		if s == " " {
			return " ", true
		}
		return "", false
	}
}

func stripFieldNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}


func (m model) wizardHelpChips(ru bool) []helpChip {
	L := func(en, r string) string {
		if ru {
			return r
		}
		return en
	}
	switch m.wizStep {
	case wizStepFields:
		return []helpChip{
			{"Esc", L("back", "назад")},
			{"↵", L("next field", "след. поле")},
			{"↑↓", L("field", "поле")},
			{"L", L("lang", "язык")},
			{"^C", L("quit", "выход")},
		}
	case wizStepConfirm:
		return []helpChip{
			{"^R", L("START", "ЗАПУСК")},
			{"↵", L("START", "ЗАПУСК")},
			{"Esc", L("edit", "править")},
			{"L", L("lang", "язык")},
			{"^C", L("quit", "выход")},
		}
	case wizStepRun:
		chips := []helpChip{
			{"p", L("progress", "прогресс")},
			{"o", L("log", "лог")},
			{"↑↓", L("scroll", "скролл")},
		}
		if !m.wizRunning {
			chips = append([]helpChip{{"↵/Esc", L("main menu", "в меню")}}, chips...)
		}
		chips = append(chips, helpChip{"L", L("lang", "язык")}, helpChip{"^C", L("quit", "выход")})
		return chips
	default:
		return []helpChip{
			{"Esc", L("back", "назад")},
			{"↵", L("next", "далее")},
			{"L", L("lang", "язык")},
			{"^C", L("quit", "выход")},
		}
	}
}

func (m model) scrollableLog(maxLines int) string {
	if maxLines < 1 {
		maxLines = 1
	}
	lines := strings.Split(m.wizMsg, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	n := len(lines)
	if n == 0 {
		return stMuted.Render(map[bool]string{true: "(лог пуст)", false: "(empty log)"}[m.lang == langRU])
	}
	off := m.wizLogOffset
	if off < 0 {
		off = 0
	}
	maxOff := n - maxLines
	if maxOff < 0 {
		maxOff = 0
	}
	if off > maxOff {
		off = maxOff
	}
	end := n - off
	if end > n {
		end = n
	}
	start := end - maxLines
	if start < 0 {
		start = 0
	}
	chunk := lines[start:end]
	out := strings.Join(chunk, "\n")
	if start > 0 || off > 0 {
		more := map[bool]string{true: "… ↑ есть выше · ↓ ниже", false: "… ↑ more above · ↓ below"}[m.lang == langRU]
		out = stMuted.Render(more) + "\n" + out
	}
	return out
}

func (m *model) clampWizLogOffset(viewport int) {
	if viewport < 1 {
		viewport = 1
	}
	lines := strings.Split(m.wizMsg, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	n := len(lines)
	maxOff := n - viewport
	if maxOff < 0 {
		maxOff = 0
	}
	if m.wizLogOffset < 0 {
		m.wizLogOffset = 0
	}
	if m.wizLogOffset > maxOff {
		m.wizLogOffset = maxOff
	}
}

