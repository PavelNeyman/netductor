package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/download"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

// PrimaryOpts — bootstrap a clean Debian VPS from operator machine (Mac/PC).
type PrimaryOpts struct {
	Host              string
	User              string
	Password          string // first-login only
	SSHPrivateKey     string // path; empty + GenerateKey => ~/.ssh/netductor_primary
	GenerateKey       bool
	KeyPassphrase     string // optional; empty = no passphrase on generated/used key
	Version           string // e.g. 0.8.1
	TelegramToken     string
	TelegramAdminID   string
	SNI               string
	DomainBase        string // e.g. netductor.neyman.top → domain set on host
	DomainHTTP        bool   // http redirect base (ignored if DomainLE)
	DomainLE          bool   // Let's Encrypt after domain set
	DomainCFProxy     bool   // CF orange on i. → REDIRECT_BASE without :8443
	DomainEmail       string
	DomainPrimary     string // explicit e.g. p2.nd.example.com (overrides base primary label)
	DomainVPN         string // explicit VPN host e.g. s.nd.example.com
	DomainRedirect    string // full REDIRECT_BASE e.g. https://i2.nd.example.com:8443
	SSHPort           int    // 0 → default 52222
	RedirectHTTPSPort string
	AgentMTLSPort     string
	LampacPort        string
	SkipInstall       bool
	WithLampac        bool
	WithGitRegistry   bool // optional thin git + local registry (addon)
}

