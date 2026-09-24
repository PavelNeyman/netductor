package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CollectOperatorSecrets SSHs to host, reads netductor secrets, writes a local file
// the operator must store offline. Returns path to the file.
// Does not write a file if SSH fails or no secrets were retrieved.
func CollectOperatorSecrets(role, user, host, keyPath, keyPass string) (string, error) {
	if user == "" {
		user = "root"
	}
	if host == "" {
		return "", fmt.Errorf("host required")
	}
	if keyPath == "" {
		return "", fmt.Errorf("SSH key path required")
	}
	script := `set +e
echo "===HOST==="
hostname -f 2>/dev/null || hostname
echo "===IP==="
hostname -I 2>/dev/null | awk '{print $1}'
echo "===SSH_PORT==="
grep -E '^Port ' /etc/ssh/sshd_config 2>/dev/null | awk '{print $2}' | tail -1
echo "===BACKUP_KEY==="
cat /etc/netductor/secrets/backup_key 2>/dev/null || true
echo "===RECOVERY_TOKEN==="
cat /etc/netductor/secrets/recovery_token 2>/dev/null || true
echo "===VERSION==="
cat /etc/netductor/VERSION 2>/dev/null || netductor version 2>/dev/null | head -1 || true
echo "===END==="
`
	out, err := runSSH("", keyPath, user, host, script, keyPass)
	raw := string(out)
	if err != nil {
		return "", fmt.Errorf("ssh collect secrets: %w\n%s", err, trimOut(raw))
	}
	if !strings.Contains(raw, "===END===") {
		return "", fmt.Errorf("ssh collect incomplete output (no END marker)")
	}

	get := func(section string) string {
		a := strings.Index(raw, "==="+section+"===")
		if a < 0 {
			return ""
		}
		rest := raw[a+len("==="+section+"==="):]
		if b := strings.Index(rest, "==="); b >= 0 {
			rest = rest[:b]
		}
		return strings.TrimSpace(rest)
	}
	hostname := get("HOST")
	ip := get("IP")
	sshPort := get("SSH_PORT")
	if sshPort == "" {
		sshPort = "52222"
	}
	backupKey := get("BACKUP_KEY")
	recTok := get("RECOVERY_TOKEN")
	ver := get("VERSION")

	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		role = "node"
	}
	// Primary must have backup key; secondary should have recovery token (backup key optional).
	switch role {
	case "primary", "core":
		if backupKey == "" {
			return "", fmt.Errorf("BACKUP_KEY empty on primary — install/backup init may have failed")
		}
	case "secondary":
		if recTok == "" && backupKey == "" {
			return "", fmt.Errorf("no RECOVERY_TOKEN or BACKUP_KEY on secondary")
		}
	default:
		if backupKey == "" && recTok == "" {
			return "", fmt.Errorf("no secrets found on %s", host)
		}
	}

	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		dir = "."
	}
	outDir := filepath.Join(dir, ".netductor", "credentials")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return "", err
	}
	ts := time.Now().UTC().Format("20060102-150405")
	safeHost := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			return r
		}
		return '-'
	}, host)
	path := filepath.Join(outDir, fmt.Sprintf("%s-%s-%s.txt", role, safeHost, ts))

	body := fmt.Sprintf(`# netductor operator credentials — KEEP OFFLINE / password manager
# Собран автоматически. Не коммитить в git. Не слать в чаты.
# WARNING: do not put ~/.netductor/ in iCloud/Dropbox/Google Drive sync; exclude from Time Machine if possible.
# Generated: %s UTC
# Role: %s

## Access / Доступ
Host:     %s
IP:       %s
SSH:      ssh -i %s -p %s %s@%s
Key file: %s  (private key stays on THIS machine only)

## Backup decryption key / Ключ расшифровки бэкапа
# netductor recover --from-secondary … --key …
# Server path: /etc/netductor/secrets/backup_key
BACKUP_KEY=%s

## Recovery token (secondary DR after "recovery arm")
# --recovery-token …  |  Server: /etc/netductor/secrets/recovery_token
RECOVERY_TOKEN=%s

## Notes
Version: %s
Recovery HTTP :8790 is OFF until: netductor recovery arm --ttl 30m
Then: netductor recover --from-secondary http://SECONDARY:8790 --recovery-token … --key …
Then: netductor recovery disarm
Password SSH is disabled after harden — only this key.

## RU
1. Приватный SSH-ключ — только на Mac.
2. BACKUP_KEY — сохранить offline (без него бэкап не расшифровать).
3. RECOVERY_TOKEN — для скачивания с secondary после arm.
4. Не синхронизировать ~/.netductor в облако.
`, time.Now().UTC().Format(time.RFC3339), role,
		hostname, ip, keyPath, sshPort, user, host, keyPath,
		backupKey, recTok, ver)

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", err
	}
	// latest symlink for role
	latest := filepath.Join(outDir, "latest-"+role+".txt")
	_ = os.Remove(latest)
	_ = os.Symlink(path, latest)
	pruneCredentials(outDir, role, 5)

	fmt.Fprintln(os.Stderr, "==> operator credentials saved:", path)
	fmt.Fprintln(os.Stderr, "==> latest symlink:", latest)
	fmt.Fprintln(os.Stderr, "    Store offline (password manager). Do not sync ~/.netductor to iCloud.")
	return path, nil
}

func trimOut(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}

// pruneCredentials keeps the newest keep files matching role-*.txt (not symlinks).
func pruneCredentials(dir, role string, keep int) {
	if keep < 1 {
		keep = 5
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	prefix := role + "-"
	var names []string
	for _, e := range ents {
		n := e.Name()
		if e.Type()&os.ModeSymlink != 0 {
			continue
		}
		if strings.HasPrefix(n, prefix) && strings.HasSuffix(n, ".txt") {
			names = append(names, n)
		}
	}
	sort.Strings(names) // timestamp in name sorts chronologically
	if len(names) <= keep {
		return
	}
	for _, n := range names[:len(names)-keep] {
		_ = os.Remove(filepath.Join(dir, n))
	}
}
