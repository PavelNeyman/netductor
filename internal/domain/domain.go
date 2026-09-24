// Package domain applies public hostnames and redirect base for VPN links / TG buttons.
package domain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// Config is the operator-facing domain mapping (DNS is external — Cloudflare etc.).
type Config struct {
	Base            string // e.g. netductor.neyman.top
	Primary         string
	VPN             string
	RedirectBase    string
	UseHTTPRedirect bool
	EnableRedirect  bool
	LE             bool   // obtain Let's Encrypt after apply
	LEEmail        string
	LEStaging      bool
	CFProxiedI     bool   // Cloudflare orange on i. → REDIRECT_BASE without :8443
}

// Expand fills empty Primary/VPN/RedirectBase from Base.
func (c *Config) Expand() {
	base := strings.TrimSpace(c.Base)
	base = strings.TrimPrefix(base, "https://")
	base = strings.TrimPrefix(base, "http://")
	base = strings.TrimSuffix(base, "/")
	c.Base = base
	if c.Primary == "" && base != "" {
		c.Primary = "primary." + base
	}
	if c.VPN == "" && base != "" {
		c.VPN = "vpn." + base
	}
	if c.RedirectBase == "" && base != "" {
		if c.UseHTTPRedirect && !c.LE {
			c.RedirectBase = "http://i." + base
		} else if c.CFProxiedI {
			// CF orange on i.: public URL on :443 via CF; origin LE stays :8443
			c.RedirectBase = "https://i." + base
		} else {
			// Direct to origin LE (443 is Reality)
			c.RedirectBase = "https://i." + base + ":8443"
		}
	}
	c.Primary = strings.TrimSpace(c.Primary)
	c.VPN = strings.TrimSpace(c.VPN)
	c.RedirectBase = strings.TrimSpace(c.RedirectBase)
}

// Apply writes conf keys, hostname files, env; optionally enables redirect unit.
func Apply(c Config) error {
	c.Expand()
	if c.CFProxiedI {
		_ = os.Setenv("NETDUCTOR_CF_PROXY_I", "1")
	}
	if c.Primary == "" && c.VPN == "" && c.RedirectBase == "" {
		return fmt.Errorf("nothing to set: pass --base netductor.example.com or explicit hosts")
	}
	_ = os.MkdirAll(paths.EtcDir(), 0o755)

	upsert := map[string]string{}
	if c.Base != "" {
		upsert["DOMAIN"] = c.Base
	}
	if c.Primary != "" {
		upsert["PUBLIC_HOSTNAME"] = c.Primary
		upsert["CORE_HOST"] = c.Primary
		_ = os.WriteFile(filepath.Join(paths.EtcDir(), "public_hostname"), []byte(c.Primary+"\n"), 0o644)
		_ = os.Setenv("NETDUCTOR_PUBLIC_HOSTNAME", c.Primary)
		_ = os.Setenv("NETDUCTOR_CORE_HOST", c.Primary)
	}
	if c.VPN != "" {
		upsert["VPN_HOST"] = c.VPN
		_ = os.WriteFile(filepath.Join(paths.EtcDir(), "vpn_hostname"), []byte(c.VPN+"\n"), 0o644)
		_ = os.Setenv("NETDUCTOR_VPN_HOST", c.VPN)
	}
	if c.RedirectBase != "" {
		upsert["REDIRECT_BASE"] = c.RedirectBase
		_ = os.Setenv("NETDUCTOR_REDIRECT_BASE", c.RedirectBase)
		if c.CFProxiedI {
			upsert["NETDUCTOR_CF_PROXY_I"] = "1"
		}
	}
	if err := upsertConf(upsert); err != nil {
		return err
	}
	ndconfig.Load()

	fmt.Fprintln(os.Stderr, "domain: applied")
	if c.Primary != "" {
		fmt.Fprintln(os.Stderr, "  CORE / primary:", c.Primary)
	}
	if c.VPN != "" {
		fmt.Fprintln(os.Stderr, "  VPN entry:", c.VPN)
	}
	if c.RedirectBase != "" {
		fmt.Fprintln(os.Stderr, "  REDIRECT_BASE:", c.RedirectBase)
	}
	if c.EnableRedirect && c.RedirectBase != "" && !c.LE {
		if err := tryEnableRedirect(); err != nil {
			fmt.Fprintln(os.Stderr, "  warn redirect:", err)
		}
	}
	if c.LE {
		if err := runLE(c); err != nil {
			fmt.Fprintln(os.Stderr, "  warn LE:", err)
			// still try enable redirect (maybe http)
			if c.EnableRedirect {
				_ = tryEnableRedirect()
			}
		}
	}
	return nil
}

func runLE(c Config) error {
	email := strings.TrimSpace(c.LEEmail)
	if email == "" {
		return fmt.Errorf("LE requires email (domain set --le --email …)")
	}
	// lazy import via exec to avoid cycle — call netductor tls le
	bin, err := os.Executable()
	if err != nil || bin == "" {
		bin = "netductor"
	}
	args := []string{"tls", "le", "--email", email, "--agree-tos"}
	if c.LEStaging {
		args = append(args, "--staging")
	}
	if c.Base != "" {
		args = append(args, "--base", c.Base)
	} else if c.Primary != "" {
		args = append(args, "--domains", c.Primary)
		if c.RedirectBase != "" {
			// extract host from URL (strip path and :port)
			rb := strings.TrimPrefix(strings.TrimPrefix(c.RedirectBase, "https://"), "http://")
			rb = strings.Split(rb, "/")[0]
			if i := strings.Index(rb, ":"); i >= 0 {
				rb = rb[:i]
			}
			if rb != "" && rb != c.Primary {
				args = []string{"tls", "le", "--email", email, "--agree-tos", "--domains", c.Primary + "," + rb}
				if c.LEStaging {
					args = append(args, "--staging")
				}
			}
		}
	}
	fmt.Fprintln(os.Stderr, "domain: obtaining Let's Encrypt…")
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Show prints current domain-related settings.
func Show() {
	ndconfig.Load()
	for _, k := range []string{
		"NETDUCTOR_DOMAIN", "NETDUCTOR_PUBLIC_HOSTNAME", "NETDUCTOR_CORE_HOST",
		"NETDUCTOR_VPN_HOST", "NETDUCTOR_REDIRECT_BASE",
	} {
		fmt.Printf("%s=%s\n", k, os.Getenv(k))
	}
	for _, f := range []string{"public_hostname", "vpn_hostname"} {
		b, err := os.ReadFile(filepath.Join(paths.EtcDir(), f))
		if err == nil {
			fmt.Printf("file:%s=%s\n", f, strings.TrimSpace(string(b)))
		}
	}
}

func upsertConf(kv map[string]string) error {
	path := ndconfig.ConfPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	existing, _ := os.ReadFile(path)
	var lines []string
	if len(existing) > 0 {
		lines = strings.Split(strings.ReplaceAll(string(existing), "\r\n", "\n"), "\n")
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(lines)+len(kv))
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

func tryEnableRedirect() error {
	unit := "/etc/systemd/system/netductor-redirect.service"
	if _, err := os.Stat(unit); err != nil {
		return fmt.Errorf("unit not installed yet")
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	return exec.Command("systemctl", "enable", "--now", "netductor-redirect").Run()
}
