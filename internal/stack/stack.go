// Package stack is a thin orchestrator.
// Policy: no spontaneous binary upgrade/rollback. Apply/Rollback/Heal/Promote only on explicit CLI/API/TG action.
// Package stack is a thin orchestrator for netductor-owned systemd units and binaries.
//
// Simplified model (after 0.9.137+):
//
//	Single writer for primary node+tg binaries: Apply / ApplyOpts only.
//	ScheduleApply → systemd-run → Apply (API/TG must not apply in-process).
//	prev/     = last successful apply only (manual Rollback target).
//	attempt/  = removed (caused version oscillation).
//	Watchdog  = restart failed units only (never swaps binaries).
//	Promote   = explicit CLI only; verifies prev binary matches VERSION.
//	Pin       = blocks Apply and Promote while stabilizing.
package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/cleanup"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/version"
)

// UnitSpec is one managed service.
type UnitSpec struct {
	Unit   string `json:"unit"`
	Binary string `json:"binary,omitempty"` // empty = no binary to swap
	Role   string `json:"role"`             // core | optional
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

// SecondaryUnits — thin RU entry (no control-plane units).
var SecondaryUnits = []UnitSpec{
	{Unit: "sing-box", Binary: "/usr/local/bin/sing-box", Role: "core"},
	{Unit: "netductor-secondary-agent", Binary: "/usr/local/bin/netductor", Role: "core"},
	{Unit: "nd-wss-sp-client", Binary: "", Role: "optional"},
	{Unit: "nd-wss-ps-server", Binary: "", Role: "optional"},
	{Unit: "wg-quick@nd-svc-sp", Binary: "", Role: "optional"},
	{Unit: "wg-quick@nd-svc-ps", Binary: "", Role: "optional"},
}

func managedUnits() []UnitSpec {
	active, _ := unitState("netductor-secondary-agent")
	if active == "active" || active == "activating" {
		return SecondaryUnits
	}
	// secondary role without agent yet: no primary API
	pa, _ := unitState("netductor-api")
	if pa != "active" && pa != "activating" {
		st, _ := os.Stat("/etc/netductor/secrets/secondary_agent_token")
		if st == nil {
			st, _ = os.Stat("/var/lib/netductor/secondary/bundle.json")
		}
		if st != nil {
			return SecondaryUnits
		}
	}
	return PrimaryUnits
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

// apiHTTPHealthy probes local control API (best-effort).
// waitAPIHealthy polls /healthz up to timeout (R15: avoid fixed long sleep looking like a hang).
func waitAPIHealthy(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if apiHTTPHealthy() {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return apiHTTPHealthy()
}

func apiHTTPHealthy() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	for _, u := range []string{
		"http://127.0.0.1:8787/healthz",
		"http://127.0.0.1:8787/health",
		"http://127.0.0.1:8787/api/health",
	} {
		resp, err := client.Get(u)
		if err != nil {
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return true
		}
	}
	return false
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
	if st.Prev != "" && st.Release != "" && verLess(st.Release, st.Prev) {
		b.WriteString("\n⚠️ <b>prev newer than running</b> — <code>netductor stack heal</code> or re-apply")
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
	for _, u := range managedUnits() {
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
		// secondary optional path units may be absent on partial bootstrap
		if u.Role == "optional" && (active == "inactive" || active == "unknown" || active == "") {
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

// PromoteLastGood restores prev/ when it is strictly newer than running AND prev binaries match VERSION.
// Safe for explicit CLI; Watchdog does NOT call this by default.
func PromoteLastGood() bool {
	if ok, _ := IsPinned(); ok {
		return false
	}
	cur := verNorm(version.Running())
	prev := readDirVersion(prevDir())
	if prev == "" || cur == "" {
		return false
	}
	if !verLess(cur, prev) {
		return false
	}
	// Refuse dirty prev/: VERSION claims X but binary is Y (classic oscillation fuel).
	if bv := verNorm(parseVerField(binVersion(filepath.Join(prevDir(), "netductor")))); bv != "" && bv != prev {
		fmt.Fprintln(os.Stderr, "stack: refuse promote — prev VERSION", prev, "!= binary", bv)
		return false
	}
	fmt.Fprintln(os.Stderr, "stack: promote last-good", prev, "over running", cur)
	if err := restoreFrom(prevDir()); err != nil {
		fmt.Fprintln(os.Stderr, "promote failed:", err)
		return false
	}
	_ = os.RemoveAll(attemptDir())
	notify.AlertOnce("stack:promote:"+prev, "⬆️ Stack promoted last-good <code>"+prev+"</code> (was <code>"+cur+"</code>)")
	return true
}

func parseVerField(s string) string {
	for _, f := range strings.Fields(s) {
		if len(f) > 0 && f[0] >= '0' && f[0] <= '9' && strings.Contains(f, ".") {
			return f
		}
	}
	return strings.TrimSpace(s)
}

// HealVersion: if prev/ is newer than running binary, restore prev (fixes 121 running / 146 prev).
func HealVersion() error {
	if ok, why := IsPinned(); ok {
		return fmt.Errorf("stack pinned: %s", why)
	}
	run := verNorm(version.Running())
	prev := verNorm(readDirVersion(prevDir()))
	bv := verNorm(parseVerField(binVersion(filepath.Join(prevDir(), "netductor"))))
	if bv != "" {
		prev = bv // binary in prev/ is ground truth
	}
	if prev == "" {
		return fmt.Errorf("no prev snapshot")
	}
	if run != "" && !verLess(run, prev) && run == prev {
		fmt.Fprintln(os.Stderr, "heal: already at", run)
		return nil
	}
	if run != "" && verLess(prev, run) {
		return fmt.Errorf("prev %s is older than running %s — use stack apply instead", prev, run)
	}
	fmt.Fprintln(os.Stderr, "heal: restoring prev", prev, "(running was", run+")")
	if err := restoreFrom(prevDir()); err != nil {
		return err
	}
	notify.AlertOnce("stack:heal:"+prev, "🩹 Stack heal restored <code>"+prev+"</code> from prev/")
	return nil
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
			_, _ = install.WaitForBackupPull(12 * time.Second)
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
	// R15: poll API health instead of fixed 12s sleep
	fmt.Fprintln(os.Stderr, "stack apply: health pending (polling api up to 15s)")
	_ = waitAPIHealthy(15 * time.Second)
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
		fmt.Fprintln(os.Stderr, "stack apply: health pending (retry poll up to 12s)")
		_ = waitAPIHealthy(12 * time.Second)
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
	// Prefer live /healthz over unit state alone (unit can be "active" while API deadlocked).
	if len(failed) == 0 && !apiHTTPHealthy() {
		fmt.Fprintln(os.Stderr, "stack apply: health pending (final poll up to 8s)")
		if !waitAPIHealthy(8 * time.Second) {
			failed = append(failed, "api-http-health")
		}
	}
	if len(failed) > 0 {
		fmt.Fprintln(os.Stderr, "health failed (api), KEEPING new binaries:", failed)
		notify.AlertOnce("stack:health:"+tag, "⚠️ Stack apply <code>"+tag+"</code> health: "+strings.Join(failed, ", ")+" — binaries kept (no auto-downgrade; prev unchanged)")
		_ = os.RemoveAll(attemptDir())
		// Do NOT snapshotPrev on health fail — avoids marking a flaky apply as last-good.
		return fmt.Errorf("apply health fail %v (binaries kept at %s)", failed, tag)
	}
	// Secondary is NOT auto-upgraded — operator must enqueue upgrade explicitly.
	msg := "✅ Stack apply ok <code>" + tag + "</code> (secondary: manual upgrade only)"
	notify.AlertOnce("stack:apply-ok:"+tag, msg)
	fmt.Fprintln(os.Stderr, "stack apply ok", tag)
	// Re-assert host baseline after binary swap (role file + firewall + watchdog).
	if err := install.EnsureHostBaseline("primary"); err != nil {
		fmt.Fprintln(os.Stderr, "warn host baseline:", err)
	}
	// Last-good = newly installed binaries (never keep ancient 0.9.121 as prev after success)
	_ = snapshotPrev()
	_ = os.RemoveAll(attemptDir())
	// brew-style: drop upgrade leftovers (dry nothing left is fine)
	if rep := cleanup.Run(true); len(rep.Items) > 0 {
		fmt.Fprintln(os.Stderr, "stack cleanup:", rep.FreedHuman, len(rep.Items), "item(s)")
	}
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

// Rollback restores last-good (prev/) binaries — manual only.
func Rollback() error {
	if ok, why := IsPinned(); ok {
		return fmt.Errorf("stack pinned: %s", why)
	}
	prev := readDirVersion(prevDir())
	bv := verNorm(parseVerField(binVersion(filepath.Join(prevDir(), "netductor"))))
	if prev != "" && bv != "" && prev != bv {
		return fmt.Errorf("refuse rollback: dirty prev/ VERSION=%s binary=%s — fix or re-snapshot", prev, bv)
	}
	if err := restoreFrom(prevDir()); err != nil {
		return err
	}
	notify.AlertOnce("stack:rollback-manual", "↩️ Stack rollback restored last-good binaries")
	fmt.Fprintln(os.Stderr, "stack rollback done")
	return nil
}

// WatchdogOnce restarts core units that are failed/inactive.
func WatchdogOnce() {
	// Only cleanup + restart failed units. Never swap binaries here (promote was causing silent version flips).
	_ = os.RemoveAll(attemptDir())
	var restarted []string
	for _, u := range managedUnits() {
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

// ScheduleApply runs stack apply via systemd-run so API/bot are not mid-request when binaries swap.
func ScheduleApply(tag string) error {
	if ok, why := IsPinned(); ok {
		return fmt.Errorf("stack pinned: %s", why)
	}
	var err error
	tag, err = ndupdate.ValidReleaseTag(tag)
	if err != nil {
		return err
	}
	if busy, cur, _ := ApplyInProgress(); busy {
		return fmt.Errorf("apply already in progress: %s", cur)
	}
	if applyUnitRunning() {
		return fmt.Errorf("apply unit %s still running", applyUnit)
	}
	// Do not take apply lock here — child `stack apply` owns the lock.
	// Tag is positional $1 so it cannot break out of the shell script.
	cmd := exec.Command("systemd-run", "--unit="+applyUnit, "--collect",
		"/bin/bash", "-c",
		`sleep 2; /usr/local/bin/netductor stack apply "$1" >>/var/log/netductor-stack-apply.log 2>&1`,
		"netductor-stack-apply", tag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemd-run: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
