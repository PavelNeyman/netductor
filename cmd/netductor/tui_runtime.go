package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/PavelNeyman/netductor/internal/nodes"
)

func runBubbleSession(mode runMode, startMenu bool, cliHost, cliUser, cliKey, cliPass string) tuiResult {
	w, h := 120, 40
	lang := detectLang()
	s := loadTUISettings()
	rh, ru := loadRemoteFromEnv()
	if rh != "" {
		s.RemoteHost = rh
	}
	if ru != "" {
		s.RemoteUser = ru
	}
	m := model{width: w, height: h, mode: mode, lang: lang, tab: tabMode, cursor: 0}
	m.applySettings(s)
	if cliHost != "" {
		m.remoteHost = cliHost
	}
	if cliUser != "" {
		m.remoteUser = cliUser
	}
	if cliKey != "" {
		m.remoteKey = cliKey
	}
	if cliPass != "" {
		m.remotePassword = cliPass
	}
	if startMenu && mode != "" {
		m.screen = screenMenu
		m.tab = tabTools
	} else {
		m.screen = screenMode
		m.tab = tabMode
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	final, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return tuiResult{action: "quit"}
	}
	fm, ok := final.(model)
	if !ok {
		return tuiResult{action: "quit"}
	}
	return fm.result
}

func runHostnameForm() {
	cur := ""
	if b, err := os.ReadFile("/etc/netductor/node_id"); err == nil {
		cur = strings.TrimSpace(string(b))
	}
	name := cur
	ok := false
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Hostname").Description("Format nd-<role>-<marker> · core|edge|lab · e.g. nd-core-nl01 · a-z0-9- only").Value(&name),
			huh.NewConfirm().Title("Apply hostname on this host?").Affirmative("Yes").Negative("Skip").Value(&ok),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		return
	}
	name = nodes.NormalizeHostname(name)
	if name == "" {
		fmt.Println("empty name, skip")
		return
	}
	_ = os.MkdirAll("/etc/netductor", 0o755)
	_ = os.WriteFile("/etc/netductor/node_id", []byte(name+string([]byte{10})), 0o644)
	_ = os.WriteFile("/etc/hostname", []byte(name+string([]byte{10})), 0o644)
	_ = exec.Command("hostnamectl", "set-hostname", name).Run()
	ip := ""
	if b, err := os.ReadFile("/etc/netductor/public_ip"); err == nil {
		ip = strings.TrimSpace(string(b))
	}
	_ = nodes.SelfRegisterLocal(name, "core", ip)
	fmt.Println("hostname:", name)
}
