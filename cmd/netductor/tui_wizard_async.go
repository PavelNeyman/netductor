package main

import (
	"bytes"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type wizLogMsg struct{ Line string }
type wizDoneMsg struct {
	Text string
	OK   bool
}

// startWizardDeployCmd runs deploy off the UI thread; result stays inside framed TUI.
func startWizardDeployCmd(m model) tea.Cmd {
	// snapshot fields so UI can still render while we work
	snap := m
	snap.wizRunning = true
	return func() tea.Msg {
		// capture stderr from deploy packages (they print progress there)
		oldErr := os.Stderr
		oldOut := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			text := snap.runWizardApplyInTUI()
			ok := !looksLikeError(text)
			return wizDoneMsg{Text: text, OK: ok}
		}
		os.Stderr = w
		os.Stdout = w
		done := make(chan string, 1)
		go func() {
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			done <- buf.String()
		}()

		text := snap.runWizardApplyInTUI()
		_ = w.Close()
		os.Stderr = oldErr
		os.Stdout = oldOut
		captured := <-done
		combined := strings.TrimSpace(captured)
		if text != "" {
			if combined != "" {
				combined += "\n"
			}
			combined += text
		}
		ok := !looksLikeError(combined) && !strings.Contains(strings.ToLower(combined), "failed") &&
			!strings.Contains(combined, "exit status")
		// soft: if explicit success phrases
		if strings.Contains(combined, "готов") || strings.Contains(strings.ToLower(combined), "finished") ||
			strings.Contains(strings.ToLower(combined), "provisioned") {
			ok = !strings.Contains(combined, "exit status")
		}
		return wizDoneMsg{Text: combined, OK: ok}
	}
}

func looksLikeError(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "error") || strings.Contains(l, "failed") ||
		strings.Contains(l, "exit status") || strings.Contains(s, "required")
}
