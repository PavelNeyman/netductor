package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"

	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/vpn"
)



func formVpnAdd() {
	var name, note string
	var ok bool
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(TT(detectLang(), "VPN user name", "Имя VPN-пользователя")).Description(TT(detectLang(), "letters, digits, _ -", "буквы, цифры, _ -")).Value(&name),
			huh.NewInput().Title(TT(detectLang(), "Note (optional)", "Заметка (опционально)")).Value(&note),
			huh.NewConfirm().Title(TT(detectLang(), "Create user?", "Создать пользователя?")).Value(&ok),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		fmt.Println(subStyle.Render("cancelled"))
		return
	}
	uuid, err := vpn.Add(name, note)
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(okStyle.Render("user created " + uuid))
	fmt.Println(subStyle.Render("QR file: " + vpn.QRPath(name)))
	link := vpn.PreferredVLESSLink(name, uuid)
	if link != "" {
		fmt.Println(subStyle.Render("VLESS: " + link))
	}
}

func formVpnRename() {
	var old, newN string
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(TT(detectLang(), "Current name", "Текущее имя")).Value(&old),
			huh.NewInput().Title(TT(detectLang(), "New name", "Новое имя")).Value(&newN),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || old == "" || newN == "" {
		return
	}
	out, err := vpn.Rename(old, newN)
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(okStyle.Render("renamed → " + out))
}

func formVpnSub() {
	var name string
	f := huh.NewForm(huh.NewGroup(huh.NewInput().Title(TT(detectLang(), "VPN user", "VPN-пользователь")).Value(&name))).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || name == "" {
		return
	}
	link := vpn.PreferredVLESSLink(name, "")
	if link == "" {
		fmt.Println(errStyle.Render("no link for " + name))
		return
	}
	fmt.Println(link)
}

func formBackupPeer() {
	fmt.Println(okStyle.Render(install.BackupPeerStatus()))
}

func formSession() {
	var note string
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(TT(detectLang(), "Note", "Заметка")).Value(&note),
		),
	).WithTheme(huh.ThemeCharm())
	_ = f.Run()
	fmt.Println(subStyle.Render("use: netductor session create"))
}


func formInstall(prepare bool) {
	var ok bool
	title := "Run netductor install?"
	if prepare {
		title = "Run netductor install --prepare?"
	}
	f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(title).Value(&ok))).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		fmt.Println(subStyle.Render("cancelled"))
		return
	}
	args := []string{"install"}
	if prepare {
		args = append(args, "--prepare")
	}
	cmd := exec.Command("netductor", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	_ = cmd.Run()
}

func formOwrtInstall() {
	fmt.Println(errStyle.Render("install-openwrt.sh not found — use edge provision via Setup wizard"))
}

func formBuild() {
	var ok bool
	f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(TT(detectLang(), "go build ./cmd/netductor ?", "go build ./cmd/netductor ?")).Value(&ok))).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		return
	}
	cmd := exec.Command("go", "build", "-o", "netductor", "./cmd/netductor")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	_ = cmd.Run()
}
