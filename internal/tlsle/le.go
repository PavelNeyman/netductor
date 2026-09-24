// Package tlsle obtains Let's Encrypt certificates via certbot (HTTP-01 standalone).
package tlsle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// Config for Obtain.
type Config struct {
	Email    string
	Domains  []string
	Base     string // if set and Domains empty → primary.<base>, i.<base>
	Staging  bool
	AgreeTOS bool
}

// Obtain installs certbot if needed, issues certs, links into secrets/tls, updates conf + redirect unit.
func Obtain(c Config) error {
	c.Email = strings.TrimSpace(c.Email)
	c.Base = strings.TrimSpace(strings.TrimPrefix(c.Base, "."))
	if c.Email == "" {
		return fmt.Errorf("--email required (LE registration)")
	}
	if len(c.Domains) == 0 && c.Base != "" {
		c.Domains = []string{"primary." + c.Base, "i." + c.Base}
	}
	if len(c.Domains) == 0 {
		// try public_hostname
		if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_hostname")); err == nil {
			if h := strings.TrimSpace(string(b)); h != "" {
				c.Domains = append(c.Domains, h)
			}
		}
	}
	if len(c.Domains) == 0 {
		return fmt.Errorf("no domains: pass --domains or --base")
	}
	if !c.AgreeTOS {
		return fmt.Errorf("--agree-tos required")
	}

	fmt.Fprintln(os.Stderr, "tls le: domains", strings.Join(c.Domains, ", "))

	if err := ensureCertbot(); err != nil {
		return err
	}
	_ = openFirewallHTTP()

	// Free :80 for standalone challenge
	_ = exec.Command("systemctl", "stop", "netductor-redirect").Run()

	args := []string{"certonly", "--standalone", "--non-interactive", "--agree-tos",
		"--email", c.Email, "--preferred-challenges", "http",
		"--keep-until-expiring", "--expand"}
	if c.Staging {
		args = append(args, "--staging")
	}
	for _, d := range c.Domains {
		args = append(args, "-d", d)
	}
	cmd := exec.Command("certbot", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = exec.Command("systemctl", "start", "netductor-redirect").Run()
		return fmt.Errorf("certbot: %w", err)
	}

	// Certbot uses first domain as live dir name
	liveName := c.Domains[0]
	live := filepath.Join("/etc/letsencrypt/live", liveName)
	fullchain := filepath.Join(live, "fullchain.pem")
	privkey := filepath.Join(live, "privkey.pem")
	if _, err := os.Stat(fullchain); err != nil {
		// fallback: search live dirs
		if alt := findLiveCert(c.Domains); alt != "" {
			fullchain = filepath.Join(alt, "fullchain.pem")
			privkey = filepath.Join(alt, "privkey.pem")
		}
	}
	if _, err := os.Stat(fullchain); err != nil {
		return fmt.Errorf("cert not found under /etc/letsencrypt/live")
	}

	dir := filepath.Join(paths.EtcDir(), "secrets", "tls")
	_ = os.MkdirAll(dir, 0o700)
	linkOrCopy(fullchain, filepath.Join(dir, "server.crt"))
	linkOrCopy(privkey, filepath.Join(dir, "server.key"))
	_ = os.Chmod(filepath.Join(dir, "server.key"), 0o600)

	// conf env for redirect
	upsert := map[string]string{
		"REDIRECT_TLS_CERT": fullchain,
		"REDIRECT_TLS_KEY":  privkey,
		"TLS_CERT":          fullchain,
		"TLS_KEY":           privkey,
	}
	// HTTPS redirect base if we have i. host
	for _, d := range c.Domains {
		if strings.HasPrefix(d, "i.") {
			upsert["REDIRECT_BASE"] = "https://" + d
			_ = os.Setenv("NETDUCTOR_REDIRECT_BASE", "https://"+d)
			break
		}
	}
	if err := writeConfKeys(upsert); err != nil {
		fmt.Fprintln(os.Stderr, "warn conf:", err)
	}
	ndconfig.Load()

	if err := rewriteRedirectUnitHTTPS(fullchain, privkey); err != nil {
		fmt.Fprintln(os.Stderr, "warn redirect unit:", err)
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "netductor-redirect").Run()
	if err := exec.Command("systemctl", "restart", "netductor-redirect").Run(); err != nil {
		fmt.Fprintln(os.Stderr, "warn restart redirect:", err)
	}

	fmt.Fprintln(os.Stderr, "tls le: OK")
	fmt.Fprintln(os.Stderr, "  cert:", fullchain)
	fmt.Fprintln(os.Stderr, "  key:", privkey)
	return nil
}

