package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

func runSetupWizard() {
	var target string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Setup wizard — what are we configuring?").
				Description("Questions first, then fully automatic apply").
				Options(
					huh.NewOption("Primary VPS (abroad control plane)", "primary"),
					huh.NewOption("Secondary VPS (RU VPN entry only)", "secondary"),
					huh.NewOption("OpenWrt router / RPi (edge agent)", "openwrt"),
					huh.NewOption("Cameras / NVR", "nvr"),
					huh.NewOption("MikroTik (ROS routes / site)", "mikrotik"),
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
	s := loadTUISettings()
	var host, user, pass, keyPath, sni, tgToken, tgAdmin string
	var genKey bool
	user = "root"
	sni = "api.vk.me"
	host = s.RemoteHost
	keyPath = s.RemoteKey
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Primary VPS host / IP").Description("Empty = install on THIS machine").Value(&host),
			huh.NewInput().Title("SSH user").Value(&user),
			huh.NewInput().Title("SSH password (first login only)").EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewConfirm().Title("Generate new SSH key?").Description("No = use path below").Value(&genKey),
			huh.NewInput().Title("SSH private key path").Placeholder("~/.ssh/netductor_primary").Value(&keyPath),
			huh.NewInput().Title("Telegram bot token").Value(&tgToken),
			huh.NewInput().Title("Telegram admin user id").Value(&tgAdmin),
			huh.NewInput().Title("Reality SNI").Placeholder("api.vk.me").Value(&sni),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if sni == "" {
		sni = "api.vk.me"
	}
	if host == "" {
		var doInstall bool
		_ = huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Run netductor install here?").Value(&doInstall))).Run()
		if doInstall {
			cmd := exec.Command("netductor", "install")
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			_ = cmd.Run()
		}
		_ = exec.Command("netductor", "vpn", "set-sni", sni).Run()
		_ = exec.Command("netductor", "fleet", "bootstrap").Run()
		out, _ := exec.Command("netductor", "doctor").CombinedOutput()
		fmt.Print(string(out))
		fmt.Println(subStyle.Render("Next: Setup wizard → Secondary"))
		return
	}
	if keyPath == "" {
		home, _ := os.UserHomeDir()
		keyPath = home + "/.ssh/netductor_primary"
		genKey = true
	}
	fmt.Println(okStyle.Render("→ deploy primary " + host + " from this machine"))
	err := deploy.DeployPrimary(deploy.PrimaryOpts{
		Host: host, User: user, Password: pass,
		SSHPrivateKey: keyPath, GenerateKey: genKey,
		Version: "0.8.3", TelegramToken: tgToken, TelegramAdminID: tgAdmin, SNI: sni,
	})
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	s.RemoteHost = host
	s.RemoteUser = user
	s.RemoteKey = keyPath
	s.RemotePassword = ""
	_ = saveTUISettings(s)
	fmt.Println(okStyle.Render("saved TUI profile → " + tuiConfigPath()))
	fmt.Println(subStyle.Render("Next: Setup wizard → Secondary"))
}

func wizardSecondary() {
	s := loadTUISettings()
	var host, user, pass, sni string
	user = "root"
	sni = "api.vk.me"
	host = s.SecondaryHost
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Secondary (RU) IP / host").Value(&host),
			huh.NewInput().Title("SSH user").Value(&user),
			huh.NewInput().Title("SSH password (first login)").EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewInput().Title("Reality SNI").Placeholder("api.vk.me").Value(&sni),
			huh.NewNote().Title("Primary").Description("Uses TUI remote_host + remote_key (set by primary deploy)"),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if err != nil || host == "" || pass == "" {
		fmt.Println(errStyle.Render("host and password required"))
		return
	}
	if sni == "" {
		sni = "api.vk.me"
	}
	if s.RemoteHost != "" && s.RemoteKey != "" {
		fmt.Println(okStyle.Render("→ deploy secondary via primary " + s.RemoteHost))
		err := deploy.DeploySecondary(deploy.SecondaryOpts{
			PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
			SecondaryHost: host, SecondaryUser: user, SecondaryPass: pass, SNI: sni,
		})
		if err != nil {
			fmt.Println(errStyle.Render(err.Error()))
			return
		}
	} else {
		fmt.Println(okStyle.Render("→ fleet provision-secondary (local primary)"))
		cmd := exec.Command("netductor", "fleet", "provision-secondary",
			"--host", host, "--user", user, "--password", pass, "--sni", sni)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println(errStyle.Render(err.Error()))
			return
		}
	}
	s.SecondaryHost = host
	_ = saveTUISettings(s)
	if s.RemoteHost != "" {
		m := &model{remoteHost: s.RemoteHost, remoteUser: s.RemoteUser, remoteKey: s.RemoteKey}
		fmt.Print(m.runNetductor("fleet", "status"))
	} else {
		out, _ := exec.Command("netductor", "fleet", "status").CombinedOutput()
		fmt.Print(string(out))
	}
	fmt.Println(subStyle.Render("Next: OpenWrt / edge or Cameras"))
}

func wizardOpenWrt() {
	s := loadTUISettings()
	var host, user, pass, id, arch, server string
	user = "root"
	arch = "arm64"
	id = s.LastEdgeID
	if id == "" {
		id = "home-owrt-1"
	}
	if s.RemoteHost != "" {
		server = "http://" + s.RemoteHost + ":8787"
	}
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Router LAN IP").Value(&host),
			huh.NewInput().Title("SSH user").Value(&user),
			huh.NewInput().Title("SSH password (if no key)").EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewInput().Title("Device ID").Value(&id),
			huh.NewInput().Title("Agent arch").Description("arm64 | armv7 | amd64 | mipsle").Value(&arch),
			huh.NewInput().Title("Primary API URL").Value(&server),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if host == "" || id == "" {
		fmt.Println(errStyle.Render("host and device id required"))
		return
	}
	fmt.Println(okStyle.Render("→ edge deploy " + host))
	err = deploy.DeployEdge(deploy.EdgeOpts{
		PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
		RouterHost: host, RouterUser: user, RouterPass: pass,
		DeviceID: id, ServerURL: server, AgentArch: arch, Version: "0.8.3",
	})
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	s.LastEdgeID = id
	_ = saveTUISettings(s)
	fmt.Println(okStyle.Render("Approve on primary: edge pending → edge approve " + id))
	if s.RemoteHost != "" {
		m := &model{remoteHost: s.RemoteHost, remoteUser: s.RemoteUser, remoteKey: s.RemoteKey}
		fmt.Print(m.runNetductor("edge", "pending"))
	}
}

func wizardNVR() {
	s := loadTUISettings()
	var deviceID, camName, camIP, camMAC, camPass, action string
	deviceID = s.LastEdgeID
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("NVR action").Options(
				huh.NewOption("List DHCP leases (edge)", "leases"),
				huh.NewOption("Add camera", "add"),
				huh.NewOption("Probe camera", "probe"),
				huh.NewOption("Record start", "rec-start"),
				huh.NewOption("Record stop", "rec-stop"),
				huh.NewOption("NVR status", "status"),
			).Value(&action),
			huh.NewInput().Title("Edge device id").Value(&deviceID),
			huh.NewInput().Title("Camera name (add)").Value(&camName),
			huh.NewInput().Title("Camera IP").Value(&camIP),
			huh.NewInput().Title("Camera MAC").Value(&camMAC),
			huh.NewInput().Title("Camera password").EchoMode(huh.EchoModePassword).Value(&camPass),
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
