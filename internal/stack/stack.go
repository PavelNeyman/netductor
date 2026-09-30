// Package stack is a thin orchestrator for netductor-owned systemd units and binaries.
package stack

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/notify"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// UnitSpec is one managed service.
type UnitSpec struct {
	Unit   string `json:"unit"`
	Binary string `json:"binary,omitempty"` // empty = no binary to swap
	Role   string `json:"role"`            // core | optional
}

// PrimaryUnits — order matters for start (deps first not strictly required; systemd handles).
var PrimaryUnits = []UnitSpec{
	{Unit: "netductor-api", Binary: "/usr/local/bin/netductor", Role: "core"},
	{Unit: "netductor-telegram-bot", Binary: "/usr/local/bin/netductor-tg", Role: "core"},
	{Unit: "netductor-redirect", Binary: "/usr/local/bin/netductor", Role: "optional"},
	{Unit: "sing-box", Binary: "/usr/local/bin/sing-box", Role: "core"},
	{Unit: "blocky", Binary: "", Role: "core"},
	{Unit: "netductor-backup.timer", Binary: "", Role: "optional"},
}

type UnitStatus struct {
	Unit    string `json:"unit"`
	Active  string `json:"active"`
	Sub     string `json:"sub,omitempty"`
	Binary  string `json:"binary,omitempty"`
	Version string `json:"version,omitempty"`
	OK      bool   `json:"ok"`
}

type Status struct {
	Release string       `json:"release"`
	Units   []UnitStatus `json:"units"`
	Prev    string       `json:"prev_release,omitempty"`
	At      int64        `json:"ts"`
}

func unitState(unit string) (active, sub string) {
	out, _ := exec.Command("systemctl", "show", unit, "-p", "ActiveState", "-p", "SubState", "--value").CombinedOutput()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) >= 1 {
		active = strings.TrimSpace(lines[0])
	}
	if len(lines) >= 2 {
		sub = strings.TrimSpace(lines[1])
	}
	return
}

func binVersion(path string) string {
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return "missing"
	}
	out, err := exec.Command(path, "version").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}


func FormatHTML(st Status) string {
	var b strings.Builder
	b.WriteString("🧱 <b>Stack</b>\n")
	b.WriteString("release <code>" + st.Release + "</code>")
	if st.Prev != "" {
		b.WriteString(" · prev <code>" + st.Prev + "</code>")
	}
	b.WriteString("\n<table bordered striped>\n<tr><th>unit</th><th>state</th><th>ok</th></tr>\n")
	for _, u := range st.Units {
		mark := "✅"
		if !u.OK {
			mark = "❌"
		}
		b.WriteString("<tr><td>" + u.Unit + "</td><td><code>" + u.Active)
		if u.Sub != "" {
			b.WriteString("/" + u.Sub)
		}
		b.WriteString("</code></td><td>" + mark + "</td></tr>\n")
	}
	b.WriteString("</table>")
	return b.String()
}

func prevDir() string {
	return filepath.Join(paths.StateDir(), "stack", "prev")
}

func curMetaPath() string {
	return filepath.Join(paths.StateDir(), "stack", "current.json")
}

// Collect status of managed units.
func Collect() Status {
	st := Status{Release: version.Release, At: time.Now().Unix()}
	if b, err := os.ReadFile(filepath.Join(prevDir(), "VERSION")); err == nil {
		st.Prev = strings.TrimSpace(string(b))
	}
	for _, u := range PrimaryUnits {
		active, sub := unitState(u.Unit)
		ok := active == "active" || active == "activating"
		// timers report active when waiting
		if strings.HasSuffix(u.Unit, ".timer") && (active == "active" || sub == "waiting") {
			ok = true
		}
		// redirect may be inactive without LE — not hard fail
		if u.Unit == "netductor-redirect" && active == "inactive" {
			ok = true
		}
		st.Units = append(st.Units, UnitStatus{
			Unit: u.Unit, Active: active, Sub: sub, Binary: u.Binary,
			Version: binVersion(u.Binary), OK: ok,
		})
	}
	return st
}

func saveCurrent(tag string) {
	_ = os.MkdirAll(filepath.Dir(curMetaPath()), 0o755)
	raw, _ := json.MarshalIndent(map[string]any{"release": tag, "at": time.Now().UTC().Format(time.RFC3339)}, "", "  ")
	_ = os.WriteFile(curMetaPath(), append(raw, '\n'), 0o644)
}

// snapshotPrev copies current node+tg binaries aside for rollback.
func snapshotPrev() error {
	dir := prevDir()
	_ = os.MkdirAll(dir, 0o755)
	for _, src := range []string{"/usr/local/bin/netductor", "/usr/local/bin/netductor-tg"} {
		b, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(dir, filepath.Base(src)), b, 0o755)
	}
	_ = os.WriteFile(filepath.Join(dir, "VERSION"), []byte(version.Release+"\n"), 0o644)
	return nil
}

// Apply downloads node+tg for tag, snapshots prev, restarts core units, health-checks.
func Apply(tag string) error {
	return ApplyOpts(tag, false)
}