func ensureCertbot() error {
	if _, err := exec.LookPath("certbot"); err == nil {
		return nil
	}
	fmt.Fprintln(os.Stderr, "tls le: installing certbot…")
	_ = exec.Command("apt-get", "update", "-qq").Run()
	cmd := exec.Command("apt-get", "install", "-y", "-qq", "certbot")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func openFirewallHTTP() error {
	// best-effort ufw
	_ = exec.Command("ufw", "allow", "80/tcp").Run()
	_ = exec.Command("ufw", "allow", "443/tcp").Run()
	return nil
}

func findLiveCert(domains []string) string {
	entries, err := os.ReadDir("/etc/letsencrypt/live")
	if err != nil {
		return ""
	}
	want := map[string]bool{}
	for _, d := range domains {
		want[d] = true
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if want[e.Name()] {
			return filepath.Join("/etc/letsencrypt/live", e.Name())
		}
	}
	// any non-README dir
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			return filepath.Join("/etc/letsencrypt/live", e.Name())
		}
	}
	return ""
}

func linkOrCopy(src, dst string) {
	_ = os.Remove(dst)
	if err := os.Symlink(src, dst); err != nil {
		b, err2 := os.ReadFile(src)
		if err2 == nil {
			_ = os.WriteFile(dst, b, 0o644)
		}
	}
}

func writeConfKeys(kv map[string]string) error {
	path := ndconfig.ConfPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	existing, _ := os.ReadFile(path)
	lines := []string{}
	if len(existing) > 0 {
		lines = strings.Split(strings.ReplaceAll(string(existing), "\r\n", "\n"), "\n")
	}
	seen := map[string]bool{}
	var out []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			out = append(out, line)
			continue
		}
		t := strings.TrimPrefix(trim, "export ")
		i := strings.IndexByte(t, '=')
		if i <= 0 {
			out = append(out, line)
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(t[:i]))
		replaced := false
		for want, val := range kv {
			if val == "" {
				continue
			}
			if key == want || key == "NETDUCTOR_"+want {
				out = append(out, want+"="+val)
				seen[want] = true
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, line)
		}
	}
	for k, v := range kv {
		if v == "" || seen[k] {
			continue
		}
		out = append(out, k+"="+v)
	}
	body := strings.Join(out, "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func rewriteRedirectUnitHTTPS(cert, key string) error {
	bin, err := os.Executable()
	if err != nil || bin == "" {
		bin = "/usr/local/bin/netductor"
	}
	// HTTP :80 for ACME renew + legacy; HTTPS :443
	unit := fmt.Sprintf(`[Unit]
Description=Netductor import redirect (TG deep links)
After=network-online.target

[Service]
Type=simple
EnvironmentFile=-/etc/netductor/netductor.conf
Environment=NETDUCTOR_REDIRECT_TLS_CERT=%s
Environment=NETDUCTOR_REDIRECT_TLS_KEY=%s
ExecStart=%s redirect-serve -listen :80 -https-listen :443 -tls-cert %s -tls-key %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, cert, key, bin, cert, key)
	path := "/etc/systemd/system/netductor-redirect.service"
	return os.WriteFile(path, []byte(unit), 0o644)
}
