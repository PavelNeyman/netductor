// Package stack is a thin orchestrator.
// Policy: no spontaneous binary upgrade/rollback. Apply/Rollback/Heal/Promote only on explicit CLI/API/TG action.
// Package stack is a thin orchestrator for netductor-owned systemd units and binaries.
//
// Simplified model (after 0.9.137+):
//
//	Single writer for primary node+tg binaries: Apply / ApplyOpts only.
//	ScheduleApply → systemd-run → Apply (API/TG must not apply in-process).
//	prev/     = last successful apply only (manual Rollback target).
//	attempt/  = durable transaction journal + pair pre-images until commit/rollback.
//	Watchdog  = restart failed units only (never swaps binaries).
//	Promote   = explicit CLI only; verifies prev binary matches VERSION.
//	Pin       = blocks Apply and Promote while stabilizing.
package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/cleanup"
	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/integrity"
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

// preparePrevSnapshot stages a complete last-good snapshot without changing prev/.
func preparePrevSnapshot(release string) error {
	root := filepath.Join(paths.StateDir(), "stack")
	next := filepath.Join(root, "prev.next")
	if err := os.RemoveAll(next); err != nil { return err }
	if err := os.MkdirAll(next, 0o755); err != nil { return err }
	if err := syncDir(root); err != nil { return err }
	for _, src := range []string{"/usr/local/bin/netductor", "/usr/local/bin/netductor-tg"} {
		b, err := os.ReadFile(src)
		if err != nil { return fmt.Errorf("snapshot %s: %w", src, err) }
		if err := durableWrite(filepath.Join(next, filepath.Base(src)), b, 0o755); err != nil { return err }
	}
	if err := durableWrite(filepath.Join(next, "VERSION"), []byte(strings.TrimSpace(release)+"\n"), 0o644); err != nil { return err }
	return syncDir(next)
}

func verifySnapshotPair(dir string, tx *applyTx) error {
	for _, item := range []struct{name, path, want string}{
		{"node", filepath.Join(dir, "netductor"), tx.NodeNewSHA},
		{"tg", filepath.Join(dir, "netductor-tg"), tx.TGNewSHA},
	} {
		got, err := fileSHA(item.path)
		if err != nil { return fmt.Errorf("verify %s last-good snapshot: %w", item.name, err) }
		if got != item.want { return fmt.Errorf("verify %s last-good snapshot: sha256 mismatch", item.name) }
	}
	return nil
}

func promotePrevSnapshot() error {
	root := filepath.Join(paths.StateDir(), "stack")
	prev, next := prevDir(), filepath.Join(root, "prev.next")
	old := filepath.Join(root, "prev.old")
	tx, err := loadTx(attemptDir())
	if err != nil { return fmt.Errorf("load committed transaction for snapshot promotion: %w", err) }
	if tx.Phase != txCommitted { return fmt.Errorf("refuse snapshot promotion for transaction phase %q", tx.Phase) }
	if err := verifySnapshotPair(next, tx); err != nil { return err }
	if err := os.RemoveAll(old); err != nil { return err }
	if _, err := os.Stat(prev); err == nil {
		if err := os.Rename(prev, old); err != nil { return err }
		if err := syncDir(root); err != nil { return err }
	} else if !os.IsNotExist(err) { return err }
	if err := os.Rename(next, prev); err != nil {
		if _, e := os.Stat(old); e == nil { _ = os.Rename(old, prev); _ = syncDir(root) }
		return err
	}
	if err := syncDir(root); err != nil { return err }
	if err := os.RemoveAll(old); err != nil { return err }
	return syncDir(root)
}

func recoverPrevSnapshotSwap() error {
	return recoverPrevSnapshotSwapForOwner(applyOwnerActive())
}

