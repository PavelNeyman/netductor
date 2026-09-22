package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

func runSetupWizard() {
	lang := detectLang()
	var target string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(FormT(lang, "setup_title")).
				Description(FormT(lang, "setup_desc")).
				Options(
					huh.NewOption(FormT(lang, "opt_primary"), "primary"),
					huh.NewOption(FormT(lang, "opt_secondary"), "secondary"),
					huh.NewOption(FormT(lang, "opt_openwrt"), "openwrt"),
					huh.NewOption(FormT(lang, "opt_nvr"), "nvr"),
					huh.NewOption(FormT(lang, "opt_mikrotik"), "mikrotik"),
				).
				Value(&target),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	switch target {
	case "primary":
		wizardPrimary()
	case "secondary":
		wizardSecondary()
	case "openwrt":
		wizardOpenWrt()
	case "nvr":
		wizardNVR()
	case "mikrotik":
		runSiteWizard()
	}
}

func wizardPrimary() {
	lang := detectLang()
	s := loadTUISettings()
	var host, user, pass, keyPath, sni, tgToken, tgAdmin string
	var genKey, useKeyPass bool
	var keyPass, keyPass2 string
	user = "root"
	sni = "api.vk.me"
	host = s.RemoteHost
	keyPath = s.RemoteKey
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(FormT(lang, "primary_host")).
				Description(FormT(lang, "primary_host_desc")).Value(&host),
			huh.NewInput().Title(FormT(lang, "ssh_user")).Value(&user),
			huh.NewInput().Title(FormT(lang, "ssh_password_first")).
				EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewConfirm().Title(FormT(lang, "gen_ssh_key")).
				Description(FormT(lang, "gen_ssh_key_desc")).Value(&genKey),
			huh.NewInput().Title(FormT(lang, "ssh_key_path")).
				Placeholder("~/.ssh/netductor_primary").Value(&keyPath),
			huh.NewInput().Title(FormT(lang, "tg_token")).Value(&tgToken),
			huh.NewInput().Title(FormT(lang, "tg_admin")).Value(&tgAdmin),
			huh.NewInput().Title(FormT(lang, "reality_sni")).Placeholder("api.vk.me").Value(&sni),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if genKey {
		_ = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().Title(FormT(lang, "key_use_passphrase")).
					Description(FormT(lang, "key_use_passphrase_desc")).Value(&useKeyPass),
			),
		).WithTheme(huh.ThemeCharm()).Run()
		if useKeyPass {
			_ = huh.NewForm(
				huh.NewGroup(
					huh.NewInput().Title(FormT(lang, "key_passphrase")).
						EchoMode(huh.EchoModePassword).Value(&keyPass),
					huh.NewInput().Title(FormT(lang, "key_passphrase_confirm")).
						EchoMode(huh.EchoModePassword).Value(&keyPass2),
				),
			).WithTheme(huh.ThemeCharm()).Run()
			if keyPass != keyPass2 {
				fmt.Println(errStyle.Render(FormT(lang, "key_passphrase_mismatch")))
				return
			}
		}
	}
	if sni == "" {
		sni = "api.vk.me"
	}
	if host == "" {
		var doInstall bool
		_ = huh.NewForm(huh.NewGroup(huh.NewConfirm().
			Title(FormT(lang, "run_install_here")).
			Value(&doInstall))).Run()
		if doInstall {
			cmd := exec.Command("netductor", "install")
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			_ = cmd.Run()
		}
		_ = exec.Command("netductor", "vpn", "set-sni", sni).Run()
		_ = exec.Command("netductor", "fleet", "bootstrap").Run()
		out, _ := exec.Command("netductor", "doctor").CombinedOutput()
		fmt.Print(string(out))
		return
	}
	fmt.Println(okStyle.Render(TT(lang, "→ deploy primary "+host, "→ деплой primary "+host)))
	err := deploy.DeployPrimary(deploy.PrimaryOpts{
		Host: host, User: user, Password: pass, SSHPrivateKey: keyPath, GenerateKey: genKey,
		KeyPassphrase: keyPass,
		Version: deploy.Release, TelegramToken: tgToken, TelegramAdminID: tgAdmin, SNI: sni,
	})
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	s.RemoteHost = host
	s.RemoteUser = user
	if keyPath != "" {
		s.RemoteKey = keyPath
	} else {
		home, _ := os.UserHomeDir()
		s.RemoteKey = home + "/.ssh/netductor_primary"
	}
	_ = saveTUISettings(s)
	fmt.Println(okStyle.Render(TT(lang, "Primary done. Key saved for secondary/edge.", "Primary готов. Ключ сохранён для secondary/edge.")))
}

