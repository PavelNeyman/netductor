package vpn

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func preferredSecondarySNIFile() string {
	return filepath.Join(paths.EtcDir(), "secrets", "preferred_relay_sni")
}

// RememberSecondarySNI persists SNI used on last successful relay provision.
func RememberSecondarySNI(sni string) error {
	sni = strings.TrimSpace(sni)
	if sni == "" {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(preferredSecondarySNIFile()), 0o700)
	return os.WriteFile(preferredSecondarySNIFile(), []byte(sni+"\n"), 0o600)
}

// ResolveSecondarySNI picks SNI for provision:
// 1) explicit flag (if non-empty)
// 2) preferred_relay_sni secret from last provision
// 3) last known SNI from relay device list for same IP (caller may pass host)
// 4) DefaultRealitySNI (api.vk.me) — never hardcode ya.ru
func ResolveSecondarySNI(flag, host string) string {
	flag = strings.TrimSpace(flag)
	if flag != "" {
		return flag
	}
	if b, err := os.ReadFile(preferredSecondarySNIFile()); err == nil {
		if v := strings.TrimSpace(string(b)); v != "" {
			return v
		}
	}
	// soft: try singbox_reality_sni only if looks like intentional relay preference — no, that's core SNI
	return DefaultRealitySNI
}
