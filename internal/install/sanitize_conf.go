package install

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/domain"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// Forbidden conf/env keys — never keep in netductor.conf or units (freeze).
var forbiddenConfKeys = map[string]struct{}{
	"NETDUCTOR_PLAIN_AGENT":        {},
	"PLAIN_AGENT":                  {},
	"NETDUCTOR_API_PUBLIC":         {},
	"API_PUBLIC":                   {},
	"NETDUCTOR_API_ALLOW_PUBLIC":   {},
	"API_ALLOW_PUBLIC":             {},
	"NETDUCTOR_TRUST_PROXY":        {},
	"TRUST_PROXY":                  {},
	"NETDUCTOR_LEGACY_ADMIN_UI":    {},
	"LEGACY_ADMIN_UI":              {},
	"NETDUCTOR_RECOVERY_SERVE_KEY": {},
	"RECOVERY_SERVE_KEY":           {},
	"NETDUCTOR_RECOVERY_FETCH_KEY": {},
	"RECOVERY_FETCH_KEY":           {},
	"NETDUCTOR_EDGE_LEGACY_TOKEN":  {},
	"EDGE_LEGACY_TOKEN":            {},
	// public HTTP redirect listen — product is :8443 only
	"NETDUCTOR_REDIRECT_LISTEN": {},
	"REDIRECT_LISTEN":           {},
}

// SanitizeNetductorConf removes footgun keys from conf file and process env.
func SanitizeNetductorConf() (removed []string) {
	p := filepath.Join(paths.EtcDir(), "netductor.conf")
	b, err := os.ReadFile(p)
	if err != nil {
		// still clear process env
		for k := range forbiddenConfKeys {
			if strings.HasPrefix(k, "NETDUCTOR_") {
				_ = os.Unsetenv(k)
			}
		}
		return nil
	}
	var out []string
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			out = append(out, line)
			continue
		}
		trim = strings.TrimPrefix(trim, "export ")
		i := strings.IndexByte(trim, '=')
		if i <= 0 {
			out = append(out, line)
			continue
		}
		key := strings.TrimSpace(trim[:i])
		ku := strings.ToUpper(key)
		if _, bad := forbiddenConfKeys[ku]; bad {
			removed = append(removed, ku)
			continue
		}
		if _, bad := forbiddenConfKeys[key]; bad {
			removed = append(removed, key)
			continue
		}
		out = append(out, line)
	}
	body := strings.Join(out, "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	_ = os.WriteFile(p, []byte(body), 0o600)
	for k := range forbiddenConfKeys {
		if strings.HasPrefix(k, "NETDUCTOR_") {
			_ = os.Unsetenv(k)
		}
	}
	if len(removed) > 0 {
		fmt.Fprintf(os.Stderr, "sanitize conf: removed %v\n", removed)
	}
	ndconfig.Load()
	return removed
}

// ReissueLEAfterRecover runs certbot when DOMAIN + LE_EMAIL restored in conf (certs not in backup).
func ReissueLEAfterRecover() error {
	ndconfig.Load()
	m := readNetductorConfMap()
	base := strings.TrimSpace(os.Getenv("NETDUCTOR_DOMAIN"))
	if base == "" {
		base = strings.TrimSpace(m["DOMAIN"])
		if base == "" {
			base = strings.TrimSpace(m["NETDUCTOR_DOMAIN"])
		}
	}
	email := strings.TrimSpace(os.Getenv("NETDUCTOR_LE_EMAIL"))
	if email == "" {
		email = strings.TrimSpace(m["LE_EMAIL"])
		if email == "" {
			email = strings.TrimSpace(m["NETDUCTOR_LE_EMAIL"])
		}
	}
	if base == "" || email == "" {
		fmt.Fprintln(os.Stderr, "recover: LE skip (need DOMAIN + LE_EMAIL in conf — set once via domain set --le)")
		return nil
	}
	fmt.Fprintf(os.Stderr, "recover: re-issue Let's Encrypt for base=%s\n", base)
	cfg := domain.Config{
		Base:           base,
		LE:             true,
		LEEmail:        email,
		EnableRedirect: true,
	}
	cfg.Expand()
	if err := domain.Apply(cfg); err != nil {
		return err
	}
	return InstallRedirect()
}
