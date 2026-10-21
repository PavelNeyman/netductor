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
	hint := "Enter · Esc · PgUp/PgDn"
	if ru {
		hint = "Enter · Esc · PgUp/PgDn"
	}
	if m.catLevel == 0 {
		title := "Day-2 catalog (Groups)"
		if ru {
			title = "Day-2 каталог (группы)"
		}
		b.WriteString(stTitle.Render(title) + "\n")
		b.WriteString(stMuted.Render(hint) + "\n\n")
		groups := m.catalogGroups()
		headerLines := 3
		vis := bodyH - headerLines
		if vis < 3 {
			vis = 3
		}
		m.clampCatOffset(len(groups), vis)
		start := m.catOffset
		end := start + vis
		if end > len(groups) {
			end = len(groups)
		}
		for i := start; i < end; i++ {
			g := groups[i]
			lab := g.LabelEN
			if ru {
				lab = g.LabelRU
			}
			if i == m.catGroupIdx {
				b.WriteString(stSel.Render("› "+lab) + " " + stMuted.Render(strings.Join(g.Sections, ",")) + "\n")
			} else {
				line := fmt.Sprintf("  %s  %s", lab, stMuted.Render("["+strings.Join(g.Sections, ",")+"]"))
				b.WriteString(stNorm.Render(line) + "\n")
			}
		}
		if start > 0 || end < len(groups) {
			b.WriteString(stMuted.Render(fmt.Sprintf("  [%d–%d / %d]", start+1, end, len(groups))) + "\n")
		}
	} else {
		gs := m.catalogGroups()
		g := gs[m.catGroupIdx]
		lab := g.LabelEN
		if ru {
			lab = g.LabelRU
		}
		b.WriteString(stTitle.Render(lab) + "\n")
		b.WriteString(stMuted.Render(hint) + "\n\n")
		acts := m.catalogActions()
		if len(acts) == 0 {
			b.WriteString(stMuted.Render("(empty)") + "\n")
		}
		headerLines := 3
		vis := bodyH - headerLines
		if vis < 3 {
			vis = 3
		}
		// actions may take 2 lines when selected — use 1 line count for offset
		m.clampCatOffset(len(acts), vis)
		start := m.catOffset
		end := start + vis
		if end > len(acts) {
			end = len(acts)
		}
		for i := start; i < end; i++ {
			a := acts[i]
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
		if start > 0 || end < len(acts) {
			b.WriteString(stMuted.Render(fmt.Sprintf("  [%d–%d / %d]", start+1, end, len(acts))) + "\n")
		}
	}
	if m.output != "" {
		// show last few lines of action output below list
		lines := strings.Split(m.output, "\n")
		tail := lines
		if len(tail) > 8 {
			tail = tail[len(tail)-8:]
		}
		b.WriteString("\n" + stMuted.Render("---") + "\n" + strings.Join(tail, "\n"))
	}
	return b.String()
}

func (m *model) clampCatOffset(n, vis int) {
	if vis < 1 {
		vis = 1
	}
	if n <= vis {
		m.catOffset = 0
		return
	}
	// keep selection visible
	idx := m.catGroupIdx
	if m.catLevel == 1 {
		idx = m.catActionIdx
	}
	if idx < m.catOffset {
		m.catOffset = idx
	}
	if idx >= m.catOffset+vis {
		m.catOffset = idx - vis + 1
	}
	maxOff := n - vis
	if m.catOffset < 0 {
		m.catOffset = 0
	}
	if m.catOffset > maxOff {
		m.catOffset = maxOff
	}
}

func (m *model) updateCatalog(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc", "q":
			if m.catLevel == 1 {
				m.catLevel = 0
				m.catActionIdx = 0
				m.catOffset = 0
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
		case "pgup", "ctrl+u":
			page := 8
			if m.catLevel == 0 {
				m.catGroupIdx -= page
				if m.catGroupIdx < 0 {
					m.catGroupIdx = 0
				}
			} else {
				m.catActionIdx -= page
				if m.catActionIdx < 0 {
					m.catActionIdx = 0
				}
			}
		case "pgdown", "ctrl+d":
			page := 8
			if m.catLevel == 0 {
				m.catGroupIdx += page
				if m.catGroupIdx >= len(m.catalogGroups()) {
					m.catGroupIdx = len(m.catalogGroups()) - 1
				}
				if m.catGroupIdx < 0 {
					m.catGroupIdx = 0
				}
			} else {
				acts := m.catalogActions()
				m.catActionIdx += page
				if m.catActionIdx >= len(acts) {
					m.catActionIdx = len(acts) - 1
				}
				if m.catActionIdx < 0 {
					m.catActionIdx = 0
				}
			}
		case "home":
			if m.catLevel == 0 {
				m.catGroupIdx = 0
			} else {
				m.catActionIdx = 0
			}
		case "end":
			if m.catLevel == 0 {
				m.catGroupIdx = len(m.catalogGroups()) - 1
				if m.catGroupIdx < 0 {
					m.catGroupIdx = 0
				}
			} else {
				acts := m.catalogActions()
				m.catActionIdx = len(acts) - 1
				if m.catActionIdx < 0 {
					m.catActionIdx = 0
				}
			}
		case "enter":
			if m.catLevel == 0 {
				m.catLevel = 1
				m.catActionIdx = 0
				m.catOffset = 0
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
	switch a.ID {
	case "vpn-policy-get":
		m.startActionForm("vpn-policy-get")
		return
	case "vpn-policy-set":
		m.startActionForm("vpn-policy-set")
		return
	case "edge-policy-get":
		m.startActionForm("edge-policy-get")
		return
	case "edge-policy-set":
		m.startActionForm("edge-policy-set")
		return
	case "services":
		m.output = m.runNetductor("services", "list")
		return
	case "policy-apply":
		m.output = m.runNetductor("policy", "apply")
		return
	}
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
