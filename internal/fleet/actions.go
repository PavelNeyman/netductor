package fleet

import (
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/channels"
)

// Action is one operator-facing "needs attention" item for Status / digest.
type Action struct {
	ID       string `json:"id"`
	Severity string `json:"severity"` // critical|warn|info
	Title    string `json:"title"`
	TitleRU  string `json:"title_ru,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// ActionRequired derives a short actionable list from live digest + channel path.
func ActionRequired() []Action {
	d := Build()
	var out []Action
	add := func(a Action) { out = append(out, a) }

	for _, s := range d.Secondaries {
		if !s.Online {
			add(Action{
				ID: "secondary-offline:" + s.ID, Severity: "critical",
				Title:   fmt.Sprintf("Secondary offline: %s", s.Name),
				TitleRU: fmt.Sprintf("Secondary offline: %s", s.Name),
				Hint:    "check agent / hoster console",
			})
		} else if !s.Uplink {
			add(Action{
				ID: "secondary-uplink:" + s.ID, Severity: "critical",
				Title:   fmt.Sprintf("Uplink down: %s", s.Name),
				TitleRU: fmt.Sprintf("Uplink down: %s", s.Name),
				Hint:    "channel / mux / primary:443 from secondary",
			})
		}
	}
	if d.BackupAgeHours != nil && *d.BackupAgeHours > 48 {
		add(Action{
			ID: "backup-age", Severity: "warn",
			Title:   fmt.Sprintf("Backup age %.0fh (>48h)", *d.BackupAgeHours),
			TitleRU: fmt.Sprintf("Возраст backup %.0fh (>48ч)", *d.BackupAgeHours),
			Hint:    "netductor backup / stack apply pre-backup",
		})
	}
	if v := strings.ToLower(d.BackupVerify); v != "" && (strings.Contains(v, "fail") || strings.Contains(v, "error")) {
		add(Action{
			ID: "backup-verify", Severity: "warn",
			Title: "Backup verify failed", TitleRU: "Backup verify не прошёл",
			Hint: d.BackupVerify,
		})
	} else if d.BackupVerify == "" {
		add(Action{
			ID: "backup-verify-missing", Severity: "info",
			Title: "No backup verify yet", TitleRU: "Backup verify ещё не запускали",
			Hint: "netductor backup verify · timer InstallBackupVerifyTimer",
		})
	} else if age := backupVerifyAgeDays(d.BackupVerify); age >= 14 {
		add(Action{
			ID: "backup-verify-stale", Severity: "warn",
			Title:   fmt.Sprintf("Backup verify %dd ago", age),
			TitleRU: fmt.Sprintf("Backup verify %d дн. назад", age),
			Hint:    "weekly timer or: netductor backup verify",
		})
	}
	if d.RecoverDrillDays != nil && *d.RecoverDrillDays > 90 {
		add(Action{
			ID: "recover-drill", Severity: "info",
			Title:   fmt.Sprintf("Recover drill %dd ago", *d.RecoverDrillDays),
			TitleRU: fmt.Sprintf("Recover drill %d дн. назад", *d.RecoverDrillDays),
			Hint:    "netductor backup drill-mark after a successful drill",
		})
	}
	for _, p := range channels.CollectPathE2E() {
		if p.Online && !p.PathOK {
			add(Action{
				ID: "path-e2e:" + p.SecondaryID, Severity: "warn",
				Title:   fmt.Sprintf("Path e2e weak: %s", p.Name),
				TitleRU: fmt.Sprintf("Path e2e слабо: %s", p.Name),
				Hint:    p.Note,
			})
		}
	}
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

// FormatActionsHTML for TG Status.
func FormatActionsHTML(ru bool) string {
	acts := ActionRequired()
	if len(acts) == 0 {
		if ru {
			return "✅ <b>Внимание:</b> нет критичных пунктов"
		}
		return "✅ <b>Action required:</b> none"
	}
	var b strings.Builder
	if ru {
		b.WriteString("⚠️ <b>Требует внимания</b>\n")
	} else {
		b.WriteString("⚠️ <b>Action required</b>\n")
	}
	for _, a := range acts {
		icon := "ℹ️"
		switch a.Severity {
		case "critical":
			icon = "🔴"
		case "warn":
			icon = "🟡"
		}
		title := a.Title
		if ru && a.TitleRU != "" {
			title = a.TitleRU
		}
		b.WriteString(fmt.Sprintf("%s %s\n", icon, title))
		if a.Hint != "" {
			b.WriteString(fmt.Sprintf("   <i>%s</i>\n", a.Hint))
		}
	}
	return b.String()
}

// backupVerifyAgeDays parses last.txt second line RFC3339; -1 if unknown.
func backupVerifyAgeDays(raw string) int {
	lines := strings.Split(raw, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "ok ") || strings.HasPrefix(ln, "fail") {
			continue
		}
		if ts, err := time.Parse(time.RFC3339, ln); err == nil {
			return int(time.Since(ts).Hours() / 24)
		}
	}
	return -1
}
