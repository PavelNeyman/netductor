package install

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

func InstallBackup() error {
	bin := "/usr/local/bin/netductor"
	if readSecret("backup_key") == "" {
		_ = writeSecret("backup_key", randomHex(32))
	}
	unit := fmt.Sprintf(`[Unit]
Description=Netductor backup once
After=network-online.target

[Service]
Type=oneshot
ExecStart=%s backup
`, bin)
	timer := `[Unit]
Description=Netductor daily backup

[Timer]
OnCalendar=*-*-* 01:00:00 UTC
# 04:00 Europe/Moscow (MSK=UTC+3)
Persistent=true
RandomizedDelaySec=30m

[Install]
WantedBy=timers.target
`
	if err := writeUnit("netductor-backup.service", unit); err != nil {
		return err
	}
	if err := writeUnit("netductor-backup.timer", timer); err != nil {
		return err
	}
	_ = run("systemctl", "enable", "netductor-backup.timer")
	_ = run("systemctl", "start", "netductor-backup.timer")
	fmt.Fprintln(os.Stderr, "backup timer enabled")
	return nil
}

func writeBackupKeyRecovery() error {
	key := readSecret("backup_key")
	if key == "" {
		return nil
	}
	dir := filepath.Join(paths.StateDir(), "backups")
	_ = os.MkdirAll(dir, 0o700)
	path := filepath.Join(dir, "BACKUP_KEY.txt")
	return os.WriteFile(path, []byte(key+"\n"), 0o600)
}

func keyBytes(pass string) []byte {
	h := sha256.Sum256([]byte(pass))
	return h[:]
}

func encryptFile(inPath, outPath, pass string) error {
	st, err := os.Stat(inPath)
	if err != nil {
		return err
	}
	// In-process AES-GCM loads the whole file twice — cap to protect bot/api RAM.
	const maxInMem = 64 << 20 // 64 MiB
	if st.Size() > maxInMem {
		return encryptFileOpenSSL(inPath, outPath, pass)
	}
	in, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(keyBytes(pass))
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	out := gcm.Seal(nonce, nonce, in, nil)
	return os.WriteFile(outPath, out, 0o600)
}

