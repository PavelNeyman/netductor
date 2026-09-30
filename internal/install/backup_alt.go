package install

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// BackupSecondaryLocal archives secondary-critical state into
// /var/lib/netductor/backups/local/ (does not need primary API).
func BackupSecondaryLocal() (string, error) {
	dir := filepath.Join(paths.StateDir(), "backups", "local")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	plain := filepath.Join(dir, "secondary-"+stamp+".tar.gz")
	args := []string{
		"-czf", plain,
		"--exclude=var/lib/netductor/backups",
		"--exclude=var/lib/netductor/registry",
		"--exclude=var/lib/netductor/nvr",
		"--exclude=var/lib/netductor/git",
		"-C", "/",
	}
	for _, p := range []string{
		"etc/netductor",
		"etc/sing-box",
		"usr/local/etc/sing-box",
		"var/lib/netductor",
		"etc/netductor/svc-paths",
	} {
		if _, err := os.Stat("/" + p); err == nil {
			args = append(args, p)
		}
	}
	if err := run("tar", args...); err != nil {
		return "", fmt.Errorf("tar: %w", err)
	}
	st, err := os.Stat(plain)
	if err != nil || st.Size() == 0 {
		return "", fmt.Errorf("tar empty")
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
	pruneBackups(dir, BackupKeepCount())
	fmt.Fprintln(os.Stderr, "backup secondary-local:", out)
	return out, nil
}

// PushBackupRecovery uploads a local .ndenc to secondary recovery API (when armed).
// Works when primary API is down — only needs recovery :8790 + token.
func PushBackupRecovery(baseURL, token, filePath string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || token == "" || filePath == "" {
		return fmt.Errorf("usage: netductor backup push-recovery --url https://SEC:8790 --token TOKEN --file path.ndenc")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	req, err := http.NewRequest(http.MethodPost, baseURL+"/recovery/upload", f)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Netductor-Backup-Name", filepath.Base(filePath))
	client := &http.Client{
		Timeout: 30 * time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // recovery self-signed
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("upload HTTP %d: %s", resp.StatusCode, string(b))
	}
	fmt.Println(strings.TrimSpace(string(b)))
	return nil
}

// PushBackupSSH copies file to secondary peers/core via scp (emergency when API + recovery both awkward).
func PushBackupSSH(host, port, keyPath, filePath string) error {
	if host == "" || filePath == "" {
		return fmt.Errorf("usage: netductor backup push-ssh --host IP [--port 52222] [--key path] --file path.ndenc")
	}
	if port == "" {
		port = "52222"
	}
	if keyPath == "" {
		keyPath = os.Getenv("HOME") + "/.ssh/netductor_primary"
	}
	remoteDir := "/var/lib/netductor/backups/peers/core"
	// ensure dir
	sshArgs := []string{"-i", keyPath, "-p", port, "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=accept-new",
		"root@" + host, "mkdir -p " + remoteDir}
	if out, err := exec.Command("ssh", sshArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("ssh mkdir: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	scpArgs := []string{"-i", keyPath, "-P", port, "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=accept-new",
		filePath, "root@" + host + ":" + remoteDir + "/"}
	if out, err := exec.Command("scp", scpArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("scp: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	// also push key sidecar if present
	keySide := filepath.Join(filepath.Dir(filePath), "BACKUP_KEY.txt")
	if _, err := os.Stat(keySide); err == nil {
		_ = exec.Command("scp", "-i", keyPath, "-P", port, "-o", "IdentitiesOnly=yes",
			keySide, "root@"+host+":"+remoteDir+"/").Run()
	}
	fmt.Println("pushed", filepath.Base(filePath), "→", host+":"+remoteDir)
	return nil
}


// InstallSecondaryLocalBackupTimer daily local backup on secondary (no primary API).
func InstallSecondaryLocalBackupTimer() error {
	bin := "/usr/local/bin/netductor"
	service := fmt.Sprintf(`[Unit]
Description=Netductor secondary local backup
After=network-online.target

[Service]
Type=oneshot
ExecStart=%s backup secondary-local
Nice=10
`, bin)
	timer := `[Unit]
Description=Netductor secondary local backup daily

[Timer]
OnCalendar=*-*-* 03:30:00
Persistent=true

[Install]
WantedBy=timers.target
`
	_ = os.WriteFile("/etc/systemd/system/netductor-secondary-local-backup.service", []byte(service), 0o644)
	_ = os.WriteFile("/etc/systemd/system/netductor-secondary-local-backup.timer", []byte(timer), 0o644)
	_ = exec.Command("systemctl", "daemon-reload").Run()
	return exec.Command("systemctl", "enable", "--now", "netductor-secondary-local-backup.timer").Run()
}