func recoverPrevSnapshotSwapForOwner(ownerActive bool) error {
	// An active Apply owns prev.next/prev.old. Startup during its API restart
	// must not clean up the in-flight directory promotion.
	if ownerActive { return nil }
	root := filepath.Join(paths.StateDir(), "stack")
	if _, err := os.Stat(root); os.IsNotExist(err) { return nil } else if err != nil { return err }
	prev := prevDir()
	next := filepath.Join(root, "prev.next")
	old := filepath.Join(root, "prev.old")
	tx, txErr := loadTx(attemptDir())
	if txErr != nil && !os.IsNotExist(txErr) {
		return fmt.Errorf("cannot recover last-good promotion with invalid transaction journal: %w", txErr)
	}
	committed := txErr == nil && tx.Phase == txCommitted

	_, prevErr := os.Stat(prev)
	_, oldErr := os.Stat(old)
	_, nextErr := os.Stat(next)
	if oldErr == nil && !committed {
		return fmt.Errorf("ambiguous last-good promotion: prev.old exists without a committed transaction")
	}
	if os.IsNotExist(prevErr) {
		if committed && nextErr == nil {
			if err := verifySnapshotPair(next, tx); err != nil { return fmt.Errorf("committed staged last-good snapshot is invalid: %w", err) }
			// Finish the first promotion after a crash between commit and rename.
			if err := os.Rename(next, prev); err != nil { return err }
			if err := syncDir(root); err != nil { return err }
		} else if committed && nextErr != nil && !os.IsNotExist(nextErr) {
			return fmt.Errorf("inspect staged committed last-good snapshot: %w", nextErr)
		} else if oldErr == nil {
			// An uncommitted rename must restore the previous last-good directory.
			if err := os.Rename(old, prev); err != nil { return err }
			if err := syncDir(root); err != nil { return err }
		}
	} else if prevErr != nil { return prevErr
	} else if committed {
		// If prev already matches the committed pair, only cleanup remains.
		// Otherwise finish the promotion from the verified staged directory.
		prevPairErr := verifySnapshotPair(prev, tx)
		if prevPairErr != nil {
			if nextErr != nil { return fmt.Errorf("committed last-good snapshot is invalid and staged replacement is unavailable: %w", prevPairErr) }
			if err := verifySnapshotPair(next, tx); err != nil { return fmt.Errorf("committed staged last-good snapshot is invalid: %w", err) }
			if oldErr == nil { return fmt.Errorf("ambiguous last-good promotion: prev differs from committed pair while prev.old exists") }
			if err := os.Rename(prev, old); err != nil { return err }
			if err := syncDir(root); err != nil { return err }
			if err := os.Rename(next, prev); err != nil {
				_ = os.Rename(old, prev)
				_ = syncDir(root)
				return err
			}
			if err := syncDir(root); err != nil { return err }
		}
	}

	// A committed first install must have either a promoted prev or a recoverable stage.
	if committed {
		if _, err := os.Stat(prev); os.IsNotExist(err) {
			if _, nextErr := os.Stat(next); os.IsNotExist(nextErr) {
				if _, oldErr := os.Stat(old); os.IsNotExist(oldErr) {
					return fmt.Errorf("committed apply has no last-good snapshot or recoverable promotion")
				}
			}
		}
	}
	// Once prev exists, old is obsolete. An uncommitted tx never promotes next.
	if err := os.RemoveAll(old); err != nil { return err }
	if err := os.RemoveAll(next); err != nil { return err }
	return syncDir(root)
}

// replaceExecutable performs a durable same-filesystem replacement and verifies bytes.
func replaceExecutable(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil { return err }
	tmp := dest + ".new"
	if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) { return err }
	in, err := os.Open(src)
	if err != nil { return err }
	defer in.Close()
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil { return err }
	_, copyErr := io.Copy(out, in)
	chmodErr := out.Chmod(0o755)
	syncErr := out.Sync()
	closeErr := out.Close()
	for _, e := range []error{copyErr, chmodErr, syncErr, closeErr} {
		if e != nil { _ = os.Remove(tmp); return e }
	}
	if err := os.Rename(tmp, dest); err != nil { _ = os.Remove(tmp); return err }
	if err := syncDir(filepath.Dir(dest)); err != nil { return err }
	a, err := fileSHA(src); if err != nil { return err }
	b, err := fileSHA(dest); if err != nil { return err }
	if a != b { return fmt.Errorf("replacement verification failed for %s", dest) }
	return nil
}

