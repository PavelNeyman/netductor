package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func menuItemsFor(mode runMode, lang tuiLang) []list.Item {
	items := []list.Item{
		menuItem{"wizard", TT(lang, "Primary / Secondary / OpenWrt / MikroTik — questions then auto",
			"Primary / Secondary / OpenWrt / MikroTik — вопросы, затем авто"), "wizard"},
	}
	switch mode {
	case modeVPS:
		items = append(items,
			menuItem{TT(lang, "Install / upgrade stack", "Install / upgrade стека"), "netductor install (idempotent)", "install"},
			menuItem{TT(lang, "Add-ons checklist", "Дополнения (чек-лист)"), TT(lang, "Lampac / TG / go2rtc", "Lampac / TG / go2rtc"), "apply-lampac"},
			menuItem{TT(lang, "Set hostname", "Задать hostname"), "nd-primary / nd-secondary / …", "hostname"},
		)
	case modeOpenWRT:
		items = append(items,
			menuItem{TT(lang, "Agent install instructions", "Инструкция agent"), "outbound netductor-agent", "agent-help"},
			menuItem{TT(lang, "Run install-openwrt.sh", "Запуск install-openwrt.sh"), TT(lang, "if present on device", "если есть на устройстве"), "owrt-install"},
			menuItem{TT(lang, "Check agent config", "Проверить конфиг agent"), "", "agent-cfg"},
		)
	case modeWorkstation:
		items = append(items,
			menuItem{TT(lang, "Bootstrap one-liner", "Bootstrap one-liner"), TT(lang, "copy-paste for VPS", "copy-paste для VPS"), "bootstrap"},
			menuItem{TT(lang, "Build netductor (local Go)", "Сборка netductor (локальный Go)"), "", "build"},
			menuItem{TT(lang, "MikroTik manage", "Управление MikroTik"), TT(lang, "identity / routes / push", "identity / routes / push"), "mt-manage"},
		)
	default:
		items = append(items,
			menuItem{TT(lang, "VPN — add user", "VPN — добавить пользователя"), "", "vpn-add"},
			menuItem{TT(lang, "VPN — link / QR", "VPN — ссылка / QR"), "", "vpn-sub"},
			menuItem{TT(lang, "Session create", "Создать session"), "", "session"},
			menuItem{TT(lang, "Live SNI", "Live SNI"), "", "sni-live"},
			menuItem{TT(lang, "Edge devices", "Edge-устройства"), "", "edge-list"},
		)
	}
	toolsDesc := "diagnostics & ops"
	if lang == langRU {
		toolsDesc = "диагностика и операции"
	}
	items = append(items,
		menuItem{"tools", toolsDesc, "noop"},
		menuItem{TT(lang, "Doctor", "Doctor"), TT(lang, "health checks", "проверки"), "doctor"},
		menuItem{TT(lang, "Status", "Status"), TT(lang, "systemd units", "юниты systemd"), "status"},
		menuItem{TT(lang, "Fleet status", "Статус флота"), "primary / secondary", "fleet-status"},
		menuItem{TT(lang, "Nodes registry", "Реестр нод"), "list", "nodes-list"},
		menuItem{TT(lang, "Secondary status", "Secondary"), "agent online", "secondary-status"},
		menuItem{TT(lang, "Secondary sync", "Sync secondary"), "VPN users → secondary", "secondary-sync"},
		menuItem{TT(lang, "Edge devices", "Edge устройства"), "list", "edge-list"},
		menuItem{TT(lang, "Edge pending", "Edge ожидание"), "approve queue", "edge-pending"},
		menuItem{TT(lang, "Edge recovery code", "Код recovery"), "", "edge-recovery"},
		menuItem{TT(lang, "NVR status", "NVR статус"), "cameras / storage", "nvr-status"},
		menuItem{TT(lang, "NVR go2rtc", "NVR go2rtc"), "write config", "nvr-go2rtc"},
		menuItem{TT(lang, "Git repos", "Git репозитории"), "list", "git-repos"},
		menuItem{TT(lang, "Git pipelines", "Git pipelines"), "list", "git-pipelines"},
		menuItem{TT(lang, "Registry status", "Registry"), "", "registry-status"},
		menuItem{TT(lang, "Add-ons status", "Дополнения"), "lampac / …", "addons-status"},
		menuItem{TT(lang, "mTLS certs", "mTLS сертификаты"), "list", "mtls-list"},
		menuItem{TT(lang, "Backup now", "Бэкап сейчас"), "encrypted + peer", "backup-now"},
		menuItem{TT(lang, "VPN users", "VPN пользователи"), "list", "vpn-list"},
		menuItem{TT(lang, "Refresh VPN links", "Обновить ссылки VPN"), "", "vpn-refresh"},
		menuItem{TT(lang, "Probes", "Probes"), "connectivity", "probe"},
		menuItem{TT(lang, "SSH known hosts", "SSH known hosts"), "TOFU", "ssh-hosts"},
		menuItem{TT(lang, "Audit tail", "Audit"), "events", "audit-tail"},
		menuItem{TT(lang, "Sites list", "Сайты"), "", "sites-list"},
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
	f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(TT(detectLang(), "go build ./cmd/netductor ?", "go build ./cmd/netductor ?")).Value(&ok))).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		return
	}
	cmd := exec.Command("go", "build", "-o", "netductor", "./cmd/netductor")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	_ = cmd.Run()
}
