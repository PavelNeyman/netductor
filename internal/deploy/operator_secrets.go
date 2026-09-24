package deploy

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CollectOperatorSecrets SSHs to host, archives ALL /etc/netductor secrets + recovery
// material to ~/.netductor/credentials/ on the operator Mac.
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
	_ = os.Chmod(keyPath, 0o600)

	// Full dump: text sections + base64 tar of secrets tree and critical state
	script := `set +e
echo "===HOST==="
hostname -f 2>/dev/null || hostname
echo "===IP==="
hostname -I 2>/dev/null | awk '{print $1}'
echo "===SSH_PORT==="
(sshd -T 2>/dev/null | awk '/^port /{print $2; exit}'; grep -E '^Port ' /etc/ssh/sshd_config 2>/dev/null | awk '{print $2}' | tail -1)
echo "===VERSION==="
cat /etc/netductor/VERSION 2>/dev/null || netductor version 2>/dev/null | head -1 || true
echo "===CONF==="
cat /etc/netductor/netductor.conf 2>/dev/null || true
echo "===PUBLIC_HOSTNAME==="
cat /etc/netductor/public_hostname 2>/dev/null || true
echo "===VPN_HOSTNAME==="
cat /etc/netductor/vpn_hostname 2>/dev/null || true
echo "===SECRETS_LIST==="
find /etc/netductor/secrets -type f 2>/dev/null | sort
echo "===BACKUP_KEY==="
cat /etc/netductor/secrets/backup_key 2>/dev/null || true
echo "===RECOVERY_TOKEN==="
cat /etc/netductor/secrets/recovery_token 2>/dev/null || true
echo "===TAR_B64==="
# secrets + conf + hostnames + secondary registry tokens + LE live certs if any
TMP=$(mktemp)
tar czf "$TMP" \
  -C / etc/netductor/secrets \
  etc/netductor/netductor.conf \
  etc/netductor/public_hostname \
  etc/netductor/vpn_hostname \
  var/lib/netductor/secondary/devices.json \
  var/lib/netductor/secondary/bundle.json \
  var/lib/netductor/nodes/registry.json \
  etc/letsencrypt/live \
  etc/letsencrypt/archive \
  etc/letsencrypt/renewal \
  2>/dev/null
base64 -w0 "$TMP" 2>/dev/null || base64 "$TMP" 2>/dev/null | tr -d '\n'
rm -f "$TMP"
echo
echo "===END==="
`

	out, err := runSSHWithPortFallback(keyPath, user, host, script, keyPass)
	raw := out
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
		if p := os.Getenv("NETDUCTOR_SSH_PORT"); p != "" {
			sshPort = p
		} else {
			sshPort = "52222"
		}
	}
	backupKey := get("BACKUP_KEY")
	recTok := get("RECOVERY_TOKEN")
	ver := get("VERSION")
	secretsList := get("SECRETS_LIST")
	conf := get("CONF")
	tarB64 := get("TAR_B64")
	// strip accidental newlines in b64
	tarB64 = strings.ReplaceAll(tarB64, "\n", "")
	tarB64 = strings.ReplaceAll(tarB64, "\r", "")

	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		role = "node"
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

	bundleDir := filepath.Join(outDir, fmt.Sprintf("%s-%s-%s", role, safeHost, ts))
	if err := os.MkdirAll(bundleDir, 0o700); err != nil {
		return "", err
	}

	// Decode full archive if present
	archivePath := filepath.Join(bundleDir, "secrets-full.tgz")
	if tarB64 != "" {
		bin, err := base64.StdEncoding.DecodeString(tarB64)
		if err != nil {
			// try raw StdEncoding with whitespace already stripped
			bin, err = base64.RawStdEncoding.DecodeString(tarB64)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn: could not decode secrets tar:", err)
		} else if len(bin) > 0 {
			if err := os.WriteFile(archivePath, bin, 0o600); err != nil {
				return "", err
			}
			// extract for convenience
			extractDir := filepath.Join(bundleDir, "extract")
			_ = os.MkdirAll(extractDir, 0o700)
			// use tar command
			cmd := fmt.Sprintf("tar xzf %s -C %s", ShellQuote(archivePath), ShellQuote(extractDir))
			_ = runLocal(cmd)
		}
	}

	if conf != "" {
		_ = os.WriteFile(filepath.Join(bundleDir, "netductor.conf"), []byte(conf+"\n"), 0o600)
	}
	_ = os.WriteFile(filepath.Join(bundleDir, "secrets-list.txt"), []byte(secretsList+"\n"), 0o600)

	path := filepath.Join(bundleDir, "README.txt")
	body := fmt.Sprintf(`# netductor operator credentials — KEEP OFFLINE
# Generated: %s UTC
# Role: %s

## Access
Host:     %s
IP:       %s
SSH:      ssh -i %s -p %s %s@%s
Key file: %s

## Quick recovery keys
BACKUP_KEY=%s
RECOVERY_TOKEN=%s

## Full dump
- secrets-full.tgz — /etc/netductor/secrets + conf + hostnames + secondary devices.json + LE live (if present)
- extract/ — unpacked copy of the archive
- netductor.conf — domain/REDIRECT_BASE snapshot
- secrets-list.txt — files that existed under /etc/netductor/secrets

## Secrets on server (names)
%s

## Notes
Version: %s
After harden only key auth works.
Restore example (on a new VPS after install):
  tar xzf secrets-full.tgz -C /
  # then fix perms: chmod -R go-rwx /etc/netductor/secrets
`, time.Now().UTC().Format(time.RFC3339), role,
		hostname, ip, keyPath, sshPort, user, host, keyPath,
		backupKey, recTok, secretsList, ver)

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", err
	}

	// also flat latest-role.txt summary for password managers
	flat := filepath.Join(outDir, fmt.Sprintf("%s-%s-%s.txt", role, safeHost, ts))
	_ = os.WriteFile(flat, []byte(body), 0o600)
	latest := filepath.Join(outDir, "latest-"+role+".txt")
	_ = os.Remove(latest)
	_ = os.Symlink(flat, latest)
	latestDir := filepath.Join(outDir, "latest-"+role+".dir")
	_ = os.Remove(latestDir)
	_ = os.Symlink(bundleDir, latestDir)
	pruneCredentials(outDir, role, 5)

	fmt.Fprintln(os.Stderr, "==> operator credentials saved:", bundleDir)
	fmt.Fprintln(os.Stderr, "==> summary:", flat)
	fmt.Fprintln(os.Stderr, "    Store offline. Do not sync ~/.netductor to iCloud.")
	return bundleDir, nil
}

