package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// VerifyLatestBackup decrypts the newest .ndenc (or lists .tar.gz) and runs tar -tf.
// Does not restore to live paths.
func VerifyLatestBackup() (string, error) {
	dir := filepath.Join(paths.StateDir(), "backups")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("backups dir: %w", err)
	}
	var newest string
	var newestT time.Time
	for _, e := range ents {
		n := e.Name()
		if !strings.HasSuffix(n, ".ndenc") && !strings.HasSuffix(n, ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if newest == "" || info.ModTime().After(newestT) {
			newest = filepath.Join(dir, n)
			newestT = info.ModTime()
		}
	}
	if newest == "" {
		return "", fmt.Errorf("no backup files in %s", dir)
	}
	src := newest
	tmp := ""
	if strings.HasSuffix(newest, ".ndenc") {
		key := strings.TrimSpace(os.Getenv("NETDUCTOR_BACKUP_KEY"))
		if key == "" {
			key = readSecret("backup_key")
		}
		if key == "" {
			return "", fmt.Errorf("backup_key missing for verify")
		}
		tmp = filepath.Join(os.TempDir(), fmt.Sprintf("nd-verify-%d.tar.gz", time.Now().UnixNano()))
		if err := decryptFile(newest, tmp, key); err != nil {
			return "", fmt.Errorf("decrypt %s: %w", filepath.Base(newest), err)
		}
		defer os.Remove(tmp)
		src = tmp
	}
	out, err := exec.Command("tar", "-tzf", src).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tar list: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	n := len(lines)
	if n > 5000 {
		// still ok
	}
	msg := fmt.Sprintf("ok file=%s members=%d age=%s", filepath.Base(newest), n, time.Since(newestT).Round(time.Minute))
	_ = os.MkdirAll(filepath.Join(paths.StateDir(), "backup-verify"), 0o700)
	_ = os.WriteFile(filepath.Join(paths.StateDir(), "backup-verify", "last.txt"), []byte(msg+"\n"+time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600)
	return msg, nil
}

// AlertVerifyFailure notifies once per day key.
func AlertVerifyFailure(err error) {
	if err == nil {
		notify.ClearAlert("backup:verify")
		return
	}
	notify.AlertOnce("backup:verify", fmt.Sprintf("🔴 Backup verify failed: %v", err))
}

// InstallBackupVerifyTimer weekly Sunday 03:15 UTC.
func InstallBackupVerifyTimer() error {
	bin, _ := os.Executable()
	if bin == "" {
		bin = "/usr/local/bin/netductor"
	}
	unit := fmt.Sprintf(`[Unit]
Description=Netductor backup verify once
[Service]
Type=oneshot
ExecStart=%s backup verify
`, bin)
	timer := `[Unit]
Description=Netductor weekly backup verify
[Timer]
OnCalendar=Sun *-*-* 03:15:00
Persistent=true
[Install]
WantedBy=timers.target
`
	if err := writeUnit("netductor-backup-verify.service", unit); err != nil {
		return err
	}
	if err := writeUnit("netductor-backup-verify.timer", timer); err != nil {
		return err
	}
	_ = run("systemctl", "enable", "--now", "netductor-backup-verify.timer")
	return nil
}


// LastBackupVerify returns last verify status file contents.
func LastBackupVerify() string {
	b, err := os.ReadFile(filepath.Join(paths.StateDir(), "backup-verify", "last.txt"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
