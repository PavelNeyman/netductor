package main

import (
	"bufio"
	"os"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type wizLogMsg struct{ Line string }
type wizDoneMsg struct {
	Text string
	OK   bool
}

var (
	wizLineCh <-chan string
	wizDoneCh <-chan wizDoneMsg
)

type wizProgStep struct {
	ID, Label string
	Done      bool
	Active    bool
}

func wizStepsForTarget(target wizTarget, lang tuiLang) []wizProgStep {
	ru := lang == langRU
	L := func(en, r string) string {
		if ru {
			return r
		}
		return en
	}
	switch target {
	case wizPrimary:
		// Core primary only — lampac/git/registry/TG are Add-ons checklist, not this wizard.
		return []wizProgStep{
			{ID: "keygen", Label: L("SSH key", "SSH-ключ"), Active: true},
			{ID: "pubkey", Label: L("Install pubkey", "Pubkey на VPS")},
			{ID: "download", Label: L("Download binary", "Скачать бинарь")},
			{ID: "install", Label: L("netductor install (core)", "netductor install (ядро)")},
			{ID: "harden", Label: L("SSH harden :52222", "SSH harden :52222")},
			{ID: "sni", Label: L("Reality SNI", "Reality SNI")},
			{ID: "fleet", Label: L("Fleet bootstrap + doctor", "Fleet + doctor")},
			{ID: "done", Label: L("Finish", "Готово")},
		}

	case wizSecondary:
		return []wizProgStep{
			{ID: "connect", Label: L("SSH to secondary", "SSH secondary"), Active: true},
			{ID: "provision", Label: L("Provision agent", "Provision agent")},
			{ID: "mtls", Label: L("mTLS enroll", "mTLS")},
			{ID: "done", Label: L("Finish", "Готово")},
		}
	case wizOpenWrt:
		return []wizProgStep{
			{ID: "ssh", Label: L("SSH to router", "SSH роутер"), Active: true},
			{ID: "agent", Label: L("Install edge agent", "Edge agent")},
			{ID: "net", Label: L("Network / Wi-Fi", "Сеть / Wi-Fi")},
			{ID: "enroll", Label: L("Enroll primary", "Enroll primary")},
			{ID: "done", Label: L("Finish", "Готово")},
		}
	default:
		return []wizProgStep{
			{ID: "run", Label: L("Running…", "Выполнение…"), Active: true},
			{ID: "done", Label: L("Finish", "Готово")},
		}
	}
}

func matchStepID(line string) string {
	l := strings.ToLower(line)
	switch {
	case strings.Contains(l, "generating ssh key"):
		return "keygen"
	case strings.Contains(l, "install ssh public key"):
		return "pubkey"
	case strings.Contains(l, "download netductor"):
		return "download"
	case strings.Contains(l, "telegram"):
		return "secrets"
	case strings.Contains(l, "netductor install"):
		return "install"
	case strings.Contains(l, "post-harden"):
		return "harden"
	case strings.Contains(l, "set sni"):
		return "sni"
	case strings.Contains(l, "fleet bootstrap"):
		return "fleet"
	case strings.Contains(l, "lampac"):
		return "lampac"
	case strings.Contains(l, "registry") || strings.Contains(l, "git bootstrap"):
		return "registry"
	case strings.Contains(l, "primary deploy done") || strings.Contains(l, "deploy done"):
		return "done"
	case strings.Contains(l, "provision"):
		return "provision"
	case strings.Contains(l, "mtls"):
		return "mtls"
	}
	return ""
}

func startWizardDeployCmd(m model) tea.Cmd {
	snap := m
	lineCh := make(chan string, 256)
	doneCh := make(chan wizDoneMsg, 1)
	wizLineCh = lineCh
	wizDoneCh = doneCh

	go func() {
		oldErr, oldOut := os.Stderr, os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			text := snap.runWizardApplyInTUI()
			close(lineCh)
			doneCh <- wizDoneMsg{Text: text, OK: !looksLikeError(text)}
			return
		}
		os.Stderr = w
		os.Stdout = w
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc := bufio.NewScanner(r)
			buf := make([]byte, 0, 64*1024)
			sc.Buffer(buf, 1024*1024)
			for sc.Scan() {
				lineCh <- sc.Text()
			}
		}()
		text := snap.runWizardApplyInTUI()
		_ = w.Close()
		os.Stderr, os.Stdout = oldErr, oldOut
		wg.Wait()
		close(lineCh)
		ok := !looksLikeError(text) && !strings.Contains(text, "exit status")
		low := strings.ToLower(text)
		if strings.Contains(text, "готов") || strings.Contains(low, "finished") ||
			strings.Contains(low, "provisioned") || strings.Contains(low, "deploy done") {
			ok = !strings.Contains(text, "exit status")
		}
		doneCh <- wizDoneMsg{Text: text, OK: ok}
	}()
	return listenWizardStream()
}

func listenWizardStream() tea.Cmd {
	return func() tea.Msg {
		if wizLineCh == nil {
			return wizDoneMsg{Text: "", OK: true}
		}
		line, ok := <-wizLineCh
		if ok {
			return wizLogMsg{Line: line + "\n"}
		}
		// channel closed — take done
		wizLineCh = nil
		if wizDoneCh != nil {
			d := <-wizDoneCh
			wizDoneCh = nil
			return d
		}
		return wizDoneMsg{OK: true}
	}
}

func looksLikeError(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "error") || strings.Contains(l, "failed") ||
		strings.Contains(l, "exit status") || strings.Contains(s, "required")
}

func (m *model) applyLogLine(line string) {
	id := matchStepID(line)
	if id == "" {
		return
	}
	for i := range m.wizSteps {
		if m.wizSteps[i].ID == id {
			m.wizSteps[i].Done = true
			m.wizSteps[i].Active = false
			if i+1 < len(m.wizSteps) && !m.wizSteps[i+1].Done {
				m.wizSteps[i+1].Active = true
			}
			return
		}
	}
}

func (m model) progressFraction() float64 {
	if len(m.wizSteps) == 0 {
		return 0
	}
	n := 0
	for _, s := range m.wizSteps {
		if s.Done {
			n++
		}
	}
	return float64(n) / float64(len(m.wizSteps))
}

func (m model) renderProgressBar(width int) string {
	if width < 8 {
		width = 8
	}
	frac := m.progressFraction()
	filled := int(frac * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pct := int(frac * 100)
	return bar + " " + itoa(pct) + "%"
}

func (m model) renderWizardStepsView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderProgressBar(minInt(40, max(12, width-4))))
	b.WriteString("\n\n")
	for _, s := range m.wizSteps {
		mark := "[ ]"
		if s.Done {
			mark = "[✓]"
		} else if s.Active {
			mark = "[…]"
		}
		b.WriteString(mark)
		b.WriteString(" ")
		b.WriteString(s.Label)
		b.WriteString("\n")
	}
	return b.String()
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
