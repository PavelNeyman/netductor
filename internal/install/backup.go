package install

import (
	"encoding/json"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func loadOffsite() (method, target, extra string) {
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "backup.offsite"))
	if err != nil {
		return "", "", ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "method":
			method = strings.TrimSpace(v)
		case "target":
			target = strings.TrimSpace(v)
		case "scp_opts", "extra":
			extra = strings.TrimSpace(v)
		}
	}
	return method, target, extra
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

func uploadOffsite(localPath string) error {
	method, target, extra := loadOffsite()
	if method == "" || target == "" {
		return nil
	}
	switch method {
	case "scp":
		args := []string{}
		if extra != "" {
			args = append(args, strings.Fields(extra)...)
		}
		args = append(args, localPath, target)
		if err := run("scp", args...); err != nil {
			return err
		}
		for _, side := range []string{"BACKUP_KEY.txt", "COMPONENTS.txt"} {
			sidePath := filepath.Join(paths.StateDir(), "backups", side)
			if st, err := os.Stat(sidePath); err == nil && st.Size() > 0 {
				kargs := append([]string{}, args[:len(args)-2]...)
				kargs = append(kargs, sidePath, target)
				_ = run("scp", kargs...)
			}
		}
		return nil
	case "rsync":
		args := []string{"-az"}
		if extra != "" {
			args = append(args, strings.Fields(extra)...)
		}
		args = append(args, localPath, target)
		return run("rsync", args...)
	case "http", "https", "curl":
		return run("curl", "-fsS", "-X", "PUT", "--data-binary", "@"+localPath, target)
	default:
		return fmt.Errorf("unknown offsite method %s", method)
	}
}

func keyBytes(pass string) []byte {
	h := sha256.Sum256([]byte(pass))
	return h[:]
}

func encryptFile(inPath, outPath, pass string) error {
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

func decryptFile(inPath, outPath, pass string) error {
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
	if len(in) < gcm.NonceSize() {
		return fmt.Errorf("ciphertext too short")
	}
	nonce, ct := in[:gcm.NonceSize()], in[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, plain, 0o600)
}

func Backup() (string, error) {
	dir := filepath.Join(paths.StateDir(), "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	plain := filepath.Join(dir, "netductor-"+stamp+".tar.gz")
	SyncComponentsFromDisk()
	// Ensure manifest exists before packing
	if len(ReadComponentsManifest()) == 0 {
		_ = WriteComponentsManifest(DefaultComponents())
	}
	args := []string{"-czf", plain, "-C", "/", "etc/netductor"}
	// Config + data for all managed services (binaries/images reinstalled from COMPONENTS).
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
	if err := uploadOffsite(out); err != nil {
		fmt.Fprintf(os.Stderr, "offsite: %v\n", err)
		_ = os.WriteFile(failMark, []byte(err.Error()), 0o600)
	} else {
		_ = os.Remove(failMark)
	}
	// Agent pull to RU secondary (HTTPS mTLS) — no primary→secondary SSH.
	n := 0
	for _, d := range secondary.List() {
		if !secondary.Online(d, 3*time.Minute) {
			continue
		}
		if err := secondary.EnqueueCmd(d.ID, "backup_pull"); err == nil {
			n++
		}
	}
	if n > 0 {
		fmt.Fprintf(os.Stderr, "backup: queued backup_pull on %d secondary agent(s)\n", n)
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
	comps := ReadComponentsManifest()
	if len(comps) == 0 {
		comps = DefaultComponents()
	}
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
	if len(comps) == 0 {
		comps = DefaultComponents()
	}
	fmt.Fprintf(os.Stderr, "recover: components %v\n", comps)

	// Install first (binaries/services). Secrets not required yet for most steps.
	if err := Run(Options{Components: comps}); err != nil {
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

	// Re-apply runtime configs from restored secrets/users
	fmt.Fprintln(os.Stderr, "recover: apply vpn / restart services")
	_ = run("netductor", "vpn", "apply")
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
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) <= keep {
		return
	}
	for i := 0; i < len(entries)-keep; i++ {
		_ = os.Remove(filepath.Join(dir, entries[i].Name()))
	}
}


// RecoverFromSecondary downloads latest .ndenc from secondary recovery API then Recover().
func RecoverFromSecondary(baseURL, recoveryToken, keyArg string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || recoveryToken == "" {
		return fmt.Errorf("usage: netductor recover --from-secondary URL --recovery-token TOKEN [--key KEY]")
	}
	client := &http.Client{Timeout: 30 * time.Minute}
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
	if kb, _, err := get("/recovery/key"); err == nil && len(kb) > 0 {
		_ = os.WriteFile(filepath.Join(tmpDir, "BACKUP_KEY.txt"), kb, 0o600)
		if keyArg == "" {
			keyArg = strings.TrimSpace(string(kb))
		}
	}
	if cb, _, err := get("/recovery/components"); err == nil && len(cb) > 0 {
		_ = os.WriteFile(filepath.Join(tmpDir, "COMPONENTS.txt"), cb, 0o644)
	}
	return Recover(arch, keyArg)
}
