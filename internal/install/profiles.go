package install

import (
	_ "embed"
	"fmt"
	"os"
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
	cert := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_TLS_CERT"))
	key := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_TLS_KEY"))
	if cert == "" {
		cert = strings.TrimSpace(os.Getenv("NETDUCTOR_TLS_CERT"))
	}
	if key == "" {
		key = strings.TrimSpace(os.Getenv("NETDUCTOR_TLS_KEY"))
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
ExecStart=%s redirect-serve -listen off -https-listen :8443 -tls-cert %s -tls-key %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, cert, key, bin, cert, key)
	} else {
		listen := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_LISTEN"))
		if listen == "" {
			// Before LE: loopback only (no public HTTP). Domain LE enables :8443.
			listen = "127.0.0.1:80"
		}
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
	return enableStart("netductor-redirect")
}