// DeployPrimary installs netductor on a remote VPS over SSH.
func DeployPrimary(o PrimaryOpts) error {
	if strings.TrimSpace(o.Host) == "" {
		return fmt.Errorf("host required")
	}
	if o.User == "" {
		o.User = "root"
	}
	if o.Version == "" {
		o.Version = Release
	}
	if !validReleaseVersion(o.Version) {
		return fmt.Errorf("invalid release version %q", o.Version)
	}
	if o.SNI == "" {
		o.SNI = ndconfig.DefaultSNI()
	}

	keyPath := strings.TrimSpace(o.SSHPrivateKey)
	if keyPath == "" {
		var err error
		keyPath, err = defaultKeyPath()
		if err != nil {
			return err
		}
	}
	pubPath := keyPath + ".pub"

	if o.GenerateKey || !fileExists(keyPath) {
		if !(fileExists(keyPath) && !o.GenerateKey) {
			fmt.Fprintln(os.Stderr, "==> generating SSH key", keyPath)
			_ = os.Remove(keyPath)
			_ = os.Remove(pubPath)
			pass := o.KeyPassphrase
			cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", pass, "-C", "netductor-primary")
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("ssh-keygen: %w", err)
			}
		}
	}
	if !fileExists(pubPath) || fileEmpty(pubPath) {
		cmd := exec.Command("ssh-keygen", "-y", "-f", keyPath)
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("public key missing: %s", pubPath)
		}
		_ = os.WriteFile(pubPath, out, 0o644)
	}
	_ = os.Chmod(keyPath, 0o600)
	pub, err := os.ReadFile(pubPath)
	if err != nil {
		return err
	}
	pubLine := strings.TrimSpace(string(pub))

	if o.Password == "" && !fileExists(keyPath) {
		return fmt.Errorf("password required for first login (or provide existing SSH key)")
	}

	fmt.Fprintln(os.Stderr, "==> install SSH public key on", o.Host)
	installKey := fmt.Sprintf(`set -e
mkdir -p /root/.ssh
chmod 700 /root/.ssh
touch /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
grep -qxF '%s' /root/.ssh/authorized_keys || echo '%s' >> /root/.ssh/authorized_keys
`, pubLine, pubLine)
	pass := o.Password
	out, err := runSSH(pass, "", o.User, o.Host, installKey, o.KeyPassphrase)
	if err != nil {
		out2, err2 := runSSH("", keyPath, o.User, o.Host, installKey, o.KeyPassphrase)
		if err2 != nil {
			return fmt.Errorf("ssh install key: %v\n%s\nfallback: %v\n%s", err, out, err2, out2)
		}
	}

	fmt.Fprintln(os.Stderr, "==> download node binary netductor-linux-* v"+o.Version+" (not netductor-op)")
	dl := fmt.Sprintf(`set -e
arch=$(uname -m)
case "$arch" in x86_64) a=amd64;; aarch64) a=arm64;; *) echo "unsupported arch $arch"; exit 1;; esac
URL="https://github.com/PavelNeyman/netductor/releases/download/v%s/netductor-linux-${a}"
if command -v curl >/dev/null 2>&1; then
  curl -fsSL -o /usr/local/bin/netductor "$URL"
elif command -v wget >/dev/null 2>&1; then
  wget -q -O /usr/local/bin/netductor "$URL"
else
  echo "need curl or wget"; exit 1
fi
chmod 755 /usr/local/bin/netductor
/usr/local/bin/netductor version
`, o.Version)
	out, err = runSSH("", keyPath, o.User, o.Host, dl, o.KeyPassphrase)
	fmt.Print(out)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}

	if o.TelegramToken != "" || o.TelegramAdminID != "" {
		fmt.Fprintln(os.Stderr, "==> write telegram secrets")
		script := "set -e; mkdir -p /etc/netductor/secrets; chmod 700 /etc/netductor/secrets\n"
		if o.TelegramToken != "" {
			script += fmt.Sprintf("printf '%%s\\n' %q > /etc/netductor/secrets/telegram_bot_token\n", o.TelegramToken)
		}
		if o.TelegramAdminID != "" {
			script += fmt.Sprintf("printf '%%s\\n' %q > /etc/netductor/secrets/telegram_admin_id\n", o.TelegramAdminID)
		}
		script += "chmod 600 /etc/netductor/secrets/* 2>/dev/null || true\n"
		out, err = runSSH("", keyPath, o.User, o.Host, script, o.KeyPassphrase)
		if err != nil {
			return fmt.Errorf("secrets: %w\n%s", err, out)
		}
	}

	// Product ports/SNI into conf BEFORE install so harden/redirect read them
	fmt.Fprintln(os.Stderr, "==> write product defaults to netductor.conf")
	if err := writeProductConfRemote(keyPath, o.User, o.Host, o.KeyPassphrase, o); err != nil {
		fmt.Fprintln(os.Stderr, "warn product conf:", err)
	}

	if !o.SkipInstall {
		fmt.Fprintln(os.Stderr, "==> netductor install (may take several minutes)")
		out, err = runSSH("", keyPath, o.User, o.Host, "netductor install", o.KeyPassphrase)
		fmt.Print(out)
		if err != nil {
			// Telegram asset missing on a hand-cut release must not block LE/domain/lampac.
			msg := err.Error() + "\n" + out
			// Only soft-fail missing telegram release asset / unit download — not core failures.
			low := strings.ToLower(msg)
			if strings.Contains(low, "telegram") && (strings.Contains(low, "404") ||
				strings.Contains(low, "not found") || strings.Contains(low, "no such file") ||
				strings.Contains(low, "failed to download") || strings.Contains(low, "asset")) {
				fmt.Fprintln(os.Stderr, "warn install partial telegram (continuing deploy):", err)
			} else {
				return fmt.Errorf("install: %w", err)
			}
		}
		// Harden moves SSH off :22 — set operator-side port before any more remote cmds
		port := "52222"
		if o.SSHPort > 0 {
			port = fmt.Sprintf("%d", o.SSHPort)
		}
		_ = os.Setenv("NETDUCTOR_SSH_PORT", port)
		fmt.Fprintln(os.Stderr, "==> post-harden SSH port", port)

		// Secrets may exist before install; re-run telegram so unit starts with netductor-tg binary
		if o.TelegramToken != "" {
			fmt.Fprintln(os.Stderr, "==> ensure telegram bot unit")
			out, err = runSSH("", keyPath, o.User, o.Host, "netductor install telegram; systemctl restart netductor-telegram-bot || true; systemctl is-active netductor-telegram-bot || true", o.KeyPassphrase)
			fmt.Print(out)
			if err != nil {
				fmt.Fprintln(os.Stderr, "warn telegram ensure:", err)
			}
		}
	}

	fmt.Fprintln(os.Stderr, "==> set SNI", o.SNI)
	out, err = runSSH("", keyPath, o.User, o.Host, "netductor vpn set-sni "+shellQuote(o.SNI), o.KeyPassphrase)
	fmt.Print(out)
	_ = err

	if o.DomainLE && strings.TrimSpace(o.DomainPrimary) == "" {
		return fmt.Errorf("domain LE requires explicit --domain-primary (and usually --domain-redirect); no p./i. invent from --domain-base")
	}
	hasDomain := strings.TrimSpace(o.DomainBase) != "" || strings.TrimSpace(o.DomainPrimary) != "" || strings.TrimSpace(o.DomainRedirect) != ""
	if hasDomain {
		fmt.Fprintln(os.Stderr, "==> domain set")
		var cmd string
		if strings.TrimSpace(o.DomainPrimary) != "" || strings.TrimSpace(o.DomainRedirect) != "" {
			cmd = "netductor domain set"
			if p := strings.TrimSpace(o.DomainPrimary); p != "" {
				cmd += " --primary " + shellQuote(p)
			}
			if v := strings.TrimSpace(o.DomainVPN); v != "" {
				cmd += " --vpn " + shellQuote(v)
			} else if strings.TrimSpace(o.DomainBase) != "" {
				cmd += " --vpn " + shellQuote("s."+strings.TrimSpace(o.DomainBase))
			}
			if r := strings.TrimSpace(o.DomainRedirect); r != "" {
				cmd += " --redirect " + shellQuote(r)
			}
			if b := strings.TrimSpace(o.DomainBase); b != "" {
				cmd += " --base " + shellQuote(b) // DOMAIN= conf only; hosts already explicit
			}
		} else {
			cmd = "netductor domain set --base " + shellQuote(o.DomainBase)
		}
		if o.DomainLE && strings.TrimSpace(o.DomainEmail) != "" {
			cmd += " --le --email " + shellQuote(o.DomainEmail)
			fmt.Fprintln(os.Stderr, "==> Let's Encrypt", o.DomainPrimary, o.DomainRedirect)
		} else if o.DomainHTTP {
			cmd += " --http"
		}
		if o.DomainCFProxy {
			cmd += " --cf-proxy"
		}
		out, err = runSSH("", keyPath, o.User, o.Host, cmd, o.KeyPassphrase)
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn domain:", err)
		} else {
			// Bot often starts during install before REDIRECT_BASE exists — reload conf.
			fmt.Fprintln(os.Stderr, "==> restart telegram bot (pick up REDIRECT_BASE)")
			out2, err2 := runSSH("", keyPath, o.User, o.Host,
				"systemctl restart netductor-telegram-bot 2>/dev/null || true; systemctl is-active netductor-telegram-bot 2>/dev/null || true",
				o.KeyPassphrase)
			fmt.Print(out2)
			if err2 != nil {
				fmt.Fprintln(os.Stderr, "warn telegram restart:", err2)
			}
		}
	}

	fmt.Fprintln(os.Stderr, "==> fleet bootstrap + doctor")
	out, _ = runSSH("", keyPath, o.User, o.Host, "netductor fleet bootstrap; netductor doctor", o.KeyPassphrase)
	fmt.Print(out)

	if o.WithLampac {
		fmt.Fprintln(os.Stderr, "==> install lampac (docker)")
		out, err = runSSH("", keyPath, o.User, o.Host, "netductor install lampac", o.KeyPassphrase)
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn lampac:", err)
		}
	}
	if o.WithGitRegistry {
		fmt.Fprintln(os.Stderr, "==> registry + git bootstrap")
		out, _ = runSSH("", keyPath, o.User, o.Host,
			"command -v git >/dev/null || apt-get install -y -qq git; "+
				"netductor registry ensure; netductor registry crane; "+
				"netductor git init netductor 2>/dev/null || true; netductor git pipelines; netductor registry status",
			o.KeyPassphrase)
		fmt.Print(out)
	}
	fmt.Fprintln(os.Stderr, "==> primary deploy done")
	port := sshPort()
	fmt.Fprintf(os.Stderr, "  SSH: ssh -i %s -p %s %s@%s\n", keyPath, port, o.User, o.Host)
	fmt.Fprintln(os.Stderr, "  Save key path in TUI settings (remote_key); NETDUCTOR_SSH_PORT="+port)
	if path, err := CollectOperatorSecrets("primary", o.User, o.Host, keyPath, o.KeyPassphrase); err != nil {
		fmt.Fprintln(os.Stderr, "warn: could not collect credentials file:", err)
	} else {
		fmt.Fprintln(os.Stderr, "  Credentials file:", path)
	}
	return nil
}

