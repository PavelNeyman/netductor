// Package stack is a thin orchestrator for netductor-owned systemd units and binaries.
package stack

import (
	"context"
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").CombinedOutput()
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
// Do NOT PromoteLastGood here — viewing Fleet/Stack must not swap binaries (caused 121↔135 loops).
func Collect() Status {
	rel := version.Running()
	st := Status{Release: rel, At: time.Now().Unix()}
	saveCurrent(rel)
	if b, err := os.ReadFile(filepath.Join(prevDir(), "VERSION")); err == nil {
		st.Prev = strings.TrimPrefix(strings.TrimSpace(string(b)), "v")
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

func attemptDir() string {
	return filepath.Join(paths.StateDir(), "stack", "attempt")
}

// snapshotBins copies node+tg (+ VERSION) into dir for rollback / last-good.
func snapshotBins(dir string) error {
	_ = os.MkdirAll(dir, 0o755)
	for _, src := range []string{"/usr/local/bin/netductor", "/usr/local/bin/netductor-tg"} {
		b, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(dir, filepath.Base(src)), b, 0o755)
	}
	ver := version.Running()
	if ver == "" {
		ver = version.Release
	}
	_ = os.WriteFile(filepath.Join(dir, "VERSION"), []byte(ver+"\n"), 0o644)
	return nil
}

// snapshotPrev is last-good (successful) snapshot — used by manual Rollback.
func snapshotPrev() error { return snapshotBins(prevDir()) }

// snapshotAttempt is pre-apply only — used if apply must abort mid-flight.
func snapshotAttempt() error { return snapshotBins(attemptDir()) }


// verNorm strips leading v.
func verNorm(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "v")
}

// verLess reports a < b for dotted numeric versions (0.9.121 < 0.9.133).
func verLess(a, b string) bool {
	a, b = verNorm(a), verNorm(b)
	if a == "" || b == "" {
		return false
	}
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(as) {
			fmt.Sscanf(as[i], "%d", &ai)
		}
		if i < len(bs) {
			fmt.Sscanf(bs[i], "%d", &bi)
		}
		if ai < bi {
			return true
		}
		if ai > bi {
			return false
		}
	}
	return false
}

func readDirVersion(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	if err != nil {
		return ""
	}
	return verNorm(string(b))
}

// PromoteLastGood restores prev/ when it is newer than running (fixes stuck 0.9.121 with prev 0.9.132).
func PromoteLastGood() bool {
	cur := verNorm(version.Running())
	prev := readDirVersion(prevDir())
	if prev == "" || cur == "" {
		return false
	}
	if !verLess(cur, prev) {
		return false
	}
	fmt.Fprintln(os.Stderr, "stack: promote last-good", prev, "over running", cur)
	if err := restoreFrom(prevDir()); err != nil {
		fmt.Fprintln(os.Stderr, "promote failed:", err)
		return false
	}
	// Never leave pre-apply snapshots around — they re-seed 0.9.121 loops on old apply code paths.
	_ = os.RemoveAll(attemptDir())
	notify.AlertOnce("stack:promote:"+prev, "⬆️ Stack promoted last-good <code>"+prev+"</code> (was <code>"+cur+"</code>)")
	return true
}

// Apply downloads node+tg for tag, snapshots prev, restarts core units, health-checks.
func Apply(tag string) error {
	return ApplyOpts(tag, false)
}

// ApplyOpts downloads node+tg; optional pre-backup; health + auto-rollback with TG alerts.
func stackPinPath() string {
	return filepath.Join(paths.StateDir(), "stack", "PIN")
}

// Pin blocks stack apply/promote until removed (stops oscillation while operator fixes disk).
func Pin(reason string) error {
	_ = os.MkdirAll(filepath.Dir(stackPinPath()), 0o755)
	if reason == "" {
		reason = "pinned"
	}
	return os.WriteFile(stackPinPath(), []byte(reason+"\n"+time.Now().UTC().Format(time.RFC3339)+"\n"), 0o644)
}

func Unpin() error { return os.Remove(stackPinPath()) }

func IsPinned() (bool, string) {
	b, err := os.ReadFile(stackPinPath())
	if err != nil {
		return false, ""
	}
	return true, strings.TrimSpace(string(b))
}

