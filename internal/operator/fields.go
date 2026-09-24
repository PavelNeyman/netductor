package operator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

// FieldGetter abstracts TUI fieldVal / maps.
type FieldGetter func(key string) string

func yesish(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "y" || s == "yes" || s == "1" || s == "true" || s == "да"
}

func orDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func expandHome(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

// PrimaryFromFields maps wizard/CLI-style keys to PrimarySpec.
// Keys: host, user, password, gen_key, key_path, key_pass, sni, domain_base, le_email,
// cf_proxy, with_lampac, with_git, tg_token, tg_admin
func PrimaryFromFields(get FieldGetter) PrimarySpec {
	s := PrimarySpec{
		Host:            strings.TrimSpace(get("host")),
		User:            orDefault(get("user"), "root"),
		Password:        get("password"),
		GenerateKey:     yesish(orDefault(get("gen_key"), "yes")),
		SSHPrivateKey:   expandHome(orDefault(get("key_path"), "~/.ssh/netductor_primary")),
		KeyPassphrase:   get("key_pass"),
		SNI:             orDefault(get("sni"), "api.vk.me"),
		DomainBase:      strings.TrimSpace(get("domain_base")),
		DomainEmail:     strings.TrimSpace(get("le_email")),
		DomainCFProxy:   yesish(get("cf_proxy")),
		WithLampac:      yesish(get("with_lampac")),
		WithGitRegistry: yesish(get("with_git")),
		TelegramToken:   strings.TrimSpace(get("tg_token")),
		TelegramAdminID: strings.TrimSpace(get("tg_admin")),
		Version: deploy.Release,
	}
	ApplyDomainFlags(&s)
	return s
}

// SecondaryFromFields maps keys: host/sec_host, user, password/sec_password, sni, key_pass
// primary_* filled by caller or FleetDeploy.
func SecondaryFromFields(get FieldGetter) SecondarySpec {
	host := strings.TrimSpace(get("sec_host"))
	if host == "" {
		host = strings.TrimSpace(get("host"))
	}
	pass := get("sec_password")
	if pass == "" {
		pass = get("password")
	}
	return SecondarySpec{
		SecondaryHost:        host,
		SecondaryUser:        orDefault(get("user"), "root"),
		SecondaryPass:        pass,
		SNI:                  orDefault(get("sni"), "api.vk.me"),
		PrimaryKeyPassphrase: get("key_pass"),
	}
}

// FleetFromFields maps Fleet wizard fields.
func FleetFromFields(get FieldGetter) FleetSpec {
	p := PrimaryFromFields(get)
	sec := SecondaryFromFields(get)
	sec.PrimaryKeyPassphrase = p.KeyPassphrase
	if sec.SNI == "" {
		sec.SNI = p.SNI
	}
	return FleetSpec{
		DoPrimary:   yesish(orDefault(get("do_primary"), "yes")),
		DoSecondary: yesish(orDefault(get("do_secondary"), "yes")),
		Primary:     p,
		Secondary:   sec,
	}
}
