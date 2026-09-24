package operator

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

// FieldGetter abstracts TUI fieldVal / maps.
type FieldGetter func(key string) string

func yesish(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "y" || s == "yes" || s == "1" || s == "true" || s == "on" || s == "да"
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
		Version:         deploy.Release,
	}
	ApplyDomainFlags(&s)
	return s
}

// SecondaryFromFields maps keys: host/sec_host, user, password/sec_password, sni, key_pass
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

// ValidHost rejects empty, overlong, option-like, and non hostname/IP targets (SSH safety).
func ValidHost(h string) bool {
	h = strings.TrimSpace(h)
	if h == "" || len(h) > 253 {
		return false
	}
	if strings.HasPrefix(h, "-") {
		return false
	}
	if strings.ContainsAny(h, " \t\n\r;|&$`\"'\\<>(){}[]*?!#%") {
		return false
	}
	if ip := net.ParseIP(h); ip != nil {
		return true
	}
	if strings.HasPrefix(h, ".") || strings.HasSuffix(h, ".") || strings.Contains(h, "..") {
		return false
	}
	for _, part := range strings.Split(h, ".") {
		if part == "" || len(part) > 63 {
			return false
		}
		if strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
			return false
		}
		for _, r := range part {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
				return false
			}
		}
	}
	return true
}

// ValidUser allows only safe remote account names for user@host.
func ValidUser(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" || len(u) > 64 {
		return false
	}
	if strings.HasPrefix(u, "-") {
		return false
	}
	for _, r := range u {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}


// ValidDeviceID restricts edge device ids used in remote paths (no path traversal).
func ValidDeviceID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// EdgeFromFields maps OpenWrt wizard keys → EdgeSpec.
// Keys: router, password, id, arch, server, net, lan_ip, wifi_ssid, wifi_key, wan_proto, guest, key_pass
// Primary* filled by caller from TUI settings when empty.
func EdgeFromFields(get FieldGetter) EdgeSpec {
	return EdgeSpec{
		RouterHost:           strings.TrimSpace(get("router")),
		RouterUser:           orDefault(get("user"), "root"),
		RouterPass:           get("password"),
		DeviceID:             strings.TrimSpace(get("id")),
		AgentArch:            orDefault(get("arch"), "arm64"),
		ServerURL:            strings.TrimSpace(get("server")),
		NetConfigure:         yesish(get("net")),
		LANIP:                strings.TrimSpace(get("lan_ip")),
		WiFiSSID24:           get("wifi_ssid"),
		WiFiKey24:            get("wifi_key"),
		WANProto:             orDefault(get("wan_proto"), "dhcp"),
		GuestEnable:          yesish(get("guest")),
		PrimaryKeyPassphrase: get("key_pass"),
	}
}
