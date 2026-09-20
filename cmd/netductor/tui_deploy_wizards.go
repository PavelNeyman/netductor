package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

func runSetupWizard() {
	lang := detectLang()
	var target string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(TT(lang, "Setup wizard — what are we configuring?", "Мастер — что настраиваем?")).
				Description(TT(lang, "Questions first, then fully automatic apply", "Сначала вопросы, затем авто-применение")).
				Options(
					huh.NewOption(TT(lang, "Primary VPS (abroad control plane)", "Primary VPS (control plane за рубежом)"), "primary"),
					huh.NewOption(TT(lang, "Secondary VPS (RU VPN entry only)", "Secondary VPS (только RU VPN entry)"), "secondary"),
					huh.NewOption(TT(lang, "OpenWrt router / RPi (edge agent)", "OpenWrt / RPi (edge agent)"), "openwrt"),
					huh.NewOption(TT(lang, "Cameras / NVR", "Камеры / NVR"), "nvr"),
					huh.NewOption(TT(lang, "MikroTik (ROS routes / site)", "MikroTik (ROS / сайт)"), "mikrotik"),
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
	var genKey bool
	user = "root"
	sni = "api.vk.me"
	host = s.RemoteHost
	keyPath = s.RemoteKey
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(TT(lang, "Primary VPS host / IP", "Primary VPS host / IP")).
				Description(TT(lang, "Empty = install on THIS machine", "Пусто = install на ЭТОЙ машине")).Value(&host),
			huh.NewInput().Title(TT(lang, "SSH user", "SSH пользователь")).Value(&user),
			huh.NewInput().Title(TT(lang, "SSH password (first login only)", "SSH пароль (только первый вход)")).
				EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewConfirm().Title(TT(lang, "Generate new SSH key?", "Сгенерировать новый SSH-ключ?")).
				Description(TT(lang, "No = use path below", "Нет = путь ниже")).Value(&genKey),
			huh.NewInput().Title(TT(lang, "SSH private key path", "Путь к SSH private key")).
				Placeholder("~/.ssh/netductor_primary").Value(&keyPath),
			huh.NewInput().Title(TT(lang, "Telegram bot token", "Токен Telegram-бота")).Value(&tgToken),
			huh.NewInput().Title(TT(lang, "Telegram admin user id", "Telegram admin user id")).Value(&tgAdmin),
			huh.NewInput().Title(TT(lang, "Reality SNI", "Reality SNI")).Placeholder("api.vk.me").Value(&sni),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if sni == "" {
		sni = "api.vk.me"
	}
	if host == "" {
		var doInstall bool
		_ = huh.NewForm(huh.NewGroup(huh.NewConfirm().
			Title(TT(lang, "Run netductor install here?", "Запустить netductor install здесь?")).
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
			huh.NewInput().Title(TT(lang, "Secondary (RU) IP / host", "Secondary (RU) IP / host")).Value(&host),
			huh.NewInput().Title(TT(lang, "SSH user", "SSH пользователь")).Value(&user),
			huh.NewInput().Title(TT(lang, "SSH password (first login)", "SSH пароль (первый вход)")).
				EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewInput().Title(TT(lang, "Reality SNI", "Reality SNI")).Placeholder("api.vk.me").Value(&sni),
			huh.NewNote().Title(TT(lang, "Primary", "Primary")).
				Description(TT(lang, "Uses TUI remote_host + remote_key (set by primary deploy)",
					"Берёт remote_host + remote_key из настроек TUI (после primary)")),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if host == "" || pass == "" {
		fmt.Println(errStyle.Render(TT(lang, "host and password required", "нужны host и пароль")))
		return
	}
	if s.RemoteHost == "" || s.RemoteKey == "" {
		fmt.Println(errStyle.Render(TT(lang, "set primary remote in Settings / primary wizard first",
			"сначала primary в Настройках / мастере primary")))
		return
	}
	fmt.Println(okStyle.Render(TT(lang, "→ deploy secondary "+host, "→ деплой secondary "+host)))
	err := deploy.DeploySecondary(deploy.SecondaryOpts{
		PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
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
	user = "root"
	arch = "arm64"
	if s.RemoteHost != "" {
		server = "https://" + s.RemoteHost + ":8789"
	}
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(TT(lang, "Router LAN IP", "LAN IP роутера")).Value(&host),
			huh.NewInput().Title(TT(lang, "SSH user", "SSH пользователь")).Value(&user),
			huh.NewInput().Title(TT(lang, "SSH password (if no key)", "SSH пароль (если нет ключа)")).
				EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewInput().Title(TT(lang, "Device ID", "Device ID")).Value(&id),
			huh.NewInput().Title(TT(lang, "Agent arch", "Arch агента")).
				Description("arm64 | armv7 | amd64 | mipsle").Value(&arch),
			huh.NewInput().Title(TT(lang, "Primary mTLS URL", "Primary mTLS URL")).
				Description("https://IP:8789").Value(&server),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if host == "" || id == "" {
		fmt.Println(errStyle.Render(TT(lang, "host and device id required", "нужны host и device id")))
		return
	}
	fmt.Println(okStyle.Render(TT(lang, "→ edge deploy "+host, "→ edge деплой "+host)))
	err := deploy.DeployEdge(deploy.EdgeOpts{
		PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
		RouterHost: host, RouterUser: user, RouterPass: pass,
		DeviceID: id, ServerURL: server, AgentArch: arch, Version: deploy.Release,
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
			huh.NewSelect[string]().Title(TT(lang, "NVR action", "Действие NVR")).Options(
				huh.NewOption(TT(lang, "List DHCP leases (edge)", "DHCP leases (edge)"), "leases"),
				huh.NewOption(TT(lang, "Add camera", "Добавить камеру"), "add"),
				huh.NewOption(TT(lang, "Probe camera", "Probe камеры"), "probe"),
				huh.NewOption(TT(lang, "Record start", "Запись start"), "rec-start"),
				huh.NewOption(TT(lang, "Record stop", "Запись stop"), "rec-stop"),
				huh.NewOption(TT(lang, "NVR status", "Статус NVR"), "status"),
			).Value(&action),
			huh.NewInput().Title(TT(lang, "Edge device id", "Edge device id")).Value(&deviceID),
			huh.NewInput().Title(TT(lang, "Camera name (add)", "Имя камеры (add)")).Value(&camName),
			huh.NewInput().Title(TT(lang, "Camera IP", "IP камеры")).Value(&camIP),
			huh.NewInput().Title(TT(lang, "Camera MAC", "MAC камеры")).Value(&camMAC),
			huh.NewInput().Title(TT(lang, "Camera password", "Пароль камеры")).
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
