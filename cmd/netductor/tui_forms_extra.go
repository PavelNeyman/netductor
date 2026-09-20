package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func menuItemsFor(mode runMode, lang tuiLang) []list.Item {
	wizDesc := "Primary / Secondary / OpenWrt / MikroTik — questions then auto"
	if lang == langRU {
		wizDesc = "Primary / Secondary / OpenWrt / MikroTik — вопросы, затем авто"
	}
	items := []list.Item{
		menuItem{"wizard", wizDesc, "wizard"},
	}
	switch mode {
	case modeVPS:
		items = append(items,
			menuItem{"Install / upgrade stack", "netductor install (idempotent)", "install"},
			menuItem{"Lampac (primary)", "primary only (Docker)", "apply-lampac"},
			menuItem{"Set hostname", "nd-primary / nd-secondary / …", "hostname"},
		)
	case modeOpenWRT:
		items = append(items,
			menuItem{"Agent install instructions", "outbound netductor-agent", "agent-help"},
			menuItem{"Run install-openwrt.sh", "if present on device", "owrt-install"},
			menuItem{"Check agent config", "", "agent-cfg"},
		)
	case modeWorkstation:
		items = append(items,
			menuItem{"Bootstrap one-liner", "copy-paste for VPS", "bootstrap"},
			menuItem{"Build netductor (local Go)", "", "build"},
			menuItem{"MikroTik manage", "identity / routes / push", "mt-manage"},
		)
	default:
		items = append(items,
			menuItem{"VPN — add user", "", "vpn-add"},
			menuItem{"VPN — link / QR", "", "vpn-sub"},
			menuItem{"Session create", "", "session"},
			menuItem{"Live SNI", "", "sni-live"},
			menuItem{"Edge devices", "", "edge-list"},
		)
	}
	toolsDesc := "diagnostics & ops"
	if lang == langRU {
		toolsDesc = "диагностика и операции"
	}
	items = append(items,
		menuItem{"tools", toolsDesc, "noop"},
		menuItem{"Doctor", "health checks", "doctor"},
		menuItem{"Status", "systemd units", "status"},
		menuItem{"Fleet status", "primary / secondary", "fleet-status"},
		menuItem{"Nodes registry", "list", "nodes-list"},
		menuItem{"Secondary status", "agent online", "secondary-status"},
		menuItem{"Backup now", "encrypted + peer", "backup-now"},
		menuItem{"VPN users → secondary", "relay sync", "relay-sync"},
		menuItem{"VPN users", "list", "vpn-list"},
		menuItem{"Refresh VPN links", "prefer secondary", "vpn-refresh"},
		menuItem{"Probes", "connectivity", "probe"},
		menuItem{"SSH known hosts", "TOFU", "ssh-hosts"},
		menuItem{"Audit tail", "events", "audit-tail"},
		menuItem{"change-mode", "", "change-mode"},
		menuItem{"quit", "", "quit"},
	)
	return items
}

func newList(title string, items []list.Item, w, h int) list.Model {
	d := list.NewDefaultDelegate()
	d.SetHeight(2)
	d.SetSpacing(1)
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(lipgloss.Color("205")).BorderForeground(lipgloss.Color("205")).Bold(true)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(lipgloss.Color("218"))
	d.Styles.NormalTitle = d.Styles.NormalTitle.Padding(0, 1)
	d.Styles.NormalDesc = d.Styles.NormalDesc.Padding(0, 1)
	if w < 40 {
		w = 40
	}
	if h < 8 {
		h = 8
	}
	l := list.New(items, d, w, h)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.SetShowHelp(false)
	return l
}

func formVpnAdd() {
	var name, note string
	var ok bool
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("VPN user name").Description("letters, digits, _ -").Value(&name),
			huh.NewInput().Title("Note (optional)").Value(&note),
			huh.NewConfirm().Title("Create user?").Value(&ok),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		fmt.Println(subStyle.Render("cancelled"))
		return
	}
	if err := vpn.AddUser(name, note); err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(okStyle.Render("user created"))
	fmt.Println(subStyle.Render("QR file: " + vpn.QRPath(name)))
	if link, err := vpn.SubscriptionLink(name); err == nil && link != "" {
		fmt.Println(subStyle.Render("VLESS QR (terminal):"))
		_ = exec.Command("qrencode", "-t", "ANSIUTF8", link).Run()
	}
}

func formVpnRename() {
	var old, newN string
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Current name").Value(&old),
			huh.NewInput().Title("New name").Value(&newN),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || old == "" || newN == "" {
		return
	}
	out, err := vpn.RenameUser(old, newN)
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(okStyle.Render("renamed → " + out + " (UUID unchanged, links still work)"))
}

func formVpnSub() {
	var name string
	f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("VPN user").Value(&name))).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || name == "" {
		return
	}
	link, err := vpn.SubscriptionLink(name)
	if err != nil || link == "" {
		fmt.Println(errStyle.Render("no subscription for " + name))
		return
	}
	fmt.Println(link)
}

func formBackupPeer() {
	fmt.Println(okStyle.Render(install.BackupPeerStatus()))
}

func formSession() {
	var note string
	var hours int
	hours = 24
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Note").Value(&note),
			huh.NewInput().Title("Hours").Value(new(string)),
		),
	).WithTheme(huh.ThemeCharm())
	_ = f.Run()
	fmt.Println(subStyle.Render("use: netductor session create"))
}

func formConfirm() {}

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
	f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("go build ./cmd/netductor ?").Value(&ok))).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		return
	}
	cmd := exec.Command("go", "build", "-o", "netductor", "./cmd/netductor")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	_ = cmd.Run()
}