// writeProductConfRemote upserts KEY=value lines safely (no shell metachar injection).
func writeProductConfRemote(keyPath, user, host, keyPass string, o PrimaryOpts) error {
	kv := map[string]string{}
	port := o.SSHPort
	if port <= 0 {
		port = 52222
	}
	kv["SSH_PORT"] = fmt.Sprintf("%d", port)
	if p := strings.TrimSpace(o.RedirectHTTPSPort); p != "" {
		kv["REDIRECT_HTTPS_PORT"] = p
	}
	if p := strings.TrimSpace(o.AgentMTLSPort); p != "" {
		kv["AGENT_MTLS_PORT"] = p
	}
	if p := strings.TrimSpace(o.LampacPort); p != "" {
		kv["LAMPAC_PORT"] = p
	}
	if sni := strings.TrimSpace(o.SNI); sni != "" {
		kv["DEFAULT_SNI"] = sni
	}
	var b strings.Builder
	b.WriteString("set -e\nmkdir -p /etc/netductor\ntouch /etc/netductor/netductor.conf\n")
	for k, v := range kv {
		if !safeConfKey(k) {
			continue
		}
		// Drop existing key lines, append new (printf %q-safe via Go %q → shell single-quoted)
		b.WriteString(fmt.Sprintf(`tmp=$(mktemp)
grep -v '^%s=' /etc/netductor/netductor.conf > "$tmp" || true
printf '%%s=%%s\n' %s %s >> "$tmp"
mv "$tmp" /etc/netductor/netductor.conf
`, k, shellQuote(k), shellQuote(v)))
	}
	out, err := runSSH("", keyPath, user, host, b.String(), keyPass)
	if out != "" {
		fmt.Print(out)
	}
	return err
}

