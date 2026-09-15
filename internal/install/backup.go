package install

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
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
OnCalendar=daily
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
		keyPath := filepath.Join(paths.StateDir(), "backups", "BACKUP_KEY.txt")
		if st, err := os.Stat(keyPath); err == nil && st.Size() > 0 {
			kargs := append([]string{}, args[:len(args)-2]...)
			kargs = append(kargs, keyPath, target)
			_ = run("scp", kargs...)
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
	etc := paths.EtcDir()
	_ = run("tar", "-czf", plain, "-C", filepath.Dir(etc), filepath.Base(etc))
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
	// Recovery key alongside archive on peer (core wipe cannot read key from encrypted blob).
	_ = writeBackupKeyRecovery()
	failMark := filepath.Join(paths.StateDir(), "backup_offsite_fail")
	if err := uploadOffsite(out); err != nil {
		fmt.Fprintf(os.Stderr, "offsite: %v\n", err)
		_ = os.WriteFile(failMark, []byte(err.Error()), 0o600)
	} else {
		_ = os.Remove(failMark)
	}
	pruneBackups(dir, 14)
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
	parent := filepath.Dir(paths.EtcDir())
	if err := run("tar", "-xzf", src, "-C", parent); err != nil {
		return err
	}
	// Ensure key is persisted for future backups after bare-metal restore.
	if key := strings.TrimSpace(keyArg); key != "" {
		_ = writeSecret("backup_key", key)
	} else if k := strings.TrimSpace(os.Getenv("NETDUCTOR_BACKUP_KEY")); k != "" {
		_ = writeSecret("backup_key", k)
	}
	return nil
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
