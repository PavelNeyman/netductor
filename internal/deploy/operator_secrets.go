package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CollectOperatorSecrets SSHs to host, reads netductor secrets, writes a local file
// the operator must store offline. Returns path to the file.
func CollectOperatorSecrets(role, user, host, keyPath, keyPass string) (string, error) {
	if user == "" {
		user = "root"
	}
	script := `set -e
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
echo "===SNI==="
grep -E 'sni|reality' /etc/netductor/*.json 2>/dev/null | head -3 || true
test -f /etc/netductor/netductor.conf && grep -i sni /etc/netductor/netductor.conf 2>/dev/null || true
echo "===VERSION==="
cat /etc/netductor/VERSION 2>/dev/null || netductor version 2>/dev/null | head -1 || true
echo "===END==="
`
	out, err := runSSH("", keyPath, user, host, script, keyPass)
	raw := string(out)
	// parse lightly
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

	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		dir = "."
	}
	outDir := filepath.Join(dir, ".netductor", "credentials")
	_ = os.MkdirAll(outDir, 0o700)
	ts := time.Now().UTC().Format("20060102-150405")
	safeHost := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			return r
		}
		return '-'
	}, host)
	path := filepath.Join(outDir, fmt.Sprintf("%s-%s-%s.txt", role, safeHost, ts))

	body := fmt.Sprintf(`# netductor operator credentials — KEEP OFFLINE / password manager
# Собран автоматически после деплоя. Не коммитить в git. Не слать в чаты.
# Generated: %s UTC
# Role: %s

## Access / Доступ
Host:     %s
IP:       %s
SSH:      ssh -i %s -p %s %s@%s
Key file: %s  (private key stays on THIS machine only)

## Backup decryption key / Ключ расшифровки бэкапа
# Нужен для: netductor recover --from-secondary … --key …
# На сервере: /etc/netductor/secrets/backup_key
BACKUP_KEY=%s

## Recovery token (secondary DR HTTP, after "recovery arm")
# Нужен для: --recovery-token при recover с secondary
# На secondary: /etc/netductor/secrets/recovery_token
# Primary may be empty.
RECOVERY_TOKEN=%s

## Notes / Заметки
Version: %s
Recovery HTTP :8790 is OFF by default.
  On secondary:  netductor recovery arm --ttl 30m
  Then recover:  netductor recover --from-secondary http://SECONDARY:8790 --recovery-token … --key …
  Then:          netductor recovery disarm

Password SSH is disabled after harden — only this key.
Save this file in a password manager, then you may delete the local copy.

## RU кратко
1. Приватный SSH-ключ — только на вашем Mac.
2. BACKUP_KEY — обязательно сохранить offline (без него бэкап с secondary не расшифровать).
3. RECOVERY_TOKEN — для скачивания бэкапа после recovery arm на secondary.
4. Пароль root после harden не работает.
`, time.Now().UTC().Format(time.RFC3339), role,
		hostname, ip, keyPath, sshPort, user, host, keyPath,
		backupKey, recTok, ver)

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", err
	}
	fmt.Fprintln(os.Stderr, "==> operator credentials saved:", path)
	fmt.Fprintln(os.Stderr, "    Store offline (password manager). Contains BACKUP_KEY / RECOVERY_TOKEN.")
	return path, err
}