func wizardSecondary() {
	lang := detectLang()
	s := loadTUISettings()
	var host, user, pass, sni string
	user = "root"
	sni = "api.vk.me"
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(FormT(lang, "secondary_host")).Value(&host),
			huh.NewInput().Title(FormT(lang, "ssh_user")).Value(&user),
			huh.NewInput().Title(FormT(lang, "ssh_password_first")).
				EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewInput().Title(FormT(lang, "reality_sni")).Placeholder("api.vk.me").Value(&sni),
			huh.NewNote().Title(TT(lang, "Primary", "Primary")).
				Description(TT(lang, "Uses TUI remote_host + remote_key (set by primary deploy)",
					"Берёт remote_host + remote_key из настроек TUI (после primary)")),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if host == "" || pass == "" {
		fmt.Println(errStyle.Render(FormT(lang, "host_pass_required")))
		return
	}
	if s.RemoteHost == "" || s.RemoteKey == "" {
		fmt.Println(errStyle.Render(FormT(lang, "set_primary_first")))
		return
	}
	fmt.Println(okStyle.Render(TT(lang, "→ deploy secondary "+host, "→ деплой secondary "+host)))
	var keyPass string
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(FormT(lang, "key_passphrase")).
				Description(TT(lang, "If primary key has passphrase (leave empty if none)",
					"Если ключ primary с passphrase (пусто = без фразы)")).
				EchoMode(huh.EchoModePassword).Value(&keyPass),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	err := deploy.DeploySecondary(deploy.SecondaryOpts{
		PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
		PrimaryKeyPassphrase: keyPass,
		SecondaryHost: host, SecondaryUser: user, SecondaryPass: pass, SNI: sni,
	})
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	s.SecondaryHost = host
	_ = saveTUISettings(s)
	fmt.Println(okStyle.Render(TT(lang, "Secondary provisioned.", "Secondary задеплоен.")))
}

func wizardOpenWrt() {
	lang := detectLang()
	s := loadTUISettings()
	var host, user, pass, id, arch, server string
	var guestOn bool
	var guestSSID, guestPIN string
	user = "root"
	arch = "arm64"
	guestSSID = "Guest"
	if s.RemoteHost != "" {
		server = "https://" + s.RemoteHost + ":8789"
	}
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(FormT(lang, "router_lan")).Value(&host),
			huh.NewInput().Title(FormT(lang, "ssh_user")).Value(&user),
			huh.NewInput().Title(FormT(lang, "ssh_password_ifkey")).
				EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewInput().Title(FormT(lang, "device_id")).Value(&id),
			huh.NewInput().Title(FormT(lang, "agent_arch")).
				Description("arm64 | armv7 | amd64 | mipsle").Value(&arch),
			huh.NewInput().Title(FormT(lang, "primary_mtls")).
				Description("https://IP:8789").Value(&server),
			huh.NewConfirm().Title(TT(lang, "Enable guest Wi‑Fi?", "Включить гостевой Wi‑Fi?")).
				Description(TT(lang, "Hidden SSID, desk PIN, captive (optional)", "Скрытый SSID, PIN кассы, captive")).
				Value(&guestOn),
			huh.NewInput().Title(TT(lang, "Guest SSID", "Guest SSID")).Value(&guestSSID),
			huh.NewInput().Title(TT(lang, "Desk PIN (staff)", "PIN кассы")).
				EchoMode(huh.EchoModePassword).Value(&guestPIN),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if host == "" || id == "" {
		fmt.Println(errStyle.Render(FormT(lang, "host_id_required")))
		return
	}
	if guestOn && guestPIN == "" {
		fmt.Println(errStyle.Render(TT(lang, "Desk PIN required when guest enabled", "Нужен PIN кассы для guest")))
		return
	}
	var netOn bool
	var lanIP, lanMask, dhcpStart, dhcpLimit, wifiSSID, wifiKey, wanProto, wanIP, wanMask, wanGW, wanDNS string
	lanMask = "255.255.255.0"
	dhcpStart = "100"
	dhcpLimit = "150"
	wanProto = "dhcp"
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title(FormT(lang, "net_configure")).
				Description(FormT(lang, "net_configure_desc")).Value(&netOn),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if netOn {
		_ = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title(FormT(lang, "lan_ip")).Placeholder("192.168.50.1").Value(&lanIP),
				huh.NewInput().Title(FormT(lang, "lan_mask")).Value(&lanMask),
				huh.NewInput().Title(FormT(lang, "dhcp_start")).Value(&dhcpStart),
				huh.NewInput().Title(FormT(lang, "dhcp_limit")).Value(&dhcpLimit),
				huh.NewInput().Title(FormT(lang, "wifi_ssid")).Value(&wifiSSID),
				huh.NewInput().Title(FormT(lang, "wifi_key")).EchoMode(huh.EchoModePassword).Value(&wifiKey),
				huh.NewInput().Title(FormT(lang, "wan_proto")).Placeholder("dhcp").Value(&wanProto),
			),
		).WithTheme(huh.ThemeCharm()).Run()
		if strings.ToLower(strings.TrimSpace(wanProto)) == "static" {
			_ = huh.NewForm(
				huh.NewGroup(
					huh.NewInput().Title(FormT(lang, "wan_ip")).Value(&wanIP),
					huh.NewInput().Title(FormT(lang, "wan_mask")).Placeholder("255.255.255.0").Value(&wanMask),
					huh.NewInput().Title(FormT(lang, "wan_gateway")).Value(&wanGW),
					huh.NewInput().Title(FormT(lang, "wan_dns")).Placeholder("1.1.1.1").Value(&wanDNS),
				),
			).WithTheme(huh.ThemeCharm()).Run()
		}
	}
	fmt.Println(okStyle.Render(TT(lang, "→ edge deploy "+host, "→ edge деплой "+host)))
	var keyPass string
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(FormT(lang, "key_passphrase")).
				Description(TT(lang, "Primary key passphrase if any (empty = none)",
					"Passphrase ключа primary, если есть (пусто = нет)")).
				EchoMode(huh.EchoModePassword).Value(&keyPass),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	err := deploy.DeployEdge(deploy.EdgeOpts{
		PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
		PrimaryKeyPassphrase: keyPass,
		RouterHost: host, RouterUser: user, RouterPass: pass,
		DeviceID: id, ServerURL: server, AgentArch: arch, Version: deploy.Release,
		NetConfigure: netOn, LANIP: lanIP, LANMask: lanMask, DHCPStart: dhcpStart, DHCPLimit: dhcpLimit,
		WiFiSSID: wifiSSID, WiFiKey: wifiKey, WANProto: wanProto, WANIP: wanIP, WANMask: wanMask,
		WANGateway: wanGW, WANDNS: wanDNS,
		GuestEnable: guestOn, GuestSSID: guestSSID, GuestPIN: guestPIN,
	})
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	s.LastEdgeID = id
	_ = saveTUISettings(s)
	fmt.Println(okStyle.Render(TT(lang, "Approve on primary: edge pending → edge approve "+id,
		"На primary: edge pending → edge approve "+id)))
	if s.RemoteHost != "" {
		m := &model{remoteHost: s.RemoteHost, remoteUser: s.RemoteUser, remoteKey: s.RemoteKey}
		fmt.Print(m.runNetductor("edge", "pending"))
	}
}