// encryptFileOpenSSL streams large archives (openssl enc AES-256-CBC + salt).
// Format differs from small GCM blobs — decryptFile tries both.
func encryptFileOpenSSL(inPath, outPath, pass string) error {
	cmd := exec.Command("openssl", "enc", "-aes-256-cbc", "-salt", "-pbkdf2",
		"-in", inPath, "-out", outPath, "-pass", "pass:"+pass)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("openssl enc: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return os.Chmod(outPath, 0o600)
}

func decryptFile(inPath, outPath, pass string) error {
	// Try legacy in-memory GCM first (small archives).
	in, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	if len(in) < 64<<20 {
		block, err := aes.NewCipher(keyBytes(pass))
		if err == nil {
			if gcm, err2 := cipher.NewGCM(block); err2 == nil && len(in) >= gcm.NonceSize() {
				nonce, ct := in[:gcm.NonceSize()], in[gcm.NonceSize():]
				if plain, err3 := gcm.Open(nil, nonce, ct, nil); err3 == nil {
					return os.WriteFile(outPath, plain, 0o600)
				}
			}
		}
	}
	// openssl / large
	cmd := exec.Command("openssl", "enc", "-d", "-aes-256-cbc", "-pbkdf2",
		"-in", inPath, "-out", outPath, "-pass", "pass:"+pass)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("decrypt: gcm and openssl failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return os.Chmod(outPath, 0o600)
}


// WaitForBackupPull waits up to timeout for online secondaries to report last_cmd backup_pull ok.
// Returns (acked, pending). Soft signal for pre-upgrade; does not fail hard if timeout.
func WaitForBackupPull(timeout time.Duration) (acked, pending int) {
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	deadline := time.Now().Add(timeout)
	online := 0
	for _, d := range secondary.List() {
		if secondary.Online(d, 2*time.Minute) {
			online++
		}
	}
	if online == 0 {
		return 0, 0
	}
	for time.Now().Before(deadline) {
		acked, pending = 0, 0
		for _, d := range secondary.List() {
			if !secondary.Online(d, 2*time.Minute) {
				continue
			}
			if d.LastCmd == "backup_pull" && d.LastCmdOK {
				// recent enough (10 min)
				if time.Since(d.LastCmdAt) < 10*time.Minute {
					acked++
					continue
				}
			}
			pending++
		}
		if pending == 0 && acked > 0 {
			return acked, 0
		}
		time.Sleep(3 * time.Second)
	}
	return acked, pending
}

func Backup() (string, error) {
	dir := filepath.Join(paths.StateDir(), "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	plain := filepath.Join(dir, "netductor-"+stamp+".tar.gz")
	SyncComponentsFromDisk()
	snapshotHostnameForBackup()
	snapshotOperatorKeysForBackup()
	// Ensure manifest exists before packing
	if len(ReadComponentsManifest()) == 0 {
		_ = WriteComponentsManifest(DefaultComponents())
	}
	// Never pack previous backups (exponential growth) or bulky local media/registry blobs.
	args := []string{
		"-czf", plain,
		"--exclude=var/lib/netductor/backups",
		"--exclude=var/lib/netductor/registry",
		"--exclude=var/lib/netductor/nvr/segments",
		"--exclude=var/lib/netductor/git",
		"-C", "/", "etc/netductor",
	}
	// Config + data for managed services (binaries/images reinstalled from COMPONENTS).
	for _, p := range []string{
		"etc/blocky",
		"etc/sing-box",
		"var/lib/netductor", // state, sites, nodes, quotas, guests, components.json
		"opt/netductor/lampac",
		"opt/netductor/profiles",
	} {
		if _, err := os.Stat("/" + p); err == nil {
			args = append(args, p)
		}
	}
	_ = run("tar", args...)
	if st, err := os.Stat(plain); err != nil || st.Size() == 0 {
		etc := paths.EtcDir()
		_ = run("tar", "-czf", plain, "-C", filepath.Dir(etc), filepath.Base(etc))
	}
	if st, err := os.Stat(plain); err != nil || st.Size() == 0 {
		return "", fmt.Errorf("tar failed")
	}
	key := readSecret("backup_key")
	out := plain
	if key != "" {
		enc := plain + ".ndenc"
		if err := encryptFile(plain, enc, key); err != nil {
			return "", err
		}
		_ = os.Remove(plain)
		out = enc
	}
	_ = os.Chmod(out, 0o600)
	// Recovery key + components list alongside archive (plain — needed before decrypt on bare metal).
	_ = writeBackupKeyRecovery()
	_ = writeComponentsSidecar(dir)
	failMark := filepath.Join(paths.StateDir(), "backup_offsite_fail")
	_ = os.Remove(failMark) // cleared unless agent pull fails below
	// Agent pull to RU secondary (HTTPS mTLS) — no primary→secondary SSH.
	n, offline := 0, 0
	for _, d := range secondary.List() {
		if !secondary.Online(d, 3*time.Minute) {
			offline++
			continue
		}
		if err := secondary.EnqueueCmd(d.ID, "backup_pull"); err == nil {
			n++
		}
	}
	if n > 0 {
		fmt.Fprintf(os.Stderr, "backup: queued backup_pull on %d secondary agent(s)\n", n)
		notify.ClearAlert("backup-offsite")
	} else if len(secondary.List()) == 0 {
		fmt.Fprintln(os.Stderr, "backup: no secondary devices registered — skip offsite pull")
	} else {
		msg := fmt.Sprintf("🔴 Backup offsite: no online secondary for backup_pull (offline=%d total=%d)", offline, len(secondary.List()))
		fmt.Fprintln(os.Stderr, msg)
		notify.AlertOnce("backup-offsite", msg)
		_ = os.WriteFile(failMark, []byte(msg), 0o600)
	}
	pruneBackups(dir, BackupKeepCount())
	return out, nil
}

// Restore unpacks backup into paths.EtcDir parent (expects tar of etc dir name).
// For .ndenc decryption key order: explicit keyArg → NETDUCTOR_BACKUP_KEY → secrets/backup_key.
func Restore(archive string, keyArg string) error {
	if archive == "" {
		return fmt.Errorf("usage: netductor restore [--key KEY] <file.tar.gz|.ndenc>")
	}
	src := archive
	tmp := ""
	if strings.HasSuffix(archive, ".ndenc") {
		key := strings.TrimSpace(keyArg)
		if key == "" {
			key = strings.TrimSpace(os.Getenv("NETDUCTOR_BACKUP_KEY"))
		}
		if key == "" {
			key = readSecret("backup_key")
		}
		if key == "" {
			return fmt.Errorf("backup_key missing — pass --key, or NETDUCTOR_BACKUP_KEY, or restore secrets first")
		}
		tmp = filepath.Join(os.TempDir(), "netductor-restore.tar.gz")
		if err := decryptFile(archive, tmp, key); err != nil {
			return err
		}
		src = tmp
		defer os.Remove(tmp)
	}
	// Prefer root extract (new backups: etc/netductor + var/lib/...). Fallback: old etc-only archives.
	if err := run("tar", "-xzf", src, "-C", "/"); err != nil {
		parent := filepath.Dir(paths.EtcDir())
		if err2 := run("tar", "-xzf", src, "-C", parent); err2 != nil {
			return err
		}
	}
	// Ensure key is persisted for future backups after bare-metal restore.
	if key := strings.TrimSpace(keyArg); key != "" {
		_ = writeSecret("backup_key", key)
	} else if k := strings.TrimSpace(os.Getenv("NETDUCTOR_BACKUP_KEY")); k != "" {
		_ = writeSecret("backup_key", k)
	}
	return nil
}

func writeComponentsSidecar(dir string) error {
	SyncComponentsFromDisk()
	comps := MergeComponents(DefaultComponents(), ReadComponentsManifest())
	body := strings.Join(comps, "\n") + "\n"
	return os.WriteFile(filepath.Join(dir, "COMPONENTS.txt"), []byte(body), 0o644)
}

// Recover is bare-metal recovery: install components from manifest, then restore data.
// Order: decrypt → read components from archive → install packages/services → extract data → apply.
func Recover(archive, keyArg string) error {
	if archive == "" {
		return fmt.Errorf("usage: netductor recover [--key KEY] <archive.ndenc>")
	}
	key := strings.TrimSpace(keyArg)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("NETDUCTOR_BACKUP_KEY"))
	}
	src := archive
	tmp := ""
	if strings.HasSuffix(archive, ".ndenc") {
		if key == "" {
			return fmt.Errorf("backup_key required for recover")
		}
		tmp = filepath.Join(os.TempDir(), "netductor-recover.tar.gz")
		if err := decryptFile(archive, tmp, key); err != nil {
			return err
		}
		src = tmp
		defer os.Remove(tmp)
	}

	// Prefer sidecar COMPONENTS.txt next to archive (works even if list not in tar yet).
	comps := readComponentsSidecarNear(archive)
	if len(comps) == 0 {
		comps = extractComponentsFromTar(src)
	}
	// Always union with primary baseline so a sparse COMPONENTS.txt cannot skip core stack.
	comps = MergeComponents(DefaultComponents(), comps)
	fmt.Fprintf(os.Stderr, "recover: components %v\n", comps)

	// BEFORE install/harden: operator pubkeys from the backup archive (primary source),
	// then optional env. Harden disables password — keys must exist first.
	injectOperatorKeysFromBackupEarly(src)
	injectOperatorKeysFromEnvEarly()

	// Install first (binaries/services). Secrets not required yet for most steps.
	if err := Run(Options{Components: comps, SkipHostname: true}); err != nil {
		fmt.Fprintf(os.Stderr, "recover: install warnings: %v\n", err)
		// continue — restore may still fix secrets
	}

	// Then overlay data from archive (secrets, users, state, lampac config).
	if err := run("tar", "-xzf", src, "-C", "/"); err != nil {
		parent := filepath.Dir(paths.EtcDir())
		if err2 := run("tar", "-xzf", src, "-C", parent); err2 != nil {
			return fmt.Errorf("restore data: %w", err)
		}
	}
	if key != "" {
		_ = writeSecret("backup_key", key)
	}
	_ = WriteComponentsManifest(comps)

	// Hostname from backup (node_id / hostname.backup) — never invent nd-primary-* on recover
	restoreHostnameFromBackup()
	// Operator pubkeys from backup + env (never private keys)
	restoreOperatorKeysFromBackup()
	reloadSSHDAfterRecover()

	// Second pass: components that need secrets or were skipped when pre-restore install aborted mid-list.
	fmt.Fprintln(os.Stderr, "recover: post-restore component pass")
	post := []string{}
	for _, c := range comps {
		switch c {
		case "vpn-users", "api", "metrics", "telegram", "backup", "lampac", "registry", "git":
			post = append(post, c)
		}
	}
	if len(post) > 0 {
		if err := Run(Options{Components: post, SkipHostname: true}); err != nil {
			fmt.Fprintf(os.Stderr, "recover: post-restore install: %v\n", err)
		}
	}

	// Re-apply runtime configs from restored secrets/users
	fmt.Fprintln(os.Stderr, "recover: re-issue LE if DOMAIN+LE_EMAIL in conf (certs not in backup)")
	if err := ReissueLEAfterRecover(); err != nil {
		fmt.Fprintf(os.Stderr, "recover: LE reissue: %v (redirect may stay down until domain set --le)\n", err)
	}
	fmt.Fprintln(os.Stderr, "recover: ensure redirect unit")
	_ = InstallRedirect()
	fmt.Fprintln(os.Stderr, "recover: ensure relay-uplink user + apply vpn from secrets (must match secondary bundle)")
	_ = run("netductor", "vpn", "ensure-relay-uplink")
	if err := run("netductor", "vpn", "apply"); err != nil {
		fmt.Fprintf(os.Stderr, "recover: vpn apply FAILED (primary conf may not match secrets): %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "recover: vpn apply OK (Reality+users from secrets/registry)")
	}
	for _, u := range []string{"sing-box", "blocky", "netductor-api", "netductor-telegram-bot", "netductor-backup.timer"} {
		_ = run("systemctl", "enable", "--now", u)
		_ = run("systemctl", "try-restart", u)
	}
	// lampac: if component listed, ensure container up (data already restored under opt)
	for _, c := range comps {
		if c == "lampac" {
			_ = InstallLampac()
			break
		}
	}
	// Agent plane: allow known secondary public IPs on :8789 (restored registry + api-allow.cidr)
	_ = SyncAgentAllowFromSecondaryRegistry()
	_ = ApplyAgentFirewall()
	fmt.Fprintln(os.Stderr, "recover: done")
	return nil
}

func readComponentsSidecarNear(archive string) []string {
	dir := filepath.Dir(archive)
	for _, name := range []string{"COMPONENTS.txt", "components.txt"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var out []string
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				out = append(out, line)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func extractComponentsFromTar(tarPath string) []string {
	// try members with components.json
	out, err := runOut("tar", "-xOf", tarPath, "var/lib/netductor/components.json")
	if err != nil {
		out, err = runOut("tar", "-xOf", tarPath, "./var/lib/netductor/components.json")
	}
	if err != nil || out == "" {
		return nil
	}
	var m ComponentsManifest
	if json.Unmarshal([]byte(out), &m) != nil {
		return nil
	}
	return m.Components
}

func pruneBackups(dir string, keep int) {
	if keep < 1 {
		keep = 1
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".ndenc") || strings.HasSuffix(n, ".tar.gz") {
			names = append(names, n)
		}
	}
	sort.Strings(names) // stamp in name → chronological
	if len(names) <= keep {
		return
	}
	for _, n := range names[:len(names)-keep] {
		_ = os.Remove(filepath.Join(dir, n))
	}
}

// PrunePeerBackups trims secondary offsite copies under peers/core.
func PrunePeerBackups(keep int) {
	pruneBackups("/var/lib/netductor/backups/peers/core", keep)
}

// RecoverFromSecondary downloads latest .ndenc from secondary recovery API then Recover().
func RecoverFromSecondary(baseURL, recoveryToken, keyArg string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || recoveryToken == "" {
		return fmt.Errorf("usage: netductor recover --from-secondary URL --recovery-token TOKEN [--key KEY]")
	}
	// Self-signed recovery TLS is default on secondary; allow skip-verify for this DR path only.
	client := &http.Client{
		Timeout: 30 * time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // recovery self-signed
		},
	}
	get := func(path string) ([]byte, http.Header, error) {
		req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Authorization", "Bearer "+recoveryToken)
		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return nil, nil, fmt.Errorf("%s: HTTP %d %s", path, resp.StatusCode, string(b))
		}
		return b, resp.Header, err
	}
	body, hdr, err := get("/recovery/latest")
	if err != nil {
		return err
	}
	name := hdr.Get("X-Netductor-Backup-Name")
	if name == "" {
		name = "from-secondary.ndenc"
	}
	tmpDir, err := os.MkdirTemp("", "nd-recover-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	arch := filepath.Join(tmpDir, name)
	if err := os.WriteFile(arch, body, 0o600); err != nil {
		return err
	}
	// Decryption key: operator-supplied only (offline). Wire key fetch is opt-in and discouraged.
	if keyArg == "" {
		keyArg = strings.TrimSpace(os.Getenv("NETDUCTOR_BACKUP_KEY"))
	}
	// RECOVERY_FETCH_KEY removed — key must be offline

	if keyArg == "" {
		return fmt.Errorf("backup key required: pass --key or NETDUCTOR_BACKUP_KEY (not served by secondary unless SERVE_KEY+FETCH_KEY)")
	}
	if cb, _, err := get("/recovery/components"); err == nil && len(cb) > 0 {
		_ = os.WriteFile(filepath.Join(tmpDir, "COMPONENTS.txt"), cb, 0o644)
	}
	return Recover(arch, keyArg)
}

// snapshotHostnameForBackup records the live hostname so recover can restore it.
func snapshotHostnameForBackup() {
	hn := ""
	if b, err := os.ReadFile("/etc/hostname"); err == nil {
		hn = strings.TrimSpace(string(b))
	}
	if hn == "" {
		if out, err := runOut("hostname"); err == nil {
			hn = strings.TrimSpace(out)
		}
	}
	if hn == "" {
		return
	}
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "hostname.backup"), []byte(hn+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "node_id"), []byte(hn+"\n"), 0o644)
}

