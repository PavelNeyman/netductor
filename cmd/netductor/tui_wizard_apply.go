package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func yesish(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "y" || s == "yes" || s == "1" || s == "true" || s == "да"
}

func wizBuildFields(id string, m *model) []wizField {
	s := loadTUISettings()
	lang := detectLang()
	switch id {
	case "primary":
		return []wizField{
			{Key: "host", Label: FormT(lang, "primary_host"), Value: s.RemoteHost, Placeholder: "x.x.x.x"},
			{Key: "user", Label: FormT(lang, "ssh_user"), Value: "root"},
			{Key: "password", Label: FormT(lang, "ssh_password_first"), Secret: true},
			{Key: "gen_key", Label: FormT(lang, "gen_ssh_key") + " (yes/no)", Value: "yes"},
			{Key: "key_path", Label: FormT(lang, "ssh_key_path"), Value: orDefault(s.RemoteKey, "~/.ssh/netductor_primary")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (empty=none)", Secret: true},
			{Key: "tg_token", Label: FormT(lang, "tg_token")},
			{Key: "tg_admin", Label: FormT(lang, "tg_admin")},
			{Key: "sni", Label: FormT(lang, "reality_sni"), Value: "api.vk.me"},
			{Key: "lampac", Label: FormT(lang, "addon_lampac") + " (yes/no)", Value: "no"},
		}
	case "secondary":
		return []wizField{
			{Key: "host", Label: "Secondary host / IP"},
			{Key: "user", Label: FormT(lang, "ssh_user"), Value: "root"},
			{Key: "password", Label: FormT(lang, "ssh_password_first"), Secret: true},
			{Key: "sni", Label: FormT(lang, "reality_sni"), Value: "api.vk.me"},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (primary key)", Secret: true},
		}
	case "openwrt":
		srv := "https://PRIMARY:8789"
		if s.RemoteHost != "" {
			srv = "https://" + s.RemoteHost + ":8789"
		}
		return []wizField{
			{Key: "router", Label: "Router LAN IP", Value: "192.168.1.1"},
			{Key: "password", Label: FormT(lang, "ssh_password_first"), Secret: true},
			{Key: "id", Label: FormT(lang, "edge_device_id"), Value: orDefault(s.LastEdgeID, "edge-1")},
			{Key: "arch", Label: "Agent arch", Value: "arm64"},
			{Key: "server", Label: "Primary mTLS URL", Value: srv},
			{Key: "net", Label: "Configure network? (yes/no)", Value: "no"},
			{Key: "lan_ip", Label: "LAN IP", Value: "192.168.50.1"},
			{Key: "wifi_ssid", Label: "Wi-Fi SSID 2.4"},
			{Key: "wifi_key", Label: "Wi-Fi password", Secret: true},
			{Key: "wan_proto", Label: "WAN (dhcp|static|pppoe)", Value: "dhcp"},
			{Key: "guest", Label: "Guest Wi-Fi? (yes/no)", Value: "no"},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (primary key)", Secret: true},
		}
	case "nvr":
		return []wizField{
			{Key: "action", Label: "Action (leases|add|probe|rec-start|rec-stop|status)", Value: "status"},
			{Key: "device_id", Label: FormT(lang, "edge_device_id"), Value: s.LastEdgeID},
			{Key: "cam_name", Label: FormT(lang, "cam_name")},
			{Key: "cam_ip", Label: FormT(lang, "cam_ip")},
			{Key: "cam_pass", Label: FormT(lang, "cam_pass"), Secret: true},
		}
	case "addons":
		return []wizField{
			{Key: "lampac", Label: FormT(lang, "addon_lampac") + " (yes/no)", Value: "yes"},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase"), Secret: true},
		}
	case "mikrotik":
		return []wizField{
			{Key: "host", Label: "MikroTik host"},
			{Key: "user", Label: "User", Value: "admin"},
			{Key: "password", Label: "Password", Secret: true},
			{Key: "name", Label: "Site name"},
		}
	default:
		return nil
	}
}

func (m model) runWizardApplyInTUI() string {
	lang := detectLang()
	s := loadTUISettings()
	switch m.wizTarget {
	case wizPrimary:
		host := m.fieldVal("host")
		if host == "" {
			return FormT(lang, "primary_host") + " required"
		}
		keyPath := expandHome(orDefault(m.fieldVal("key_path"), "~/.ssh/netductor_primary"))
		err := deploy.DeployPrimary(deploy.PrimaryOpts{
			Host: host, User: orDefault(m.fieldVal("user"), "root"), Password: m.fieldVal("password"),
			SSHPrivateKey: keyPath, GenerateKey: yesish(m.fieldVal("gen_key")),
			KeyPassphrase: m.fieldVal("key_pass"), WithLampac: yesish(m.fieldVal("lampac")),
			Version: deploy.Release, TelegramToken: m.fieldVal("tg_token"), TelegramAdminID: m.fieldVal("tg_admin"),
			SNI: orDefault(m.fieldVal("sni"), "api.vk.me"),
		})
		if err != nil {
			return err.Error()
		}
		s.RemoteHost = host
		s.RemoteUser = orDefault(m.fieldVal("user"), "root")
		s.RemoteKey = keyPath
		_ = saveTUISettings(s)
		return TT(lang, "Primary deploy finished: "+host, "Primary готов: "+host)
	case wizSecondary:
		if s.RemoteHost == "" || s.RemoteKey == "" {
			return FormT(lang, "set_primary_first")
		}
		err := deploy.DeploySecondary(deploy.SecondaryOpts{
			PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
			PrimaryKeyPassphrase: m.fieldVal("key_pass"),
			SecondaryHost: m.fieldVal("host"), SecondaryUser: orDefault(m.fieldVal("user"), "root"),
			SecondaryPass: m.fieldVal("password"), SNI: orDefault(m.fieldVal("sni"), "api.vk.me"),
		})
		if err != nil {
			return err.Error()
		}
		return TT(lang, "Secondary deploy finished", "Secondary готов")
	case wizOpenWrt:
		id := m.fieldVal("id")
		err := deploy.DeployEdge(deploy.EdgeOpts{
			PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
			PrimaryKeyPassphrase: m.fieldVal("key_pass"),
			RouterHost: m.fieldVal("router"), RouterUser: "root", RouterPass: m.fieldVal("password"),
			DeviceID: id, ServerURL: m.fieldVal("server"), AgentArch: orDefault(m.fieldVal("arch"), "arm64"),
			Version: deploy.Release, NetConfigure: yesish(m.fieldVal("net")),
			LANIP: m.fieldVal("lan_ip"), WiFiSSID24: m.fieldVal("wifi_ssid"), WiFiKey24: m.fieldVal("wifi_key"),
			WANProto: m.fieldVal("wan_proto"), GuestEnable: yesish(m.fieldVal("guest")),
		})
		if err != nil {
			return err.Error()
		}
		s.LastEdgeID = id
		_ = saveTUISettings(s)
		return TT(lang, "Edge provisioned: "+id, "Edge: "+id)
	case wizNVR:
		mm := &model{remoteHost: s.RemoteHost, remoteUser: s.RemoteUser, remoteKey: s.RemoteKey}
		switch m.fieldVal("action") {
		case "leases":
			return mm.runNetductor("nvr", "leases", m.fieldVal("device_id"), "--wait")
		case "add":
			return mm.runNetductor("nvr", "cameras", "add", "name="+m.fieldVal("cam_name"), "ip="+m.fieldVal("cam_ip"), "password="+m.fieldVal("cam_pass"))
		case "probe":
			return mm.runNetductor("nvr", "probe", m.fieldVal("cam_name"))
		case "rec-start":
			return mm.runNetductor("nvr", "record", "start", m.fieldVal("cam_name"))
		case "rec-stop":
			return mm.runNetductor("nvr", "record", "stop", m.fieldVal("cam_name"))
		default:
			return mm.runNetductor("nvr", "status")
		}
	case "addons":
		if s.RemoteHost == "" || s.RemoteKey == "" {
			return FormT(lang, "set_primary_first")
		}
		if !yesish(m.fieldVal("lampac")) {
			return TT(lang, "Nothing selected.", "Ничего не выбрано.")
		}
		out, err := deploy.RunOnPrimary(s.RemoteHost, orDefault(s.RemoteUser, "root"), s.RemoteKey, m.fieldVal("key_pass"), "netductor install lampac")
		if err != nil {
			return out + "\n" + err.Error()
		}
		return out + "\n" + TT(lang, "Lampac install finished.", "Lampac установлен.")
	case wizMikroTik:
		return TT(lang, "MikroTik: use Ops/CLI for full site push", "MikroTik: Ops/CLI")
	default:
		return "unknown target"
	}
}