func wizardNVR() {
	lang := detectLang()
	s := loadTUISettings()
	var deviceID, camName, camIP, camMAC, camPass, action string
	deviceID = s.LastEdgeID
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title(FormT(lang, "nvr_action")).Options(
				huh.NewOption(TT(lang, "List DHCP leases (edge)", "DHCP leases (edge)"), "leases"),
				huh.NewOption(TT(lang, "Add camera", "Добавить камеру"), "add"),
				huh.NewOption(TT(lang, "Probe camera", "Probe камеры"), "probe"),
				huh.NewOption(TT(lang, "Record start", "Запись start"), "rec-start"),
				huh.NewOption(TT(lang, "Record stop", "Запись stop"), "rec-stop"),
				huh.NewOption(TT(lang, "NVR status", "Статус NVR"), "status"),
			).Value(&action),
			huh.NewInput().Title(FormT(lang, "edge_device_id")).Value(&deviceID),
			huh.NewInput().Title(FormT(lang, "cam_name")).Value(&camName),
			huh.NewInput().Title(FormT(lang, "cam_ip")).Value(&camIP),
			huh.NewInput().Title(FormT(lang, "cam_mac")).Value(&camMAC),
			huh.NewInput().Title(FormT(lang, "cam_pass")).
				EchoMode(huh.EchoModePassword).Value(&camPass),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	m := &model{remoteHost: s.RemoteHost, remoteUser: s.RemoteUser, remoteKey: s.RemoteKey}
	switch action {
	case "leases":
		fmt.Print(m.runNetductor("nvr", "leases", deviceID, "--wait"))
	case "add":
		args := []string{"nvr", "cameras", "add", "name=" + camName, "ip=" + camIP, "mac=" + camMAC, "password=" + camPass}
		fmt.Print(m.runNetductor(args...))
	case "probe":
		fmt.Print(m.runNetductor("nvr", "probe", camName))
	case "rec-start":
		fmt.Print(m.runNetductor("nvr", "record", "start", camName))
	case "rec-stop":
		fmt.Print(m.runNetductor("nvr", "record", "stop", camName))
	default:
		fmt.Print(m.runNetductor("nvr", "status"))
	}
}
