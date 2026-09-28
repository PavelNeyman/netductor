package install

import (
	"github.com/PavelNeyman/netductor/internal/ndconfig"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

//go:embed profiles/nd-oc.conf
var embeddedNdOcConf string

// EnsureClientProfiles writes SR Config (nd-oc.conf) used by TG workcfg document send.
func EnsureClientProfiles() error {
	dirs := []string{
		filepath.Join(paths.OptDir(), "profiles"),
		filepath.Join(paths.EtcDir(), "profiles"),
	}
	for _, d := range dirs {
		_ = os.MkdirAll(d, 0o755)
		p := filepath.Join(d, "nd-oc.conf")
		if st, err := os.Stat(p); err == nil && st.Size() > 100 {
			continue
		}
		if err := os.WriteFile(p, []byte(embeddedNdOcConf), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "profiles: wrote %s\n", p)
	}
	return nil
}

// EnsureDomainConfig writes netductor.conf keys for redirect/VPN host from env.
// Env: NETDUCTOR_DOMAIN or NETDUCTOR_PUBLIC_HOSTNAME (e.g. netductor.work.gd)
//      NETDUCTOR_REDIRECT_BASE (full URL) overrides derived http://domain
func EnsureDomainConfig() error {
	domain := strings.TrimSpace(os.Getenv("NETDUCTOR_DOMAIN"))
	if domain == "" {
		domain = strings.TrimSpace(os.Getenv("NETDUCTOR_PUBLIC_HOSTNAME"))
	}
	base := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_BASE"))
	if base == "" && domain != "" {
		if !strings.HasPrefix(domain, "http") {
			base = "http://" + domain
		} else {
			base = domain
			domain = strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
		}
	}
	conf := filepath.Join(paths.EtcDir(), "netductor.conf")
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	existing, _ := os.ReadFile(conf)
	body := string(existing)
	writeKey := func(key, val string) {
		if val == "" {
			return
		}
		line := key + "=" + val
		if strings.Contains(body, key+"=") {
			// keep existing unless force
			return
		}
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		body += line + "\n"
		_ = os.Setenv("NETDUCTOR_"+strings.TrimPrefix(key, "NETDUCTOR_"), val)
		if key == "REDIRECT_BASE" {
			_ = os.Setenv("NETDUCTOR_REDIRECT_BASE", val)
		}
	}
	if domain != "" {
		writeKey("PUBLIC_HOSTNAME", domain)
		writeKey("CORE_HOST", domain)
	}
	if base != "" {
		writeKey("REDIRECT_BASE", base)
	}
	if body != string(existing) && body != "" {
		if err := os.WriteFile(conf, []byte(body), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "domain: updated %s\n", conf)
	}
	if base == "" && domain == "" {
		fmt.Fprintln(os.Stderr, "domain: NETDUCTOR_DOMAIN / REDIRECT_BASE unset — TG import url-buttons need http(s) base")
	}
	return nil
}

// InstallRedirect unit for deep-link buttons (optional listen :80).
func InstallRedirect() error {
	bin, err := os.Executable()
	if err != nil || bin == "" {
		bin = "/usr/local/bin/netductor"
	}
	// Prefer LE HTTPS :8443 when certs present (after domain set --le). No public :80.
	// Env first, then /etc/netductor/netductor.conf (recover restores conf but unit may lag).
	cert := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_TLS_CERT"))
	key := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_TLS_KEY"))
	if cert == "" {
		cert = strings.TrimSpace(os.Getenv("NETDUCTOR_TLS_CERT"))
	}
	if key == "" {
		key = strings.TrimSpace(os.Getenv("NETDUCTOR_TLS_KEY"))
	}
	if conf := readNetductorConfMap(); conf != nil {
		if cert == "" {
			cert = conf["REDIRECT_TLS_CERT"]
			if cert == "" {
				cert = conf["NETDUCTOR_REDIRECT_TLS_CERT"]
			}
		}
		if key == "" {
			key = conf["REDIRECT_TLS_KEY"]
			if key == "" {
				key = conf["NETDUCTOR_REDIRECT_TLS_KEY"]
			}
		}
		if os.Getenv("NETDUCTOR_REDIRECT_BASE") == "" && conf["REDIRECT_BASE"] != "" {
			_ = os.Setenv("NETDUCTOR_REDIRECT_BASE", conf["REDIRECT_BASE"])
		}
	}
	var unit string
	if cert != "" && key != "" {
		unit = fmt.Sprintf(`[Unit]
Description=Netductor import redirect (TG deep links)
After=network-online.target

[Service]
Type=simple
EnvironmentFile=-/etc/netductor/netductor.conf
Environment=NETDUCTOR_REDIRECT_TLS_CERT=%s
Environment=NETDUCTOR_REDIRECT_TLS_KEY=%s
ExecStart=%s redirect-serve -listen off -https-listen :%s -tls-cert %s -tls-key %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, cert, key, bin, ndconfig.RedirectHTTPSPort(), cert, key)
	} else {
		// Product: no public HTTP :80 (only HTTPS :8443 after LE).
		listen := "off"
		unit = fmt.Sprintf(`[Unit]
Description=Netductor import redirect (TG deep links)
After=network-online.target

[Service]
Type=simple
EnvironmentFile=-/etc/netductor/netductor.conf
ExecStart=%s redirect-serve -listen %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, bin, listen)
	}
	if err := writeUnit("netductor-redirect.service", unit); err != nil {
		return err
	}
	base := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_BASE"))
	if base == "" {
		fmt.Fprintln(os.Stderr, "redirect: unit installed; set REDIRECT_BASE and start when domain ready")
		return nil
	}
	if err := enableStart("netductor-redirect"); err != nil {
		return err
	}
	EnsureRedirectRunning()
	return nil
}

// EnsureRedirectRunning starts netductor-redirect if the unit exists, is enabled
// (or has LE certs + REDIRECT_BASE), and is not active. Prevents collect/TG spam
// after binary updates that kill the shared netductor process.
func EnsureRedirectRunning() {
	unit := "/etc/systemd/system/netductor-redirect.service"
	if _, err := os.Stat(unit); err != nil {
		return
	}
	out, _ := exec.Command("systemctl", "is-active", "netductor-redirect").Output()
	if strings.TrimSpace(string(out)) == "active" {
		return
	}
	// enabled or has conf base
	en, _ := exec.Command("systemctl", "is-enabled", "netductor-redirect").Output()
	enS := strings.TrimSpace(string(en))
	base := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_BASE"))
	if base == "" {
		if conf := readNetductorConfMap(); conf != nil {
			base = conf["REDIRECT_BASE"]
		}
	}
	if enS != "enabled" && base == "" {
		return
	}
	_ = exec.Command("systemctl", "start", "netductor-redirect").Run()
}

func readNetductorConfMap() map[string]string {
	b, err := os.ReadFile("/etc/netductor/netductor.conf")
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}