// installStagedPair installs node+tg from staged files. If tg fails after node, restores node from attempt/.
func installStagedPair(nodeSrc, tgSrc string) error {
	return installStagedPairTo(nodeSrc, tgSrc, "/usr/local/bin/netductor", "/usr/local/bin/netductor-tg", attemptDir())
}

// installStagedPairTo is the testable core for pair install with restore on partial failure.
func installStagedPairTo(nodeSrc, tgSrc, nodeDest, tgDest, att string) error {
	if err := os.MkdirAll(att, 0o755); err != nil {
		return fmt.Errorf("attempt dir: %w", err)
	}
	if err := syncDir(filepath.Dir(att)); err != nil {
		return fmt.Errorf("attempt dir sync: %w", err)
	}
	tx := applyTx{ID: fmt.Sprintf("tx-%d", time.Now().UnixNano()), NodeDest: nodeDest, TGDest: tgDest, StartedAt: time.Now().UTC()}
	for _, pair := range []struct{ path, name, field string }{
		{nodeDest, "netductor", "node"},
		{tgDest, "netductor-tg", "tg"},
	} {
		st, err := os.Stat(pair.path)
		if err != nil { return fmt.Errorf("stat required pre-image %s: %w", pair.name, err) }
		if !st.Mode().IsRegular() { return fmt.Errorf("%s is not a regular file", pair.path) }
		sha, err := durableBackup(pair.path, filepath.Join(att, pair.name))
		if err != nil {
			return fmt.Errorf("backup %s: %w", pair.name, err)
		}
		if pair.field == "node" {
			tx.NodePreSHA = sha
		} else {
			tx.TGPreSHA = sha
		}
	}
	nsha, err := fileSHA(nodeSrc)
	if err != nil {
		return fmt.Errorf("node source: %w", err)
	}
	tsha, err := fileSHA(tgSrc)
	if err != nil {
		return fmt.Errorf("tg source: %w", err)
	}
	tx.NodeNewSHA, tx.TGNewSHA = nsha, tsha
	tx.Phase = txPrepared
	if err := writeTx(att, tx); err != nil {
		return fmt.Errorf("tx prepared: %w", err)
	}
	if err := replaceExecutable(nodeSrc, nodeDest); err != nil {
		return fmt.Errorf("install node: %w", err)
	}
	tx.Phase = txNodeInstalled
	if err := writeTx(att, tx); err != nil {
		return fmt.Errorf("tx node_installed: %w", err)
	}
	if err := replaceExecutable(tgSrc, tgDest); err != nil {
		if rerr := restoreChecked(filepath.Join(att, "netductor"), nodeDest, tx.NodePreSHA); rerr != nil {
			return fmt.Errorf("install tg: %w (node restore failed: %v)", err, rerr)
		}
		tx.Phase = txPrepared
		_ = writeTx(att, tx)
		return fmt.Errorf("install tg: %w (node restored from attempt snapshot)", err)
	}
	tx.Phase = txTGInstalled
	if err := writeTx(att, tx); err != nil {
		return fmt.Errorf("tx tg_installed: %w", err)
	}
	return nil
}

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
	if err := cleanupTerminalAttempt(attemptDir()); err != nil { fmt.Fprintln(os.Stderr, "promote: retain apply journal:", err) }
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
	if t2, err := ndupdate.ValidReleaseTag(tag); err != nil {
		return err
	} else {
		tag = t2
	}
	// Recover stale journal before acquiring this run's owner lock; otherwise
	// the new lock could make an old incomplete transaction look active.
	if err := RecoverIncompleteApply(); err != nil { return err }
	if !TryAcquireApplyLock(tag) {
		busy, cur, _ := ApplyInProgress()
		_ = busy
		return fmt.Errorf("apply already in progress: %s", cur)
	}
	defer ClearApplyLock()
	if _, err := ndupdate.BeginNodeApply(tag, ""); err != nil {
		// Registry conflict is rare next to stack lock; surface it.
		return fmt.Errorf("update job registry: %w", err)
	}
	applyOK := false
	var applyFail error
	defer func() {
		if applyOK {
			_, _ = ndupdate.FinishNodeApply(true, "")
		} else {
			msg := ""
			if applyFail != nil {
				msg = applyFail.Error()
			}
			_, _ = ndupdate.FinishNodeApply(false, msg)
		}
	}()
	defer func() { _ = notify.FlushAlerts(true) }()
	cur := verNorm(version.Running())
	want := verNorm(tag)
	if cur != "" && want != "" && verLess(want, cur) {
		applyFail = fmt.Errorf("refuse downgrade: running %s > target %s (use manual binary replace if intentional)", cur, want)
		return applyFail
	}
	if os.Getenv("NETDUCTOR_STACK_SKIP_PREFLIGHT") != "1" {
		pf := Preflight(tag)
		fmt.Fprint(os.Stderr, FormatPreflight(pf))
		if !pf.OK {
			applyFail = fmt.Errorf("preflight failed for %s (set NETDUCTOR_STACK_SKIP_PREFLIGHT=1 to override)", tag)
			return applyFail
		}
	}
	// Only terminal evidence can be cleaned before a new attempt.
	if err := cleanupTerminalAttempt(attemptDir()); err != nil { applyFail = fmt.Errorf("clean previous apply evidence: %w", err); return applyFail }
	if !noBackup {
		fmt.Fprintln(os.Stderr, "stack apply: pre-backup")
		path, err := install.Backup()
		if err != nil {
			applyFail = fmt.Errorf("pre-backup required: %w (use stack apply --no-backup only if intentional)", err)
			notify.AlertOnce("stack:apply-fail:"+tag, "🔴 Stack apply failed (backup): "+err.Error())
			return applyFail
		}
		fmt.Fprintln(os.Stderr, "backup:", path)
		_, _ = install.WaitForBackupPull(12*time.Second, time.Now().UTC().Add(-2*time.Second), "", nil)
	}
	fmt.Fprintln(os.Stderr, "stack apply:", tag)
	notify.AlertOnce("stack:apply:"+tag, "⬆️ Stack apply <code>"+tag+"</code> started")
	// Stage both binaries before touching live paths — avoid node-only partial apply.
	stageDir := filepath.Join(paths.StateDir(), "stack", "stage")
	if err := os.RemoveAll(stageDir); err != nil {
		applyFail = fmt.Errorf("remove stale stage dir: %w", err)
		return applyFail
	}
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		applyFail = fmt.Errorf("stage dir: %w", err)
		return applyFail
	}
	nodeStage := filepath.Join(stageDir, "netductor")
	tgStage := filepath.Join(stageDir, "netductor-tg")
	if err := ndupdate.DownloadReleaseAsset(tag, "node", nodeStage); err != nil {
		notify.AlertOnce("stack:apply-fail:"+tag, "🔴 Stack apply failed (node download): "+err.Error())
		applyFail = fmt.Errorf("node download: %w", err)
		return applyFail
	}
	if err := ndupdate.DownloadReleaseAsset(tag, "tg", tgStage); err != nil {
		notify.AlertOnce("stack:apply-fail:"+tag, "🔴 Stack apply failed (tg download): "+err.Error())
		applyFail = fmt.Errorf("tg download: %w", err)
		return applyFail
	}
	if err := installStagedPair(nodeStage, tgStage); err != nil {
		applyFail = rollbackApplyFailure(attemptDir(), fmt.Errorf("install staged pair: %w", err))
		notify.AlertOnce("stack:apply-fail:"+tag, "🔴 Stack apply failed (install): "+applyFail.Error())
		return applyFail
	}
	_ = os.RemoveAll(stageDir)
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		applyFail = rollbackApplyFailure(attemptDir(), fmt.Errorf("systemctl daemon-reload: %w", err))
		return applyFail
	}
	if err := exec.Command("systemctl", "reset-failed", "netductor-api", "netductor-telegram-bot").Run(); err != nil {
		applyFail = rollbackApplyFailure(attemptDir(), fmt.Errorf("systemctl reset-failed: %w", err))
		return applyFail
	}
	if err := exec.Command("systemctl", "restart", "netductor-api").Run(); err != nil {
		applyFail = rollbackApplyFailure(attemptDir(), fmt.Errorf("systemctl restart netductor-api: %w", err))
		notify.AlertOnce("stack:apply-fail:"+tag, "🔴 Stack apply failed (api restart): "+applyFail.Error())
		return applyFail
	}
	if err := exec.Command("systemctl", "restart", "netductor-telegram-bot").Run(); err != nil {
		applyFail = rollbackApplyFailure(attemptDir(), fmt.Errorf("systemctl restart netductor-telegram-bot: %w", err))
		notify.AlertOnce("stack:apply-fail:"+tag, "🔴 Stack apply failed (tg restart): "+applyFail.Error())
		return applyFail
	}
	// redirect is best-effort (may be absent on some roles)
	if err := exec.Command("systemctl", "try-restart", "netductor-redirect").Run(); err != nil {
		fmt.Fprintln(os.Stderr, "systemctl try-restart netductor-redirect:", err)
	}
	// Bot long-poll / restart is flaky for 10–20s — do NOT hard-fail on telegram-bot alone
	// (that caused auto-rollback to ancient prev like 0.9.121 after a successful binary replace).
	// R15: poll API health instead of fixed 12s sleep
	fmt.Fprintln(os.Stderr, "stack apply: health pending (polling api up to 15s)")
	_ = waitAPIHealthy(15 * time.Second)
	st := Collect()
	// Apply requires completed systemd startup, not merely "activating".
	// Collect intentionally treats activating as OK for status displays.
	var failed []string
	for _, u := range st.Units {
		if (u.Unit == "netductor-api" || u.Unit == "netductor-telegram-bot") && u.Active != "active" {
			failed = append(failed, u.Unit+":"+u.Active)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintln(os.Stderr, "health soft-fail (api), retry restart:", failed)
		var retryErrors []string
		if err := exec.Command("systemctl", "reset-failed", "netductor-api", "netductor-telegram-bot").Run(); err != nil {
			retryErrors = append(retryErrors, "systemctl-reset-failed:"+err.Error())
		}
		if err := exec.Command("systemctl", "restart", "netductor-api", "netductor-telegram-bot").Run(); err != nil {
			retryErrors = append(retryErrors, "systemctl-restart-retry:"+err.Error())
		}
		fmt.Fprintln(os.Stderr, "stack apply: health pending (retry poll up to 12s)")
		_ = waitAPIHealthy(12 * time.Second)
		st = Collect()
		failed = nil
		for _, u := range st.Units {
			if (u.Unit == "netductor-api" || u.Unit == "netductor-telegram-bot") && u.Active != "active" {
				failed = append(failed, u.Unit+":"+u.Active)
			}
		}
		failed = append(failed, retryErrors...)
	}
	// Prefer live /healthz over unit state alone (unit can be "active" while API deadlocked).
	if len(failed) == 0 && !apiHTTPHealthy() {
		fmt.Fprintln(os.Stderr, "stack apply: health pending (final poll up to 8s)")
		if !waitAPIHealthy(8 * time.Second) {
			failed = append(failed, "api-http-health")
		}
	}
	if len(failed) > 0 {
		applyFail = rollbackApplyFailure(attemptDir(), fmt.Errorf("apply health failed: %v", failed))
		notify.AlertOnce("stack:health:"+tag, "🔴 Stack apply <code>"+tag+"</code> failed health checks: "+strings.Join(failed, ", "))
		return applyFail
	}
	// Secondary is NOT auto-upgraded — operator must enqueue upgrade explicitly.
	notify.AlertOnce("stack:apply-checks:"+tag, "Stack apply health checks passed for <code>"+tag+"</code>; finalizing transaction")
	fmt.Fprintln(os.Stderr, "stack apply checks passed", tag)
	// Baseline and last-good snapshot preparation are part of the transaction.
	if err := install.EnsureHostBaseline(firewall.DetectRole()); err != nil {
		applyFail = rollbackApplyFailure(attemptDir(), fmt.Errorf("host baseline after apply: %w", err))
		return applyFail
	}
	if err := preparePrevSnapshot(tag); err != nil {
		applyFail = rollbackApplyFailure(attemptDir(), fmt.Errorf("prepare last-good snapshot: %w", err))
		return applyFail
	}
	if err := commitInstalledPair(attemptDir()); err != nil {
		applyFail = rollbackApplyFailure(attemptDir(), fmt.Errorf("commit installed pair: %w", err))
		return applyFail
	}
	// The binary pair is now durably committed. Later snapshot-directory
	// promotion is recoverable bookkeeping and must not mark the release failed.
	applyOK = true
	ndupdate.WriteVERSION(tag)
	saveCurrent(tag)
	if err := promotePrevSnapshot(); err != nil {
		applyFail = fmt.Errorf("promote last-good snapshot after durable commit: %w", err)
		return applyFail
	}
	if err := integrity.WriteManifest(); err != nil {
		fmt.Fprintln(os.Stderr, "warn: write integrity manifest after commit:", err)
	}
	notify.AlertOnce("stack:apply-ok:"+tag, "✅ Stack apply committed <code>"+tag+"</code> (secondary: manual upgrade only)")
	_ = notify.FlushAlerts(true)
	if err := cleanupTerminalAttempt(attemptDir()); err != nil {
		fmt.Fprintln(os.Stderr, "warn: retain committed apply evidence:", err)
	}
	// brew-style: drop upgrade leftovers (dry nothing left is fine)
	if rep := cleanup.Run(true); len(rep.Items) > 0 {
		fmt.Fprintln(os.Stderr, "stack cleanup:", rep.FreedHuman, len(rep.Items), "item(s)")
	}
	applyOK = true
	return nil
}