// ApplyOpts downloads node+tg; optional pre-backup; health + auto-rollback with TG alerts.
func ApplyOpts(tag string, noBackup bool) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		var err error
		tag, err = ndupdate.LatestReleaseTag()
		if err != nil {
			return err
		}
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	if !noBackup {
		fmt.Fprintln(os.Stderr, "stack apply: pre-backup")
		if path, err := install.Backup(); err != nil {
			fmt.Fprintln(os.Stderr, "warn backup:", err)
		} else {
			fmt.Fprintln(os.Stderr, "backup:", path)
			_, _ = install.WaitForBackupPull(20 * time.Second)
		}
	}
	_ = snapshotPrev()
	fmt.Fprintln(os.Stderr, "stack apply:", tag)
	notify.AlertOnce("stack:apply:"+tag, "⬆️ Stack apply <code>"+tag+"</code> started")
	if err := ndupdate.DownloadReleaseAsset(tag, "node", "/usr/local/bin/netductor"); err != nil {
		notify.AlertOnce("stack:apply-fail:"+tag, "🔴 Stack apply failed (node): "+err.Error())
		return fmt.Errorf("node: %w", err)
	}
	if err := ndupdate.DownloadReleaseAsset(tag, "tg", "/usr/local/bin/netductor-tg"); err != nil {
		notify.AlertOnce("stack:apply-fail:"+tag, "🔴 Stack apply failed (tg): "+err.Error())
		return fmt.Errorf("tg: %w", err)
	}
	ndupdate.WriteVERSION(tag)
	saveCurrent(tag)
	_ = exec.Command("systemctl", "restart", "netductor-api").Run()
	_ = exec.Command("systemctl", "restart", "netductor-telegram-bot").Run()
	_ = exec.Command("systemctl", "try-restart", "netductor-redirect").Run()
	time.Sleep(2 * time.Second)
	st := Collect()
	var failed []string
	for _, u := range st.Units {
		if u.Unit == "netductor-api" || u.Unit == "netductor-telegram-bot" {
			if !u.OK {
				failed = append(failed, u.Unit+":"+u.Active)
			}
		}
	}
	if len(failed) > 0 {
		fmt.Fprintln(os.Stderr, "health failed, rolling back:", failed)
		notify.AlertOnce("stack:rollback:"+tag, "↩️ Stack auto-rollback after failed apply <code>"+tag+"</code>: "+strings.Join(failed, ", "))
		if err := Rollback(); err != nil {
			return fmt.Errorf("apply health fail %v; rollback: %w", failed, err)
		}
		return fmt.Errorf("apply health fail %v; rolled back", failed)
	}
	// queue secondary agents to same release
	secN := 0
	for _, d := range secondary.List() {
		if secondary.Online(d, 2*time.Minute) {
			if err := secondary.EnqueueCmd(d.ID, "upgrade:"+tag); err == nil {
				secN++
			}
		}
	}
	msg := "✅ Stack apply ok <code>" + tag + "</code>"
	if secN > 0 {
		msg += fmt.Sprintf(" · secondary upgrade queued=%d", secN)
	}
	notify.AlertOnce("stack:apply-ok:"+tag, msg)
	fmt.Fprintln(os.Stderr, "stack apply ok", tag, "secondary_queued", secN)
	return nil
}

// Rollback restores prev binaries and restarts.
func Rollback() error {
	dir := prevDir()
	for _, name := range []string{"netductor", "netductor-tg"} {
		src := filepath.Join(dir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		b, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		dest := "/usr/local/bin/" + name
		if err := os.WriteFile(dest, b, 0o755); err != nil {
			return err
		}
	}
	if b, err := os.ReadFile(filepath.Join(dir, "VERSION")); err == nil {
		ndupdate.WriteVERSION(strings.TrimSpace(string(b)))
		saveCurrent(strings.TrimSpace(string(b)))
	}
	_ = exec.Command("systemctl", "restart", "netductor-api").Run()
	_ = exec.Command("systemctl", "restart", "netductor-telegram-bot").Run()
	notify.AlertOnce("stack:rollback-manual", "↩️ Stack rollback restored prev binaries")
	fmt.Fprintln(os.Stderr, "stack rollback done")
	return nil
}

// WatchdogOnce restarts core units that are failed/inactive.
func WatchdogOnce() {
	var restarted []string
	for _, u := range PrimaryUnits {
		if u.Role != "core" {
			continue
		}
		active, sub := unitState(u.Unit)
		need := false
		if u.Unit == "sing-box" || u.Unit == "blocky" {
			need = active == "failed" || sub == "auto-restart"
		} else if u.Unit != "netductor-redirect" {
			need = active == "failed" || active == "inactive"
		}
		if need {
			_ = exec.Command("systemctl", "try-restart", u.Unit).Run()
			restarted = append(restarted, u.Unit)
		}
	}
	if len(restarted) > 0 {
		notify.AlertOnce("stack:watchdog:"+strings.Join(restarted, ","),
			"🔧 Stack watchdog restarted: <code>"+strings.Join(restarted, ", ")+"</code>")
	}
}

// InstallWatchdogTimer installs a short periodic unit that runs `netductor stack watchdog`.
func InstallWatchdogTimer() error {
	bin := "/usr/local/bin/netductor"
	service := fmt.Sprintf(`[Unit]
Description=Netductor stack watchdog (one-shot)
After=network-online.target

[Service]
Type=oneshot
ExecStart=%s stack watchdog
Nice=10
`, bin)
	timer := `[Unit]
Description=Netductor stack watchdog every 5 min

[Timer]
OnBootSec=2min
OnUnitActiveSec=5min
Persistent=true

[Install]
WantedBy=timers.target
`
	if err := os.WriteFile("/etc/systemd/system/netductor-stack-watchdog.service", []byte(service), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("/etc/systemd/system/netductor-stack-watchdog.timer", []byte(timer), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "--now", "netductor-stack-watchdog.timer").Run()
	return nil
}
