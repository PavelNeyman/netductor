package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/sites"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func runSiteWizard() {
	siteID, name, rpiID, mtID := "home", "Home", "", "mt-home"
	rpiLAN := "192.168.88.2"
	mtHost, mtUser, mtPass := "192.168.88.1", "admin", ""
	mtPort := "22"
	doPush := false

	lang := detectLang()
	desc := TT(lang, "One site = MikroTik (routing) + RPi OpenWrt (VPN agent). SSH to MikroTik must work from THIS machine (usually home LAN).",
		"Один сайт = MikroTik (маршруты) + RPi OpenWrt (VPN agent). SSH до MikroTik с ЭТОЙ машины (обычно домашний LAN).")

	f := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(TT(detectLang(), "Site wizard", "Мастер сайта")).Description(desc),
			huh.NewInput().Title(TT(detectLang(), "Site ID", "Site ID")).Value(&siteID).Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("required")
				}
				return nil
			}),
			huh.NewInput().Title(TT(detectLang(), "Display name", "Отображаемое имя")).Value(&name),
			huh.NewInput().Title(TT(detectLang(), "RPi LAN IP (gateway for MT)", "RPi LAN IP (шлюз для MT)")).Value(&rpiLAN),
			huh.NewInput().Title(TT(detectLang(), "RPi edge device_id (if already approved)", "RPi edge device_id (если уже approved)")).Description(TT(detectLang(), "leave empty and fill later", "можно оставить пустым")).Value(&rpiID),
			huh.NewInput().Title(TT(detectLang(), "MikroTik identity name", "MikroTik identity")).Value(&mtID),
		),
		huh.NewGroup(
			huh.NewNote().Title(TT(detectLang(), "MikroTik SSH", "MikroTik SSH")).Description(TT(detectLang(), "Credentials are not stored on disk.", "Учётные данные не сохраняются на диск.")),
			huh.NewInput().Title(TT(detectLang(), "MT host / IP", "MT host / IP")).Value(&mtHost),
			huh.NewInput().Title(TT(detectLang(), "SSH user", "SSH пользователь")).Value(&mtUser),
			huh.NewInput().Title(TT(detectLang(), "SSH password", "SSH пароль")).Password(true).Value(&mtPass),
			huh.NewInput().Title(TT(detectLang(), "SSH port", "SSH порт")).Value(&mtPort),
			huh.NewConfirm().Title(TT(detectLang(), "Push RSC now?", "Залить RSC сейчас?")).Affirmative(TT(detectLang(), "Push", "Залить")).Negative(TT(detectLang(), "Only save site + show RSC", "Только сохранить сайт + показать RSC")).Value(&doPush),
		),
	).WithTheme(huh.ThemeCharm())

	if err := f.Run(); err != nil {
		fmt.Println(subStyle.Render("cancelled"))
		return
	}
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		return
	}
	if name == "" {
		name = siteID
	}
	s, err := sites.Upsert(sites.Site{
		ID: siteID, Name: name, RPiID: strings.TrimSpace(rpiID), MikroTikID: strings.TrimSpace(mtID),
	})
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(okStyle.Render(fmt.Sprintf("site saved: %s rpi=%s mt=%s", s.ID, s.RPiID, s.MikroTikID)))

	rsc, err := sites.RSCForSiteWithGateway(siteID, strings.TrimSpace(rpiLAN))
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(subStyle.Render("--- RSC ---"))
	fmt.Println(rsc)

	if !doPush {
		fmt.Println(subStyle.Render("Push skipped. Run Site wizard again or: netductor sites push ..."))
		printRPiHint(siteID, name)
		return
	}
	port, _ := strconv.Atoi(strings.TrimSpace(mtPort))
	c := mikrotik.ConnOpts{Host: mtHost, User: mtUser, Password: mtPass, Port: port}
	if err := mikrotik.PushRSC(mtHost, mtUser, mtPass, nil, rsc, port); err != nil {
		fmt.Println(errStyle.Render("push failed: " + err.Error()))
		fmt.Println(subStyle.Render("Check LAN reachability to " + mtHost + " and SSH enabled on RouterOS."))
		return
	}
	fmt.Println(okStyle.Render("RSC pushed to " + mtHost))
	// Install Mac/operator pubkey and best-effort disable password login
	if pub := operatorPubKeyFromTUI(); pub != "" {
		if err := mikrotik.HardenSSH(c, pub); err != nil {
			fmt.Println(errStyle.Render("SSH harden (key): " + err.Error()))
		} else {
			fmt.Println(okStyle.Render("operator pubkey installed on MikroTik; password login best-effort disabled"))
		}
	} else {
		fmt.Println(subStyle.Render("no operator pubkey in TUI profile — skip SSH harden (deploy primary first or set remote_key)"))
	}
	printRPiHint(siteID, name)
}

