package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PrimaryOpts — bootstrap a clean Debian VPS from operator machine (Mac/PC).
type PrimaryOpts struct {
	Host            string
	User            string
	Password        string // first-login only
	SSHPrivateKey   string // path; empty + GenerateKey => ~/.ssh/netductor_primary
	GenerateKey     bool
	KeyPassphrase  string // optional; empty = no passphrase on generated/used key
	Version         string // e.g. 0.8.1
	TelegramToken   string
	TelegramAdminID string
	SNI             string
	SkipInstall     bool
	WithLampac      bool
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
	if o.SNI == "" {
		o.SNI = "api.vk.me"
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
	if !fileExists(pubPath) {
		return fmt.Errorf("public key missing: %s", pubPath)
	}
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

	fmt.Fprintln(os.Stderr, "==> download netductor v"+o.Version)
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

	if !o.SkipInstall {
		fmt.Fprintln(os.Stderr, "==> netductor install (may take several minutes)")
		out, err = runSSH("", keyPath, o.User, o.Host, "netductor install", o.KeyPassphrase)
		fmt.Print(out)
		if err != nil {
			return fmt.Errorf("install: %w", err)
		}
		// Secrets may exist before install; re-run telegram so unit starts with netductor-tg binary
		if o.TelegramToken != "" {
			fmt.Fprintln(os.Stderr, "==> ensure telegram bot unit")
			out, err = runSSH("", keyPath, o.User, o.Host, "netductor install telegram; systemctl restart netductor-telegram-bot || true; systemctl is-active netductor-telegram-bot || true", o.KeyPassphrase)
			fmt.Print(out)
			if err != nil {
				fmt.Fprintln(os.Stderr, "warn telegram ensure:", err)
			}
		}
		if os.Getenv("NETDUCTOR_SSH_PORT") == "" {
			_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
			fmt.Fprintln(os.Stderr, "==> post-harden SSH port 52222")
		}
	}

	fmt.Fprintln(os.Stderr, "==> set SNI", o.SNI)
	out, err = runSSH("", keyPath, o.User, o.Host, "netductor vpn set-sni "+shellQuote(o.SNI), o.KeyPassphrase)
	fmt.Print(out)
	_ = err

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
	fmt.Fprintln(os.Stderr, "==> registry + git bootstrap")
	out, _ = runSSH("", keyPath, o.User, o.Host,
		"command -v git >/dev/null || apt-get install -y -qq git; "+
			"netductor registry ensure; netductor registry crane; "+
			"netductor git init netductor 2>/dev/null || true; netductor git pipelines; netductor registry status",
		o.KeyPassphrase)
	fmt.Print(out)
	fmt.Fprintln(os.Stderr, "==> primary deploy done")
	port := sshPort()
	fmt.Fprintf(os.Stderr, "  SSH: ssh -i %s -p %s %s@%s\n", keyPath, port, o.User, o.Host)
	fmt.Fprintln(os.Stderr, "  Save key path in TUI settings (remote_key); NETDUCTOR_SSH_PORT="+port)
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// EnsureAgentBinary downloads netductor-agent for arch into destDir, returns path.
func EnsureAgentBinary(version, goarch, destDir string) (string, error) {
	if version == "" {
		version = Release
	}
	if goarch == "" {
		goarch = "arm64"
	}
	_ = os.MkdirAll(destDir, 0o755)
	name := "netductor-agent-linux-" + goarch
	dest := filepath.Join(destDir, name)
	url := fmt.Sprintf("https://github.com/PavelNeyman/netductor/releases/download/v%s/%s", version, name)
	fmt.Fprintln(os.Stderr, "==> fetch", url)
	cmd := exec.Command("curl", "-fsSL", "-o", dest, url)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("download agent %s: %w (check Releases assets)", name, err)
	}
	_ = os.Chmod(dest, 0o755)
	return dest, nil
}
