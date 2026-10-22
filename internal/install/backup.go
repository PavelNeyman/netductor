package install

import (
	"crypto/aes"
	"encoding/binary"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
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
		if err := writeSecret("backup_key", randomHex(32)); err != nil {
			return fmt.Errorf("backup_key: %w", err)
		}
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
	// Large files: chunked AES-GCM stream (NDG2) — authenticated.
	const maxInMem = 64 << 20 // 64 MiB
	if st.Size() > maxInMem {
		return encryptFileStreamGCM(inPath, outPath, pass)
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

// NDG2: chunked AES-GCM. Magic "NDG2" + chunkSize(BE u32) + (nonce12 | ctLen BE u32 | ciphertext)*.
const ndg2Magic = "NDG2"
const ndg2Chunk = 1 << 20 // 1 MiB

func encryptFileStreamGCM(inPath, outPath, pass string) error {
	in, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	block, err := aes.NewCipher(keyBytes(pass))
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	if _, err := out.Write([]byte(ndg2Magic)); err != nil {
		return err
	}
	var sz [4]byte
	binary.BigEndian.PutUint32(sz[:], uint32(ndg2Chunk))
	if _, err := out.Write(sz[:]); err != nil {
		return err
	}
	buf := make([]byte, ndg2Chunk)
	for {
		n, rerr := io.ReadFull(in, buf)
		if n > 0 {
			nonce := make([]byte, gcm.NonceSize())
			if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
				return err
			}
			ct := gcm.Seal(nil, nonce, buf[:n], nil)
			if _, err := out.Write(nonce); err != nil {
				return err
			}
			binary.BigEndian.PutUint32(sz[:], uint32(len(ct)))
			if _, err := out.Write(sz[:]); err != nil {
				return err
			}
			if _, err := out.Write(ct); err != nil {
				return err
			}
		}
		if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	// Authenticated end marker: random nonce + zero length (distinguishes truncation).
	endNonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, endNonce); err != nil {
		return err
	}
	if _, err := out.Write(endNonce); err != nil {
		return err
	}
	binary.BigEndian.PutUint32(sz[:], 0)
	if _, err := out.Write(sz[:]); err != nil {
		return err
	}
	return nil
}

func decryptFileStreamGCM(inPath, outPath, pass string) error {
	in, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer in.Close()
	mag := make([]byte, 4)
	if _, err := io.ReadFull(in, mag); err != nil {
		return err
	}
	if string(mag) != ndg2Magic {
		return fmt.Errorf("not NDG2")
	}
	var sz [4]byte
	if _, err := io.ReadFull(in, sz[:]); err != nil {
		return err
	}
	_ = binary.BigEndian.Uint32(sz[:])
	block, err := aes.NewCipher(keyBytes(pass))
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	nonce := make([]byte, gcm.NonceSize())
	chunks := 0
	for {
		n, err := io.ReadFull(in, nonce)
		if err == io.EOF {
			break // clean end only with zero bytes read by ReadFull → EOF
		}
		if err == io.ErrUnexpectedEOF {
			return fmt.Errorf("ndg2: truncated nonce (%d/%d bytes)", n, len(nonce))
		}
		if err != nil {
			return err
		}
		if _, err := io.ReadFull(in, sz[:]); err != nil {
			return fmt.Errorf("ndg2: truncated chunk len: %w", err)
		}
		ctLen := binary.BigEndian.Uint32(sz[:])
		// Final marker: ctLen == 0 after full nonce → authenticated end of stream
		if ctLen == 0 {
			// optional: remaining must be EOF
			var one [1]byte
			if _, err := io.ReadFull(in, one[:]); err != io.EOF && err != io.ErrUnexpectedEOF {
				if err == nil {
					return fmt.Errorf("ndg2: trailing data after end marker")
				}
				return err
			}
			return nil
		}
		if ctLen > (ndg2Chunk + uint32(gcm.Overhead()) + 64) {
			return fmt.Errorf("ndg2: bad chunk len %d", ctLen)
		}
		ct := make([]byte, ctLen)
		if _, err := io.ReadFull(in, ct); err != nil {
			return fmt.Errorf("ndg2: truncated ciphertext: %w", err)
		}
		plain, err := gcm.Open(nil, nonce, ct, nil)
		if err != nil {
			return fmt.Errorf("ndg2 auth: %w", err)
		}
		if _, err := out.Write(plain); err != nil {
			return err
		}
		chunks++
	}
	// End marker required (nonce + zero length). Plain EOF after data = truncated or legacy — reject.
	if chunks == 0 {
		return fmt.Errorf("ndg2: empty ciphertext")
	}
	return fmt.Errorf("ndg2: missing end marker (truncated or pre-0.9.471 archive — re-backup)")
}

// encryptFileOpenSSL: legacy only (decrypt still supports old CBC archives).
func encryptFileOpenSSL(inPath, outPath, pass string) error {
	cmd := exec.Command("openssl", "enc", "-aes-256-cbc", "-salt", "-pbkdf2",
		"-in", inPath, "-out", outPath, "-pass", "env:NETDUCTOR_BACKUP_PASS")
	cmd.Env = append(os.Environ(), "NETDUCTOR_BACKUP_PASS="+pass)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("openssl enc: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return os.Chmod(outPath, 0o600)
}

func decryptFile(inPath, outPath, pass string) error {
	const gcmTryMax = 64 << 20 // never ReadFile multi-GB into RAM
	st, err := os.Stat(inPath)
	if err != nil {
		return err
	}
	if f, e := os.Open(inPath); e == nil {
		mag := make([]byte, 4)
		_, _ = io.ReadFull(f, mag)
		_ = f.Close()
		if string(mag) == ndg2Magic {
			return decryptFileStreamGCM(inPath, outPath, pass)
		}
	}
	// Try legacy in-memory GCM only for small archives.
	if st.Size() > 0 && st.Size() < gcmTryMax {
		in, err := os.ReadFile(inPath)
		if err != nil {
			return err
		}
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
	// openssl / large or GCM miss — R10: pass via env, not argv
	cmd := exec.Command("openssl", "enc", "-d", "-aes-256-cbc", "-pbkdf2",
		"-in", inPath, "-out", outPath, "-pass", "env:NETDUCTOR_BACKUP_PASS")
	cmd.Env = append(os.Environ(), "NETDUCTOR_BACKUP_PASS="+pass)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("decrypt: gcm and openssl failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return os.Chmod(outPath, 0o600)
}

// WaitForBackupPull waits for online secondaries to ack backup_pull with key=ok.
// If cmdID is non-empty, LastCmdLog must contain cmd_id=<cmdID> (ties ack to this run).
func WaitForBackupPull(timeout time.Duration, notBefore time.Time, cmdID string) (acked, pending int) {
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	if notBefore.IsZero() {
		notBefore = time.Now().UTC().Add(-30 * time.Second)
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
			cmdOK := strings.HasPrefix(d.LastCmd, "backup_pull") && d.LastCmdOK &&
				!d.LastCmdAt.Before(notBefore) &&
				strings.Contains(d.LastCmdLog, "key=ok")
			if cmdOK && cmdID != "" {
				cmdOK = strings.Contains(d.LastCmdLog, "cmd_id="+cmdID)
			}
			if cmdOK {
				acked++
				continue
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

func newBackupPullCmdID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
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
		"--exclude=var/lib/netductor/ci-cache",
		"--exclude=var/lib/netductor/releases",
		"--exclude=var/lib/netductor/agents",
		"--exclude=var/lib/netductor/build-backup-mirror",
		"--exclude=var/lib/netductor/stack",
		"--exclude=var/lib/netductor/lampac/ts",
		"--exclude=var/lib/docker",
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
	if err := run("tar", args...); err != nil {
		_ = os.Remove(plain)
		etc := paths.EtcDir()
		if err2 := run("tar", "-czf", plain, "-C", filepath.Dir(etc), filepath.Base(etc)); err2 != nil {
			_ = os.Remove(plain)
			return "", fmt.Errorf("tar failed: %v (fallback: %v)", err, err2)
		}
	}
	if st, err := os.Stat(plain); err != nil || st.Size() == 0 {
		_ = os.Remove(plain)
		return "", fmt.Errorf("tar failed: empty archive")
	}
	key := strings.TrimSpace(readSecret("backup_key"))
	if key == "" {
		_ = os.Remove(plain)
		return "", fmt.Errorf("backup_key missing — refuse plaintext backup (set secrets/backup_key)")
	}
	enc := plain + ".ndenc"
	if err := encryptFile(plain, enc, key); err != nil {
		_ = os.Remove(plain)
		return "", err
	}
	_ = os.Remove(plain)
	out := enc
	_ = os.Chmod(out, 0o600)
	// Recovery key + components list alongside archive (plain — needed before decrypt on bare metal).
	_ = writeBackupKeyRecovery()
	_ = writeComponentsSidecar(dir)
	failMark := filepath.Join(paths.StateDir(), "backup_offsite_fail")
	_ = os.Remove(failMark) // cleared unless agent pull fails below
	// Agent pull to RU secondary (HTTPS mTLS) — no primary→secondary SSH.
	n, offline := 0, 0
	pullID := newBackupPullCmdID()
	for _, d := range secondary.List() {
		if !secondary.Online(d, 3*time.Minute) {
			offline++
			continue
		}
		if err := secondary.EnqueueCmd(d.ID, "backup_pull:"+pullID); err == nil {
			n++
		}
	}
	if n > 0 {
		fmt.Fprintf(os.Stderr, "backup: queued backup_pull on %d secondary agent(s)\n", n)
		// Do not clear offsite alert on enqueue alone — wait for this run's ack + key.
		started := time.Now().UTC().Add(-2 * time.Second)
		acked, pending := WaitForBackupPull(90*time.Second, started, pullID)
		if acked > 0 && pending == 0 {
			fmt.Fprintf(os.Stderr, "backup: offsite confirmed (%d secondary acked backup_pull with key)\n", acked)
			notify.ClearAlert("backup-offsite")
		} else {
			msg := fmt.Sprintf("🔴 Backup offsite: queued on %d, acked=%d pending=%d (pull/key not verified)", n, acked, pending)
			fmt.Fprintln(os.Stderr, msg)
			notify.AlertOnce("backup-offsite", msg)
			_ = os.WriteFile(failMark, []byte(msg), 0o600)
		}
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
		tmpDir, err := os.MkdirTemp("", "nd-restore-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)
		_ = os.Chmod(tmpDir, 0o700)
		tmp = filepath.Join(tmpDir, "restore.tar.gz")
		if err := decryptFile(archive, tmp, key); err != nil {
			return err
		}
		src = tmp
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
		tmpDir, err := os.MkdirTemp("", "nd-recover-plain-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)
		_ = os.Chmod(tmpDir, 0o700)
		tmp = filepath.Join(tmpDir, "recover.tar.gz")
		if err := decryptFile(archive, tmp, key); err != nil {
			return err
		}
		src = tmp
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
	var critical []string
	if err := run("netductor", "vpn", "apply"); err != nil {
		fmt.Fprintf(os.Stderr, "recover: vpn apply FAILED (primary conf may not match secrets): %v\n", err)
		critical = append(critical, "vpn apply: "+err.Error())
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
	if err := ApplyAgentFirewall(); err != nil {
		fmt.Fprintf(os.Stderr, "recover: agent firewall: %v\n", err)
		critical = append(critical, "firewall: "+err.Error())
	}
	if len(critical) > 0 {
		msg := "recover incomplete: " + strings.Join(critical, "; ")
		fmt.Fprintln(os.Stderr, msg)
		notify.AlertOnce("backup:drill-fail", "🔴 Recover incomplete — drill not marked: "+strings.Join(critical, "; "))
		return fmt.Errorf("%s", msg)
	}
	if err := MarkRecoverDrill(); err != nil {
		fmt.Fprintf(os.Stderr, "recover: drill-mark: %v\n", err)
		return err
	}
	fmt.Fprintln(os.Stderr, "recover: drill-mark OK (recover-drill-last.txt)")
	notify.AlertOnce("backup:drill-ok", "✅ Recover drill marked (auto after successful recover)")
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

// MarkRecoverDrill records a successful recovery/restore drill timestamp for fleet digest.
func MarkRecoverDrill() error {
	p := filepath.Join(paths.StateDir(), "recover-drill-last.txt")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	line := fmt.Sprintf("%d\n", time.Now().Unix())
	return os.WriteFile(p, []byte(line), 0o600)
}

// RecoverFromBuildHost downloads latest .ndenc from build-host recovery API then Recover().
func RecoverFromBuildHost(baseURL, recoveryToken, keyArg string) error {
	return RecoverFromSecondary(baseURL, recoveryToken, keyArg)
}

func caPathFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_CA")); v != "" {
		return v
	}
	return "/etc/netductor/secrets/recovery_peer.crt"
}

// recoveryHTTPClient: pin required (CA PEM or cert fingerprint). No silent insecure.
// Order: NETDUCTOR_RECOVERY_CA → secrets/recovery_peer.crt → NETDUCTOR_RECOVERY_FINGERPRINT / secrets/recovery_peer.fp
// Escape: NETDUCTOR_RECOVERY_INSECURE=1 only (stderr WARN).
func recoveryHTTPClient() (*http.Client, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	caPath := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_CA"))
	if caPath == "" {
		caPath = "/etc/netductor/secrets/recovery_peer.crt"
	}
	if b, err := os.ReadFile(caPath); err == nil && len(b) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("recovery CA pin present but not valid PEM (%s)", caPath)
		}
		cfg.RootCAs = pool
	} else if fp := recoveryFingerprint(); fp != "" {
		want := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(fp), "sha256:"))
		cfg.InsecureSkipVerify = true //nolint:gosec // replaced by VerifyPeerCertificate
		cfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("recovery: empty peer cert")
			}
			sum := sha256.Sum256(rawCerts[0])
			got := fmt.Sprintf("%x", sum[:])
			if got != want {
				return fmt.Errorf("recovery fingerprint mismatch (got sha256:%s)", got)
			}
			return nil
		}
	} else if os.Getenv("NETDUCTOR_RECOVERY_INSECURE") == "1" {
		cfg.InsecureSkipVerify = true //nolint:gosec
		fmt.Fprintln(os.Stderr, "WARN recovery: NETDUCTOR_RECOVERY_INSECURE=1 — TLS not verified")
	} else {
		return nil, fmt.Errorf("recovery pin required: set NETDUCTOR_RECOVERY_CA, place %s, or NETDUCTOR_RECOVERY_FINGERPRINT=sha256:… (or INSECURE=1 for first contact)", caPath)
	}
	return &http.Client{
		Timeout:   30 * time.Minute,
		Transport: &http.Transport{TLSClientConfig: cfg},
	}, nil
}

func recoveryFingerprint() string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_FINGERPRINT")); v != "" {
		return v
	}
	b, err := os.ReadFile("/etc/netductor/secrets/recovery_peer.fp")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// RecoverFromSecondary downloads latest .ndenc from secondary recovery API then Recover().
func RecoverFromSecondary(baseURL, recoveryToken, keyArg string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || recoveryToken == "" {
		return fmt.Errorf("usage: netductor recover --from-secondary URL --recovery-token TOKEN [--key KEY]")
	}
	if !strings.HasPrefix(baseURL, "https://") {
		return fmt.Errorf("recovery URL must be https:// (got %s)", baseURL)
	}
	client, err := recoveryHTTPClient()
	if err != nil {
		return err
	}
	getSmall := func(path string, max int64) ([]byte, http.Header, error) {
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
		b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
		if resp.StatusCode >= 300 {
			return nil, nil, fmt.Errorf("%s: HTTP %d %s", path, resp.StatusCode, string(b))
		}
		if int64(len(b)) > max {
			return nil, nil, fmt.Errorf("%s: response exceeds %d bytes", path, max)
		}
		return b, resp.Header, err
	}
	tmpDir, err := os.MkdirTemp("", "nd-recover-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	req, err := http.NewRequest(http.MethodGet, baseURL+"/recovery/latest", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+recoveryToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("/recovery/latest: HTTP %d %s", resp.StatusCode, string(b))
	}
	name := resp.Header.Get("X-Netductor-Backup-Name")
	if name == "" {
		name = "from-secondary.ndenc"
	}
	name = filepath.Base(name)
	arch := filepath.Join(tmpDir, name)
	f, err := os.OpenFile(arch, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	const maxArch = 512 << 20
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxArch+1))
	_ = f.Close()
	if err != nil {
		return err
	}
	if n > maxArch {
		_ = os.Remove(arch)
		return fmt.Errorf("recovery archive exceeds %d bytes", maxArch)
	}
	// Decryption key: operator-supplied only (offline). Wire key fetch is opt-in and discouraged.
	if keyArg == "" {
		keyArg = strings.TrimSpace(os.Getenv("NETDUCTOR_BACKUP_KEY"))
	}
	// RECOVERY_FETCH_KEY removed — key must be offline

	if keyArg == "" {
		return fmt.Errorf("backup key required: pass --key or NETDUCTOR_BACKUP_KEY (not served by secondary unless SERVE_KEY+FETCH_KEY)")
	}
	if cb, _, err := getSmall("/recovery/components", 1<<20); err == nil && len(cb) > 0 {
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

// validHostnameStrict: DNS-like labels only (recovery trust boundary — no shell metacharacters).
func validHostnameStrict(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 253 {
		return false
	}
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '.' {
			continue
		}
		return false
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || strings.Contains(name, "..") {
		return false
	}
	return true
}

// restoreHostnameFromBackup applies hostname from restored etc (no auto nd-primary-* invent).
// Never shells out with archive data (P1.1).
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
	if !validHostnameStrict(name) {
		fmt.Fprintf(os.Stderr, "recover: refuse hostname %q (invalid charset)\n", name)
		return
	}
	fmt.Fprintf(os.Stderr, "recover: hostname=%s (from backup)\n", name)
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "node_id"), []byte(name+"\n"), 0o644)
	_ = os.WriteFile("/etc/hostname", []byte(name+"\n"), 0o644)
	_ = run("hostnamectl", "set-hostname", name)
	ensureHostsLocalLine(name)
}

// ensureHostsLocalLine adds "127.0.1.1 <name>" without invoking a shell.
func ensureHostsLocalLine(name string) {
	const path = "/etc/hosts"
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	needle := "127.0.1.1 " + name
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == needle || strings.HasPrefix(line, needle+" ") || strings.HasPrefix(line, needle+"\t") {
			return
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(needle + "\n")
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