func rollbackApplyFailure(att string, cause error) error {
	tx, err := loadTx(att)
	if err != nil {
		if os.IsNotExist(err) { return cause }
		return fmt.Errorf("%v; load rollback journal: %w", cause, err)
	}
	if err := rollbackPair(att, tx); err != nil { return fmt.Errorf("%v; rollback blocked: %w", cause, err) }
	var rollbackPostcheckErrs []string
	if err := install.EnsureHostBaseline(firewall.DetectRole()); err != nil {
		rollbackPostcheckErrs = append(rollbackPostcheckErrs, "host baseline: "+err.Error())
	}
	var restartErrs []string
	for _, unit := range []string{"netductor-api", "netductor-telegram-bot"} {
		if err := exec.Command("systemctl", "restart", unit).Run(); err != nil { restartErrs = append(restartErrs, unit+": "+err.Error()) }
	}
	if len(restartErrs) > 0 { rollbackPostcheckErrs = append(rollbackPostcheckErrs, "service restart: "+strings.Join(restartErrs, "; ")) }
	if len(restartErrs) == 0 {
		if !waitAPIHealthy(20*time.Second) { rollbackPostcheckErrs = append(rollbackPostcheckErrs, "API health check failed") }
		for _, unit := range []string{"netductor-api", "netductor-telegram-bot"} {
			active, _ := unitState(unit)
			if active != "active" { rollbackPostcheckErrs = append(rollbackPostcheckErrs, unit+" is "+active) }
		}
	}
	if len(rollbackPostcheckErrs) > 0 {
		return fmt.Errorf("%v; restored binary pair but rollback post-check failed: %s", cause, strings.Join(rollbackPostcheckErrs, "; "))
	}
	return cause
}

