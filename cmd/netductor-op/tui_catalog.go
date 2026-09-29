package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/PavelNeyman/netductor/internal/opcatalog"
)

// Day-2 catalog browser (same Groups + Actions as Web/TG).
// Enter group → list actions → run CLI on remote when possible.

func (m *model) openCatalog() {
	m.screen = screenCatalog
	m.catGroupIdx = 0
	m.catActionIdx = 0
	m.catLevel = 0 // 0 groups, 1 actions
	m.output = ""
}

func (m *model) catalogGroups() []opcatalog.Group {
	return opcatalog.Groups()
}

func (m *model) catalogActions() []opcatalog.Action {
	gs := m.catalogGroups()
	if m.catGroupIdx < 0 || m.catGroupIdx >= len(gs) {
		return nil
	}
	lang := "en"
	if m.lang == langRU {
		lang = "ru"
	}
	_ = lang
	return opcatalog.ActionsInGroup(gs[m.catGroupIdx].ID, opcatalog.SurfTUI)
}

func (m *model) renderCatalog(bodyH, w int) string {
	ru := m.lang == langRU
	var b strings.Builder
	if m.catLevel == 0 {
		title := "Day-2 catalog (Groups)"
		if ru {
			title = "Day-2 каталог (группы)"
		}
		b.WriteString(stTitle.Render(title) + "\n")
		b.WriteString(stMuted.Render("Enter → actions · Esc → menu") + "\n\n")
		for i, g := range m.catalogGroups() {
			lab := g.LabelEN
			if ru {
				lab = g.LabelRU
			}
			line := fmt.Sprintf("  %s  %s", lab, stMuted.Render("["+strings.Join(g.Sections, ",")+"]"))
			if i == m.catGroupIdx {
				b.WriteString(stSel.Render("› "+lab) + " " + stMuted.Render(strings.Join(g.Sections, ",")) + "\n")
			} else {
				b.WriteString(stNorm.Render(line) + "\n")
			}
		}
	} else {
		gs := m.catalogGroups()
		g := gs[m.catGroupIdx]
		lab := g.LabelEN
		if ru {
			lab = g.LabelRU
		}
		b.WriteString(stTitle.Render(lab) + "\n")
		b.WriteString(stMuted.Render("Enter → run · Esc → groups") + "\n\n")
		acts := m.catalogActions()
		if len(acts) == 0 {
			b.WriteString(stMuted.Render("(empty)") + "\n")
		}
		for i, a := range acts {
			label := a.Label(map[bool]string{true: "ru", false: "en"}[ru])
			cli := a.CLI
			if cli == "" || cli == "—" {
				cli = a.Method + " " + a.Path
			}
			if i == m.catActionIdx {
				b.WriteString(stSel.Render("› "+label) + "\n  " + stMuted.Render(cli) + "\n")
			} else {
				b.WriteString(stNorm.Render("  "+label) + "\n")
			}
		}
	}
	if m.output != "" {
		b.WriteString("\n" + stMuted.Render("---") + "\n" + m.output)
	}
	return b.String()
}

func (m *model) updateCatalog(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.catLevel == 1 {
				m.catLevel = 0
				m.catActionIdx = 0
				return m, nil
			}
			m.screen = screenMenu
			return m, nil
		case "up", "k":
			if m.catLevel == 0 {
				if m.catGroupIdx > 0 {
					m.catGroupIdx--
				}
			} else if m.catActionIdx > 0 {
				m.catActionIdx--
			}
		case "down", "j":
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
		case "enter":
			if m.catLevel == 0 {
				m.catLevel = 1
				m.catActionIdx = 0
				return m, nil
			}
			acts := m.catalogActions()
			if m.catActionIdx >= 0 && m.catActionIdx < len(acts) {
				m.runCatalogAction(acts[m.catActionIdx])
			}
		}
	}
	return m, nil
}

func (m *model) runCatalogAction(a opcatalog.Action) {
	cli := strings.TrimSpace(a.CLI)
	if cli == "" || cli == "—" {
		m.output = fmt.Sprintf("%s %s\n(no CLI mapping — use Web Control or session API)", a.Method, a.Path)
		return
	}
	args := strings.Fields(cli)
	out := m.runNetductor(args...)
	if len(out) > 4000 {
		out = out[len(out)-4000:]
	}
	m.output = out
}