// snapshotOperatorKeysForBackup stores authorized public keys under etc/netductor (in backup).
// Private keys are never stored.
func snapshotOperatorKeysForBackup() {
	src := "/root/.ssh/authorized_keys"
	b, err := os.ReadFile(src)
	if err != nil || len(b) == 0 {
		return
	}
	var lines []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "ssh-") || strings.HasPrefix(line, "ecdsa-") {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return
	}
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	path := filepath.Join(paths.EtcDir(), "operator_authorized_keys")
	prev, _ := os.ReadFile(path)
	seen := map[string]bool{}
	var out []string
	for _, line := range append(strings.Split(string(prev), "\n"), lines...) {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		out = append(out, line)
	}
	_ = os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o600)
}

// restoreHostnameFromBackup applies hostname from restored etc (no auto nd-primary-* invent).
func restoreHostnameFromBackup() {
	name := ""
	for _, p := range []string{
		filepath.Join(paths.EtcDir(), "hostname.backup"),
		filepath.Join(paths.EtcDir(), "node_id"),
		"/etc/hostname",
	} {
		if b, err := os.ReadFile(p); err == nil {
			name = strings.TrimSpace(string(b))
			if name != "" {
				break
			}
		}
	}
	if name == "" {
		fmt.Fprintln(os.Stderr, "recover: no hostname in backup — leave system hostname unchanged")
		return
	}
	name = strings.ToLower(name)
	fmt.Fprintf(os.Stderr, "recover: hostname=%s (from backup)\n", name)
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "node_id"), []byte(name+"\n"), 0o644)
	_ = os.WriteFile("/etc/hostname", []byte(name+"\n"), 0o644)
	_ = run("hostnamectl", "set-hostname", name)
	_ = run("bash", "-c", fmt.Sprintf(
		`grep -q '%s' /etc/hosts || echo '127.0.1.1 %s' >> /etc/hosts`, name, name))
}

