package install

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/domain"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
)

// ReissueLEAfterRecover runs certbot when DOMAIN + LE_EMAIL are in conf (certs not in backup).
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
