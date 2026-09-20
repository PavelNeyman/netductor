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

	desc := "One site = MikroTik (routing) + RPi OpenWrt (VPN agent). "
	desc += "SSH to MikroTik must work from THIS machine (usually home LAN)."

	f := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Site wizard").Description(desc),
			huh.NewInput().Title("Site ID").Value(&siteID).Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("required")
				}
				return nil
			}),
			huh.NewInput().Title("Display name").Value(&name),
			huh.NewInput().Title("RPi LAN IP (gateway for MT)").Value(&rpiLAN),
			huh.NewInput().Title("RPi edge device_id (if already approved)").Description("leave empty and fill later").Value(&rpiID),
			huh.NewInput().Title("MikroTik identity name").Value(&mtID),
		),
		huh.NewGroup(
			huh.NewNote().Title("MikroTik SSH").Description("Credentials are not stored on disk."),
			huh.NewInput().Title("MT host / IP").Value(&mtHost),
			huh.NewInput().Title("SSH user").Value(&mtUser),
			huh.NewInput().Title("SSH password").Password(true).Value(&mtPass),
			huh.NewInput().Title("SSH port").Value(&mtPort),
			huh.NewConfirm().Title("Push RSC now?").Affirmative("Push").Negative("Only save site + show RSC").Value(&doPush),
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
			huh.NewNote().Title("MikroTik manage").Description("One-shot SSH · not stored"),
			huh.NewInput().Title("Host").Value(&host),
			huh.NewInput().Title("User").Value(&user),
			huh.NewInput().Title("Password").Password(true).Value(&pass),
			huh.NewInput().Title("Port").Value(&portStr),
			huh.NewSelect[string]().Title("Action").Options(
				huh.NewOption("Identity", "identity"),
				huh.NewOption("Resources (CPU/RAM)", "resource"),
				huh.NewOption("IP routes", "routes"),
				huh.NewOption("Ping 1.1.1.1", "ping"),
				huh.NewOption("Push site RSC", "push"),
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
			huh.NewSelect[string]().Title("Live SNI preset").Options(presets...).Value(&choice),
			huh.NewConfirm().Title("Apply on core + refresh relay/links?").Affirmative("Apply").Negative("Cancel").Value(&ok),
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