// restoreOperatorKeysFromBackup writes pubkeys into /root/.ssh/authorized_keys.
// Sources (public keys only): backup operator_authorized_keys, NETDUCTOR_OPERATOR_PUBKEY, NETDUCTOR_OPERATOR_PUBKEY_FILE.
func restoreOperatorKeysFromBackup() {
	var lines []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		if !(strings.HasPrefix(s, "ssh-") || strings.HasPrefix(s, "ecdsa-")) {
			return
		}
		seen[s] = true
		lines = append(lines, s)
	}
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "operator_authorized_keys")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			add(line)
		}
	}
	add(os.Getenv("NETDUCTOR_OPERATOR_PUBKEY"))
	if fp := strings.TrimSpace(os.Getenv("NETDUCTOR_OPERATOR_PUBKEY_FILE")); fp != "" {
		if b, err := os.ReadFile(fp); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				add(line)
			}
		}
	}
	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "recover: no operator pubkeys in backup/env — ensure console access before harden")
		return
	}
	_ = os.MkdirAll("/root/.ssh", 0o700)
	path := "/root/.ssh/authorized_keys"
	prev, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(prev), "\n") {
		add(line)
	}
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "recover: authorized_keys: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "recover: installed %d operator pubkey(s)\n", len(lines))
}