func ApplyOpts(tag string, noBackup bool) error {
	if ok, why := IsPinned(); ok {
		return fmt.Errorf("stack pinned — unpin first (netductor stack unpin): %s", why)
	}
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
	if !TryAcquireApplyLock(tag) {
		busy, cur, _ := ApplyInProgress()
		_ = busy
		return fmt.Errorf("apply already in progress: %s", cur)
	}
	defer ClearApplyLock()
	cur := verNorm(version.Running())
	want := verNorm(tag)
	if cur != "" && want != "" && verLess(want, cur) {
		return fmt.Errorf("refuse downgrade: running %s > target %s (use manual binary replace if intentional)", cur, want)
	}
	// Never keep attempt/ — old builds restored it on health fail and fought prev/ (121↔132 loops).
	_ = os.RemoveAll(attemptDir())
	if !noBackup {
		fmt.Fprintln(os.Stderr, "stack apply: pre-backup")
		if path, err := install.Backup(); err != nil {
			fmt.Fprintln(os.Stderr, "warn backup:", err)
		} else {
			fmt.Fprintln(os.Stderr, "backup:", path)
			_, _ = install.WaitForBackupPull(20 * time.Second)
		}
	}
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
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "reset-failed", "netductor-api", "netductor-telegram-bot").Run()
	_ = exec.Command("systemctl", "restart", "netductor-api").Run()
	_ = exec.Command("systemctl", "restart", "netductor-telegram-bot").Run()
	_ = exec.Command("systemctl", "try-restart", "netductor-redirect").Run()
	// Bot long-poll / restart is flaky for 10–20s — do NOT hard-fail on telegram-bot alone
	// (that caused auto-rollback to ancient prev like 0.9.121 after a successful binary replace).
	time.Sleep(12 * time.Second)
	st := Collect()
	var failed []string
	for _, u := range st.Units {
		if u.Unit == "netductor-api" && !u.OK {
			failed = append(failed, u.Unit+":"+u.Active)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintln(os.Stderr, "health soft-fail (api), retry restart:", failed)
		_ = exec.Command("systemctl", "reset-failed", "netductor-api", "netductor-telegram-bot").Run()
		_ = exec.Command("systemctl", "restart", "netductor-api", "netductor-telegram-bot").Run()
		time.Sleep(10 * time.Second)
		st = Collect()
		failed = nil
		for _, u := range st.Units {
			if u.Unit == "netductor-api" && !u.OK {
				failed = append(failed, u.Unit+":"+u.Active)
			}
		}
	}
	// Soft warn if bot still down — leave new binaries in place
	for _, u := range st.Units {
		if u.Unit == "netductor-telegram-bot" && !u.OK {
			fmt.Fprintln(os.Stderr, "warn: telegram-bot not active after apply (no rollback):", u.Active)
			notify.AlertOnce("stack:bot-warn:"+tag, "⚠️ Stack apply <code>"+tag+"</code>: telegram-bot="+u.Active+" (binaries kept)")
		}
	}
	if len(failed) > 0 {
		// Keep new binaries — auto-restore to attempt caused endless 0.9.121 loops when prev was newer.
		fmt.Fprintln(os.Stderr, "health failed (api), KEEPING new binaries:", failed)
		notify.AlertOnce("stack:health:"+tag, "⚠️ Stack apply <code>"+tag+"</code> health: "+strings.Join(failed, ", ")+" — binaries kept (no auto-downgrade)")
		_ = snapshotPrev() // still record as last-good if we got this far with new bins
		_ = os.RemoveAll(attemptDir())
		return fmt.Errorf("apply health fail %v (binaries kept at %s)", failed, tag)
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
	// Last-good = newly installed binaries (never keep ancient 0.9.121 as prev after success)
	_ = snapshotPrev()
	_ = os.RemoveAll(attemptDir())
	return nil
}

func restoreFrom(dir string) error {
	restored := 0
	for _, name := range []string{"netductor", "netductor-tg"} {
		src := filepath.Join(dir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		b, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile("/usr/local/bin/"+name, b, 0o755); err != nil {
			return err
		}
		restored++
	}
	if restored == 0 {
		return fmt.Errorf("no binaries in %s", dir)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "VERSION")); err == nil {
		ndupdate.WriteVERSION(strings.TrimSpace(string(b)))
		saveCurrent(strings.TrimSpace(string(b)))
	}
	_ = exec.Command("systemctl", "restart", "netductor-api").Run()
	_ = exec.Command("systemctl", "restart", "netductor-telegram-bot").Run()
	return nil
}

// Rollback restores last-good (prev/) binaries — never auto-called to ancient snapshots after success.
func Rollback() error {
	if err := restoreFrom(prevDir()); err != nil {
		return err
	}
	notify.AlertOnce("stack:rollback-manual", "↩️ Stack rollback restored last-good binaries")
	fmt.Fprintln(os.Stderr, "stack rollback done")
	return nil
}

// WatchdogOnce restarts core units that are failed/inactive.
func WatchdogOnce() {
	// Drop attempt/ always — must not outlive a successful newer install.
	if av := readDirVersion(attemptDir()); av != "" {
		rv := verNorm(version.Running())
		if rv == "" || verLess(av, rv) || av == rv {
			_ = os.RemoveAll(attemptDir())
		}
	}
	if ok, _ := IsPinned(); !ok {
		_ = PromoteLastGood()
	}
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