func restoreFrom(dir string) error {
	// Recover before taking our own lock so an old journal is not mistaken for
	// this restore's active transaction.
	if err := RecoverIncompleteApply(); err != nil { return err }
	if !TryAcquireApplyLock("manual-restore") { return fmt.Errorf("another stack apply/restore is active") }
	defer ClearApplyLock()
	if err := cleanupTerminalAttempt(attemptDir()); err != nil { return err }

	root := filepath.Join(paths.StateDir(), "stack")
	stage := filepath.Join(root, "restore-stage")
	if err := os.RemoveAll(stage); err != nil { return err }
	if err := os.MkdirAll(stage, 0o700); err != nil { return err }
	defer os.RemoveAll(stage)
	for _, name := range []string{"netductor", "netductor-tg"} {
		src := filepath.Join(dir, name)
		st, err := os.Stat(src)
		if err != nil { return fmt.Errorf("rollback source %s: %w", name, err) }
		if !st.Mode().IsRegular() { return fmt.Errorf("rollback source %s is not a regular file", name) }
		data, err := os.ReadFile(src)
		if err != nil { return err }
		if err := durableWrite(filepath.Join(stage, name), data, 0o755); err != nil { return err }
	}
	release := ""
	if b, err := os.ReadFile(filepath.Join(dir, "VERSION")); err == nil {
		release = strings.TrimSpace(string(b))
	} else if !os.IsNotExist(err) { return err }
	if release == "" { release = parseVerField(binVersion(filepath.Join(stage, "netductor"))) }
	if release == "" { return fmt.Errorf("rollback source has no verifiable release marker") }

	if err := installStagedPair(filepath.Join(stage, "netductor"), filepath.Join(stage, "netductor-tg")); err != nil {
		return rollbackApplyFailure(attemptDir(), fmt.Errorf("install rollback pair: %w", err))
	}
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return rollbackApplyFailure(attemptDir(), fmt.Errorf("rollback daemon-reload: %w", err))
	}
	if err := exec.Command("systemctl", "reset-failed", "netductor-api", "netductor-telegram-bot").Run(); err != nil {
		return rollbackApplyFailure(attemptDir(), fmt.Errorf("rollback reset-failed: %w", err))
	}
	for _, unit := range []string{"netductor-api", "netductor-telegram-bot"} {
		if err := exec.Command("systemctl", "restart", unit).Run(); err != nil {
			return rollbackApplyFailure(attemptDir(), fmt.Errorf("rollback restart %s: %w", unit, err))
		}
	}
	if !waitAPIHealthy(20*time.Second) {
		return rollbackApplyFailure(attemptDir(), fmt.Errorf("rollback API health check failed"))
	}
	for _, unit := range []string{"netductor-api", "netductor-telegram-bot"} {
		active, _ := unitState(unit)
		if active != "active" {
			return rollbackApplyFailure(attemptDir(), fmt.Errorf("rollback service %s is %s", unit, active))
		}
	}
	if err := install.EnsureHostBaseline(firewall.DetectRole()); err != nil {
		return rollbackApplyFailure(attemptDir(), fmt.Errorf("rollback host baseline: %w", err))
	}
	if err := commitInstalledPair(attemptDir()); err != nil {
		return rollbackApplyFailure(attemptDir(), fmt.Errorf("commit restored pair: %w", err))
	}
	ndupdate.WriteVERSION(release)
	saveCurrent(release)
	if err := cleanupTerminalAttempt(attemptDir()); err != nil {
		return fmt.Errorf("restored pair committed; retain journal: %w", err)
	}
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
	// Watchdog must never delete a transaction journal or its pre-images.
	if err := RecoverIncompleteApply(); err != nil {
		fmt.Fprintln(os.Stderr, "watchdog: apply recovery blocked:", err)
		return
	}
	// A persisted rolled_back journal is now fully post-checked by
	// RecoverIncompleteApply; remove it only after those checks succeed.
	if tx, err := loadTx(attemptDir()); err == nil && tx.Phase == txRolledBack {
		if err := cleanupTerminalAttempt(attemptDir()); err != nil {
			fmt.Fprintln(os.Stderr, "watchdog: retain rolled-back journal:", err)
			return
		}
	}
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
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("watchdog daemon-reload: %w", err)
	}
	if err := exec.Command("systemctl", "enable", "--now", "netductor-stack-watchdog.timer").Run(); err != nil {
		return fmt.Errorf("enable watchdog timer: %w", err)
	}
	active, sub := unitState("netductor-stack-watchdog.timer")
	if active != "active" && sub != "waiting" {
		return fmt.Errorf("watchdog timer is %s/%s after enable", active, sub)
	}
	return nil
}

// ScheduleApply runs stack apply via systemd-run so API/bot are not mid-request when binaries swap.
func ScheduleApply(tag string) error {
	if ok, why := IsPinned(); ok {
		return fmt.Errorf("stack pinned: %s", why)
	}
	tag = strings.TrimSpace(tag)
	var err error
	if tag == "" {
		tag, err = ndupdate.LatestReleaseTag()
		if err != nil {
			return err
		}
	}
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