// injectOperatorKeysFromBackupEarly extracts operator_authorized_keys (and optional
// root/.ssh/authorized_keys) from the decrypted tar before harden runs.
// Full tar extract still happens after install; this only peeks public keys.
func injectOperatorKeysFromBackupEarly(tarPath string) {
	if tarPath == "" {
		return
	}
	var blobs []string
	for _, member := range []string{
		"etc/netductor/operator_authorized_keys",
		"./etc/netductor/operator_authorized_keys",
		"root/.ssh/authorized_keys",
		"./root/.ssh/authorized_keys",
	} {
		out, err := runOut("tar", "-xOf", tarPath, member)
		if err != nil || strings.TrimSpace(out) == "" {
			continue
		}
		blobs = append(blobs, out)
	}
	if len(blobs) == 0 {
		fmt.Fprintln(os.Stderr, "recover: no operator pubkeys inside archive yet — try env or wait for full restore")
		return
	}
	sshDir := "/root/.ssh"
	_ = os.MkdirAll(sshDir, 0o700)
	ak := filepath.Join(sshDir, "authorized_keys")
	existing, _ := os.ReadFile(ak)
	seen := map[string]bool{}
	var lines []string
	for _, block := range append([]string{string(existing)}, blobs...) {
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if seen[line] {
				continue
			}
			seen[line] = true
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return
	}
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(ak, []byte(body), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "recover: write authorized_keys early:", err)
		return
	}
	_ = os.Chmod(sshDir, 0o700)
	fmt.Fprintf(os.Stderr, "recover: early operator pubkey(s) from backup=%d (before harden)\n", len(lines))
}

