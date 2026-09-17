package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type hitRect struct {
	X, Y, W, H int
	Action     string // tab:wizard|tools|ops|mode, lang, row:N
}

var (
	stTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1)
	stTabOn   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("63")).Padding(0, 1)
	stTabOff  = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238")).Padding(0, 1)
	stChipKey = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Padding(0, 1)
	stChipLab = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236")).Padding(0, 1).MarginRight(1)
	stHeader  = lipgloss.NewStyle().Background(lipgloss.Color("235"))
	stHelpBG  = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("250"))
	stSel     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Background(lipgloss.Color("237")).Padding(0, 1)
	stNorm    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Padding(0, 1)
	stMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	stDetail  = lipgloss.NewStyle().Foreground(lipgloss.Color("253")).Padding(1, 2)
	stBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63"))
)

func (m *model) clearHits() { m.hits = m.hits[:0] }

func (m *model) addHit(x, y, w, h int, action string) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m.hits = append(m.hits, hitRect{X: x, Y: y, W: w, H: h, Action: action})
}

func (m *model) hitTest(x, y int) string {
	for i := len(m.hits) - 1; i >= 0; i-- {
		r := m.hits[i]
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return r.Action
		}
	}
	return ""
}

func (m *model) currentEntries() []menuEntry {
	if m.screen == screenMode || m.tab == tabMode {
		return modeEntries(m.lang)
	}
	if m.tab == tabWizard {
		return wizTargetEntries(m.lang)
	}
	if m.tab == tabOps {
		return opsEntries(m.mode, m.lang)
	}
	if m.tab == tabSettings {
		return settingsEntries(m.lang)
	}
	return toolsEntries(m.mode, m.lang)
}

func (m *model) currentHelp() []helpChip {
	loc := l10n(m.lang)
	if m.screen == screenOutput {
		return loc.HelpOutput
	}
	if m.screen == screenMode || m.tab == tabMode {
		return loc.HelpModeBar
	}
	switch m.tab {
	case tabWizard:
		return loc.HelpWizard
	case tabOps:
		return loc.HelpOps
	case tabSettings:
		return loc.HelpSettings
	default:
		return loc.HelpTools
	}
}

func renderChip(c helpChip) string {
	return stChipKey.Render(c.Key) + stChipLab.Render(" "+c.Label+" ")
}

func (m *model) renderHeader() string {
	m.clearHits()
	loc := l10n(m.lang)
	w := max(40, m.width)

	title := stTitle.Render(" " + loc.App + " ")
	if m.hasRemote() {
		title += stChipKey.Render(" " + m.remoteLabel() + " ")
	}
	tabs := []struct {
		id, label string
	}{
		{tabWizard, loc.TabWizard},
		{tabTools, loc.TabTools},
		{tabOps, loc.TabOps},
		{tabSettings, loc.TabSettings},
		{tabMode, loc.TabMode},
	}
	var parts []string
	parts = append(parts, title)
	x := lipgloss.Width(title)
	for _, t := range tabs {
		lab := " " + t.label + " "
		var cell string
		if m.tab == t.id && m.screen != screenOutput {
			cell = stTabOn.Render(lab)
		} else {
			cell = stTabOff.Render(lab)
		}
		m.addHit(x, 0, lipgloss.Width(cell), 1, "tab:"+t.id)
		parts = append(parts, cell)
		x += lipgloss.Width(cell) + 1
		parts = append(parts, " ")
		x++
	}
	lang := "EN"
	if m.lang == langRU {
		lang = "RU"
	}
	langBadge := stChipKey.Render(" " + lang + " ")
	clock := stMuted.Render(time.Now().Format("15:04:05"))
	right := lipgloss.JoinHorizontal(lipgloss.Top, langBadge, " ", clock)
	m.addHit(w-lipgloss.Width(right), 0, lipgloss.Width(langBadge), 1, "lang")

	left := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	row := left + strings.Repeat(" ", gap) + right
	return stHeader.Width(w).Render(row)
}

func (m *model) renderHelpBar() string {
	w := max(40, m.width)
	chips := m.currentHelp()
	var rows []string
	var cur string
	curW := 0
	helpY := m.helpY
	x := 1
	for _, c := range chips {
		p := renderChip(c)
		pw := lipgloss.Width(p) + 1
		if curW > 0 && curW+pw > w-2 {
			rows = append(rows, cur)
			cur, curW = p, pw
			x = 1
			helpY++
		} else {
			if cur != "" {
				cur += " "
				curW++
				x++
			}
			// hit for chip is informational only
			_ = helpY
			cur += p
			curW += pw
			x += pw
		}
	}
	if cur != "" {
		rows = append(rows, cur)
	}
	bar := strings.Join(rows, "\n")
	return stHelpBG.Width(w).Render(bar)
}