func runSSHWithPortFallback(keyPath, user, host, script, keyPass string) (string, error) {
	ports := []string{}
	if p := os.Getenv("NETDUCTOR_SSH_PORT"); p != "" {
		ports = append(ports, p)
	}
	for _, p := range []string{"52222", "22"} {
		dup := false
		for _, x := range ports {
			if x == p {
				dup = true
				break
			}
		}
		if !dup {
			ports = append(ports, p)
		}
	}
	var lastOut string
	var lastErr error
	prev := os.Getenv("NETDUCTOR_SSH_PORT")
	for _, p := range ports {
		_ = os.Setenv("NETDUCTOR_SSH_PORT", p)
		out, err := runSSH("", keyPath, user, host, script, keyPass)
		lastOut, lastErr = out, err
		if err == nil {
			if prev != "" {
				_ = os.Setenv("NETDUCTOR_SSH_PORT", prev)
			}
			return out, nil
		}
	}
	if prev != "" {
		_ = os.Setenv("NETDUCTOR_SSH_PORT", prev)
	} else {
		_ = os.Unsetenv("NETDUCTOR_SSH_PORT")
	}
	return lastOut, lastErr
}

func runLocal(cmd string) error {
	c := exec.Command("sh", "-c", cmd)
	return c.Run()
}

func trimOut(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}

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
	sort.Strings(names)
	if len(names) <= keep {
		return
	}
	for _, n := range names[:len(names)-keep] {
		_ = os.Remove(filepath.Join(dir, n))
		// also remove matching dir without .txt
		base := strings.TrimSuffix(n, ".txt")
		_ = os.RemoveAll(filepath.Join(dir, base))
	}
}
