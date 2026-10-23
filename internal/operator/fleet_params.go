package operator

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// NormalizeFleetParams maps Web flat job params (and nested maps) into FleetSpec.
func NormalizeFleetParams(p map[string]any) (FleetSpec, error) {
	var f FleetSpec
	if p == nil {
		return f, fmt.Errorf("fleet: empty params")
	}
	if b, err := json.Marshal(p); err == nil {
		_ = json.Unmarshal(b, &f)
	}
	if _, ok := p["do_primary"]; ok {
		f.DoPrimary = boolParam(p, "do_primary", false)
	}
	if _, ok := p["do_secondary"]; ok {
		f.DoSecondary = boolParam(p, "do_secondary", false)
	}
	if _, ok := p["do_build"]; ok {
		f.DoBuild = boolParam(p, "do_build", false)
	}
	prim := PrimaryFromFlatParams(p)
	mergePrimary(&f.Primary, prim)
	if h := strParam(p, "secondary_host", "SecondaryHost"); h != "" {
		f.Secondary.SecondaryHost = h
	}
	if u := strParam(p, "secondary_user", "SecondaryUser"); u != "" {
		f.Secondary.SecondaryUser = u
	}
	if pw := strParam(p, "secondary_password", "SecondaryPass", "secondary_pass"); pw != "" {
		f.Secondary.SecondaryPass = pw
	}
	if sni := strParam(p, "sni", "SNI"); sni != "" {
		if f.Secondary.SNI == "" {
			f.Secondary.SNI = sni
		}
		if f.Primary.SNI == "" {
			f.Primary.SNI = sni
		}
	}
	if kp := strParam(p, "key_passphrase", "key_pass", "PrimaryKeyPassphrase"); kp != "" {
		f.Secondary.PrimaryKeyPassphrase = kp
		if f.Primary.KeyPassphrase == "" {
			f.Primary.KeyPassphrase = kp
		}
	}
	if h := strParam(p, "primary_host"); h != "" {
		if f.Primary.Host == "" {
			f.Primary.Host = h
		}
		if f.Secondary.PrimaryHost == "" {
			f.Secondary.PrimaryHost = h
		}
	}
	if k := strParam(p, "primary_key", "key"); k != "" {
		k = expandHome(k)
		if f.Primary.SSHPrivateKey == "" {
			f.Primary.SSHPrivateKey = k
			f.Primary.GenerateKey = false
		}
		if f.Secondary.PrimaryKey == "" {
			f.Secondary.PrimaryKey = k
		}
	}
	if h := strParam(p, "build_host", "BuildHost"); h != "" {
		f.Build.Host = h
	}
	if u := strParam(p, "build_user", "BuildUser"); u != "" {
		f.Build.User = u
	}
	if pw := strParam(p, "build_password", "BuildPassword"); pw != "" {
		f.Build.Password = pw
	}
	if l := strParam(p, "build_label", "Label"); l != "" {
		f.Build.Label = l
	}
	if f.Build.Label == "" {
		f.Build.Label = "build-1"
	}
	if f.Build.User == "" {
		f.Build.User = "root"
	}
	if !f.DoPrimary && !f.DoSecondary && !f.DoBuild {
		return f, fmt.Errorf("fleet: nothing selected (primary and/or secondary and/or build)")
	}
	return f, nil
}

// PrimaryFromFlatParams maps Web primary/fleet flat keys into PrimarySpec.
func PrimaryFromFlatParams(p map[string]any) PrimarySpec {
	s := PrimarySpec{
		Host:              strParam(p, "host", "primary_host", "Host"),
		User:              strParam(p, "user", "primary_user", "User"),
		Password:          strParam(p, "password", "primary_password", "Password"),
		SSHPrivateKey:     expandHome(strParam(p, "key", "primary_key", "ssh_private_key", "SSHPrivateKey")),
		KeyPassphrase:     strParam(p, "key_passphrase", "key_pass", "KeyPassphrase"),
		SNI:               strParam(p, "sni", "SNI"),
		DomainBase:        strParam(p, "domain_base", "DomainBase"),
		DomainPrimary:     strParam(p, "domain_primary", "DomainPrimary"),
		DomainVPN:         strParam(p, "domain_vpn", "DomainVPN"),
		DomainRedirect:    strParam(p, "domain_redirect", "DomainRedirect"),
		DomainEmail:       strParam(p, "le_email", "domain_email", "DomainEmail"),
		RedirectHTTPSPort: strParam(p, "redirect_https_port", "RedirectHTTPSPort"),
		AgentMTLSPort:     strParam(p, "agent_mtls_port", "AgentMTLSPort"),
		LampacPort:        strParam(p, "lampac_port", "LampacPort"),
		TelegramToken:     strParam(p, "tg_token", "telegram_token", "TelegramToken"),
		TelegramAdminID:   strParam(p, "tg_admin", "telegram_admin_id", "TelegramAdminID"),
	}
	if s.User == "" {
		s.User = "root"
	}
	if v, ok := p["ssh_port"]; ok {
		s.SSHPort = toInt(v)
	}
	s.DomainCFProxy = boolParam(p, "cf_proxy", false)
	s.WithLampac = boolParam(p, "with_lampac", false)
	s.WithGitRegistry = boolParam(p, "with_git", false)
	if s.SSHPrivateKey == "" {
		s.GenerateKey = true
	}
	return s
}

func mergePrimary(dst *PrimarySpec, src PrimarySpec) {
	if dst.Host == "" {
		dst.Host = src.Host
	}
	if dst.User == "" {
		dst.User = src.User
	}
	if dst.Password == "" {
		dst.Password = src.Password
	}
	if dst.SSHPrivateKey == "" {
		dst.SSHPrivateKey = src.SSHPrivateKey
	}
	if dst.KeyPassphrase == "" {
		dst.KeyPassphrase = src.KeyPassphrase
	}
	if dst.SNI == "" {
		dst.SNI = src.SNI
	}
	if dst.DomainBase == "" {
		dst.DomainBase = src.DomainBase
	}
	if dst.DomainPrimary == "" {
		dst.DomainPrimary = src.DomainPrimary
	}
	if dst.DomainVPN == "" {
		dst.DomainVPN = src.DomainVPN
	}
	if dst.DomainRedirect == "" {
		dst.DomainRedirect = src.DomainRedirect
	}
	if dst.DomainEmail == "" {
		dst.DomainEmail = src.DomainEmail
	}
	if dst.SSHPort == 0 {
		dst.SSHPort = src.SSHPort
	}
	if dst.RedirectHTTPSPort == "" {
		dst.RedirectHTTPSPort = src.RedirectHTTPSPort
	}
	if dst.AgentMTLSPort == "" {
		dst.AgentMTLSPort = src.AgentMTLSPort
	}
	if dst.LampacPort == "" {
		dst.LampacPort = src.LampacPort
	}
	if dst.TelegramToken == "" {
		dst.TelegramToken = src.TelegramToken
	}
	if dst.TelegramAdminID == "" {
		dst.TelegramAdminID = src.TelegramAdminID
	}
	if src.DomainCFProxy {
		dst.DomainCFProxy = true
	}
	if src.WithLampac {
		dst.WithLampac = true
	}
	if src.WithGitRegistry {
		dst.WithGitRegistry = true
	}
	if src.GenerateKey && dst.SSHPrivateKey == "" {
		dst.GenerateKey = true
	}
}

func toInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		return 0
	}
}
