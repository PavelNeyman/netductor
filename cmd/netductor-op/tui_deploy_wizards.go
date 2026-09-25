package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Legacy huh deploy removed (v0.8.94+). Framed Setup wizard only.

func runSetupWizard() {
	lang := detectLang()
	s := loadTUISettings()
	m := model{width: 120, height: 40, mode: modeOperator, lang: lang, tab: tabTools, cursor: 0}
	m.applySettings(s)
	m.startWizard()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func wizardPrimary()   { runSetupWizard() }
func wizardSecondary() { runSetupWizard() }
func wizardOpenWrt()   { runSetupWizard() }
func wizardNVR()       { runSetupWizard() }
func wizardAddons()    { runSetupWizard() }