// injectOperatorKeysFromEnvEarly merges optional NETDUCTOR_OPERATOR_PUBKEY after backup peek.
func injectOperatorKeysFromEnvEarly() {
	var lines []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		if !(strings.HasPrefix(s, "ssh-") || strings.HasPrefix(s, "ecdsa-")) {
			return
		}
		seen[s] = true
		lines = append(lines, s)
	}
	add(os.Getenv("NETDUCTOR_OPERATOR_PUBKEY"))
	if fp := strings.TrimSpace(os.Getenv("NETDUCTOR_OPERATOR_PUBKEY_FILE")); fp != "" {
		if b, err := os.ReadFile(fp); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				add(line)
			}
		}
	}
	prev, _ := os.ReadFile("/root/.ssh/authorized_keys")
	for _, line := range strings.Split(string(prev), "\n") {
		add(line)
	}
	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "recover: no extra NETDUCTOR_OPERATOR_PUBKEY (backup keys already applied if present)")
		return
	}
	_ = os.MkdirAll("/root/.ssh", 0o700)
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile("/root/.ssh/authorized_keys", []byte(body), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "recover: early authorized_keys: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "recover: early operator pubkey(s)=%d (before harden)\n", len(lines))
}

func reloadSSHDAfterRecover() {
	_ = run("systemctl", "reload", "ssh")
	_ = run("systemctl", "reload", "sshd")
	_ = run("service", "ssh", "reload")
}