func operatorPubKeyFromTUI() string {
	s := loadTUISettings()
	key := strings.TrimSpace(s.RemoteKey)
	if key == "" {
		home, _ := os.UserHomeDir()
		key = home + "/.ssh/netductor_primary"
	}
	b, err := os.ReadFile(key + ".pub")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func printRPiHint(siteID, name string) {
	fmt.Println()
	fmt.Println(titleStyle.Render("Next: RPi OpenWrt"))
	h := "1) Flash OpenWrt on RPi, set LAN toward MT. "
	h += "2) Install netductor-agent, enroll to core, approve in Admin/TG. "
	h += "3) netductor sites add --id " + siteID + " --name \"" + name + "\" --rpi <device_id>"
	fmt.Println(subStyle.Render(h))
}

func runSitesListTUI() {
	list, err := sites.List()
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	if len(list) == 0 {
		fmt.Println(subStyle.Render("no sites — run Site setup wizard"))
		return
	}
	for _, s := range list {
		fmt.Printf("%s  %s  rpi=%s  mt=%s\n", s.ID, s.Name, s.RPiID, s.MikroTikID)
	}
}

func runMikroTikManage() {
	host, user, pass, portStr := "192.168.88.1", "admin", "", "22"
	action := "identity"
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(TT(detectLang(), "MikroTik manage", "Управление MikroTik")).Description(TT(detectLang(), "One-shot SSH · not stored", "Разовый SSH · не сохраняется")),
			huh.NewInput().Title(TT(detectLang(), "Host", "Host")).Value(&host),
			huh.NewInput().Title(TT(detectLang(), "User", "Пользователь")).Value(&user),
			huh.NewInput().Title(TT(detectLang(), "Password", "Пароль")).Password(true).Value(&pass),
			huh.NewInput().Title(TT(detectLang(), "Port", "Порт")).Value(&portStr),
			huh.NewSelect[string]().Title(TT(detectLang(), "Action", "Действие")).Options(
				huh.NewOption(TT(detectLang(), "Identity", "Identity"), "identity"),
				huh.NewOption(TT(detectLang(), "Resources (CPU/RAM)", "Ресурсы (CPU/RAM)"), "resource"),
				huh.NewOption(TT(detectLang(), "IP routes", "IP routes"), "routes"),
				huh.NewOption(TT(detectLang(), "Ping 1.1.1.1", "Ping 1.1.1.1"), "ping"),
				huh.NewOption(TT(detectLang(), "Push site RSC", "Залить site RSC"), "push"),
			).Value(&action),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil {
		return
	}
	port, _ := strconv.Atoi(portStr)
	c := mikrotik.ConnOpts{Host: host, User: user, Password: pass, Port: port}
	var out string
	var err error
	switch action {
	case "identity":
		out, err = mikrotik.Identity(c)
	case "resource":
		out, err = mikrotik.Resource(c)
	case "routes":
		out, err = mikrotik.Routes(c)
	case "ping":
		out, err = mikrotik.PingFromMT(c, "1.1.1.1")
	case "push":
		runSiteWizard()
		return
	}
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(out)
}

func runSNIForm() {
	presets := vpnListSNIOptions()
	choice := "yandex"
	ok := false
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title(TT(detectLang(), "Live SNI preset", "Пресет Live SNI")).Options(presets...).Value(&choice),
			huh.NewConfirm().Title(TT(detectLang(), "Apply on core + refresh secondary/links?", "Применить на primary + обновить secondary/links?")).Affirmative(TT(detectLang(), "Apply", "Применить")).Negative(TT(detectLang(), "Cancel", "Отмена")).Value(&ok),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		return
	}
	sniName := choice
	for _, p := range vpn.ListSNIPresets() {
		if p.Name == choice {
			sniName = p.SNI
			break
		}
	}
	if err := vpn.SetSNI(sniName); err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(okStyle.Render("active SNI: " + vpn.ActiveSNI()))
}

func vpnListSNIOptions() []huh.Option[string] {
	var opts []huh.Option[string]
	for _, p := range vpn.ListSNIPresets() {
		opts = append(opts, huh.NewOption(p.Name+" · "+p.SNI, p.Name))
	}
	if len(opts) == 0 {
		opts = append(opts, huh.NewOption("ya.ru", "yandex"))
	}
	return opts
}