func (m *model) renderSplit() string {
	w := max(40, m.width)
	h := max(8, m.height)
	headerH := 1
	helpH := lipgloss.Height(m.renderHelpBar())
	if helpH < 1 {
		helpH = 1
	}
	bodyH := h - headerH - helpH - 1
	if bodyH < 5 {
		bodyH = 5
	}
	leftW := w * 2 / 5
	if leftW < 24 {
		leftW = 24
	}
	if leftW > 48 {
		leftW = 48
	}
	rightW := w - leftW - 3
	if rightW < 20 {
		rightW = 20
		leftW = w - rightW - 3
	}

	entries := m.currentEntries()
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(entries) && len(entries) > 0 {
		m.cursor = len(entries) - 1
	}

	// left list — track absolute Y: header is y=0, body starts y=1
	var leftLines []string
	bodyY0 := 1
	for i, e := range entries {
		line := e.Title
		short := e.Short
		var row string
		if i == m.cursor {
			row = stSel.Width(leftW).Render(line)
			leftLines = append(leftLines, row)
			leftLines = append(leftLines, stSel.Width(leftW).Render("  "+short))
		} else {
			row = stNorm.Width(leftW).Render(line)
			leftLines = append(leftLines, row)
			leftLines = append(leftLines, stMuted.Width(leftW).Render("  "+short))
		}
		// each entry takes 2 rows
		m.addHit(0, bodyY0+i*2, leftW, 2, fmt.Sprintf("row:%d", i))
	}
	leftBody := strings.Join(leftLines, "\n")
	leftBody = lipgloss.NewStyle().Width(leftW).Height(bodyH).MaxHeight(bodyH).Render(leftBody)

	detail := ""
	if len(entries) > 0 {
		e := entries[m.cursor]
		detail = stDetail.Width(rightW-2).Render(
			stTitle.Render(e.Title) + "\n\n" + e.Detail,
		)
	}
	detail = stBorder.Width(rightW).Height(bodyH).MaxHeight(bodyH).Render(
		lipgloss.NewStyle().Width(rightW-2).Height(bodyH-2).Render(detail),
	)

	gap := " "
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBody, gap, detail)
}

func (m *model) renderOutput() string {
	w := max(40, m.width)
	h := max(8, m.height)
	help := m.renderHelpBar()
	helpH := lipgloss.Height(help)
	head := stTitle.Width(w).Render("Output")
	bodyH := h - lipgloss.Height(head) - helpH - 1
	if bodyH < 3 {
		bodyH = 3
	}
	body := lipgloss.NewStyle().Width(w-2).Height(bodyH).MaxHeight(bodyH).Render(m.output)
	used := lipgloss.Height(head) + lipgloss.Height(body) + helpH
	gap := h - used
	if gap < 0 {
		gap = 0
	}
	return head + "\n" + body + strings.Repeat("\n", gap) + help
}

func (m model) View() string {
	if m.quitting && m.result.action == "quit" {
		return stMuted.Render("bye") + "\n"
	}
	w, h := m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	// helpY for mouse
	helpH := 1
	m.helpY = h - helpH - 1

	if m.screen == screenOutput {
		return m.renderOutput()
	}
	if m.screen == screenWizard {
		return m.renderWizard()
	}
	header := m.renderHeader()
	body := m.renderSplit()
	help := m.renderHelpBar()
	used := lipgloss.Height(header) + lipgloss.Height(body) + lipgloss.Height(help)
	gap := h - used
	if gap < 0 {
		gap = 0
	}
	return header + "\n" + body + strings.Repeat("\n", gap) + help
}


// handleMouse uses layout geometry (not View-time hits) so coordinates match Ghostty.
func (m model) handleMouse(x, y int) (tea.Model, tea.Cmd) {
	w := max(40, m.width)
	_ = max(8, m.height)
	if y <= 0 {
		// header: lang is rightmost ~8 cells; tabs after title
		if x >= w-12 {
			return m.toggleLang()
		}
		// rough tab zones: title ~12, then 4 tabs ~12 each
		tx := x - 12
		if tx >= 0 {
			ti := tx / 12
			tabs := []string{tabWizard, tabTools, tabOps, tabSettings, tabMode}
			if ti >= 0 && ti < len(tabs) {
				m.tab = tabs[ti]
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
			}
		}
		return m, nil
	}
	if m.screen == screenOutput {
		return m, nil
	}
	// body starts at y=1; each entry 2 rows
	row := (y - 1) / 2
	ents := m.currentEntries()
	if row >= 0 && row < len(ents) {
		leftW := w * 2 / 5
		if leftW < 24 {
			leftW = 24
		}
		if leftW > 48 {
			leftW = 48
		}
		if x < leftW {
			m.cursor = row
		}
	}
	return m, nil
}