func safeConfKey(k string) bool {
	if k == "" {
		return false
	}
	for _, c := range k {
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return false
	}
	return true
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// validReleaseVersion rejects shell/path injection in release tags used in download URLs.
func validReleaseVersion(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 32 {
		return false
	}
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// EnsureAgentBinary downloads netductor-agent for arch into destDir, returns path.
// Online: curl + SHA256SUMS from GitHub, then cache SHA256SUMS next to the binary.
// Offline: reuse cache only if local SHA256SUMS (or SHA256SUMS-<ver>) verifies the file.
func EnsureAgentBinary(version, goarch, destDir string) (string, error) {
	if version == "" {
		version = Release
	}
	if !validReleaseVersion(version) {
		return "", fmt.Errorf("invalid release version %q", version)
	}
	if goarch == "" {
		goarch = "arm64"
	}
	if !ValidAgentArch(goarch) || goarch == "auto" {
		return "", fmt.Errorf("invalid agent arch %q", goarch)
	}
	_ = os.MkdirAll(destDir, 0o755)
	name := "netductor-agent-linux-" + goarch
	dest := filepath.Join(destDir, name)
	ver := strings.TrimPrefix(version, "v")
	tag := "v" + ver
	skip := os.Getenv("NETDUCTOR_UPDATE_SKIP_VERIFY") == "1" && os.Getenv("NETDUCTOR_UPDATE_SKIP_VERIFY_CONFIRM") == "yes"

	// Offline / pre-fetched: reuse cache if non-empty and checksum matches local sums.
	if st, err := os.Stat(dest); err == nil && st.Size() > 1024 {
		if skip {
			fmt.Fprintln(os.Stderr, "==> using cached agent (verify skipped)", dest)
			_ = os.Chmod(dest, 0o755)
			return dest, nil
		}
		if err := verifyAgentAgainstLocalSums(destDir, ver, name, dest); err != nil {
			return "", fmt.Errorf("cached agent %s: %w (re-run offline-prep or remove cache)", name, err)
		}
		fmt.Fprintln(os.Stderr, "==> using cached agent", dest)
		_ = os.Chmod(dest, 0o755)
		return dest, nil
	}

	url := fmt.Sprintf("https://github.com/PavelNeyman/netductor/releases/download/%s/%s", tag, name)
	fmt.Fprintln(os.Stderr, "==> fetch", url)
	// R9: shared download.Get (https + size + optional SHA). Cache sums for offline A.
	want := ""
	if !skip {
		sums, _, err := fetchAndCacheSHA256SUMS(tag, destDir, ver)
		if err != nil {
			return "", fmt.Errorf("SHA256SUMS for agent: %w", err)
		}
		var ok bool
		want, ok = sums[name]
		if !ok || want == "" {
			return "", fmt.Errorf("no checksum for %s in SHA256SUMS", name)
		}
	}
	if err := download.Get(url, dest, download.Options{
		MaxBytes:       512 << 20,
		UserAgent:      "netductor-deploy",
		ExpectedSHA256: want,
		FileMode:       0o755,
	}); err != nil {
		_ = os.Remove(dest)
		return "", fmt.Errorf("download agent %s: %w (pre-place at %s for offline)", name, err, dest)
	}
	return dest, nil
}

// agentSumsPaths returns candidate local SHA256SUMS paths (versioned first, then generic).
func agentSumsPaths(destDir, ver string) []string {
	out := []string{}
	if ver != "" {
		out = append(out, filepath.Join(destDir, "SHA256SUMS-"+ver))
		out = append(out, filepath.Join(destDir, "SHA256SUMS-v"+ver))
	}
	out = append(out, filepath.Join(destDir, "SHA256SUMS"))
	return out
}

func loadLocalAgentSums(destDir, ver string) (map[string]string, string, error) {
	var lastErr error
	for _, p := range agentSumsPaths(destDir, ver) {
		b, err := os.ReadFile(p)
		if err != nil {
			lastErr = err
			continue
		}
		sums, err := ndupdate.ParseSHA256SUMS(b)
		if err != nil {
			lastErr = err
			continue
		}
		return sums, p, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no SHA256SUMS in %s", destDir)
	}
	return nil, "", lastErr
}

func verifyAgentAgainstLocalSums(destDir, ver, name, dest string) error {
	sums, path, err := loadLocalAgentSums(destDir, ver)
	if err != nil {
		return fmt.Errorf("missing local SHA256SUMS: %w", err)
	}
	want, ok := sums[name]
	if !ok || want == "" {
		return fmt.Errorf("no checksum for %s in %s", name, path)
	}
	got, err := fileSHA256Hex(dest)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch (got %s want %s from %s)", got, want, path)
	}
	return nil
}

// fetchAndCacheSHA256SUMS downloads release sums and writes SHA256SUMS + SHA256SUMS-<ver> under destDir.
func fetchAndCacheSHA256SUMS(tag, destDir, ver string) (map[string]string, []byte, error) {
	raw, sums, err := ndupdate.FetchSHA256SUMSRaw(tag)
	if err != nil {
		return nil, nil, err
	}
	_ = os.WriteFile(filepath.Join(destDir, "SHA256SUMS"), raw, 0o644)
	if ver != "" {
		_ = os.WriteFile(filepath.Join(destDir, "SHA256SUMS-"+ver), raw, 0o644)
	}
	return sums, raw, nil
}

func fileSHA256Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func fileEmpty(p string) bool {
	st, err := os.Stat(p)
	return err != nil || st.Size() == 0
}
