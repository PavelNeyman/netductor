package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

type runMode string

const (
	modeVPS         runMode = "vps"
	modeOpenWRT     runMode = "openwrt"
	modeWorkstation runMode = "workstation"
	modeOperator    runMode = "operator"
)

func detectSuggestedMode() (runMode, string) {
	if _, err := os.Stat("/etc/openwrt_release"); err == nil {
		return modeOpenWRT, "found /etc/openwrt_release"
	}
	if _, err := os.Stat("/etc/config/network"); err == nil {
		if _, err2 := os.Stat("/sbin/uci"); err2 == nil {
			return modeOpenWRT, "UCI + /etc/config/network"
		}
	}
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		if os.Geteuid() == 0 && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return modeVPS, "Debian, root, no GUI"
		}
		if os.Geteuid() == 0 && os.Getenv("SSH_CONNECTION") != "" {
			return modeVPS, "Debian root over SSH"
		}
	}
	if runtime.GOOS == "darwin" {
		return modeWorkstation, "macOS"
	}
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return modeWorkstation, "GUI session"
	}
	return modeOperator, "no strong signal"
}

func describeMode(m runMode) (string, string) {
	return describeModeLang(m, detectLang())
}

func describeModeLang(m runMode, lang tuiLang) (string, string) {
	if lang == langRU {
		switch m {
		case modeVPS:
			return "Настройка VPS", "Стек на сервере: VPN, DNS, API, бот…"
		case modeOpenWRT:
			return "OpenWrt / edge", "Агент на роутере, сеть сайта, тоннель к VPS"
		case modeWorkstation:
			return "Рабочая станция (PC/Mac)", "Сборка, bootstrap, удалённые хелперы"
		default:
			return "Панель оператора", "Пользователи, сессии, edge, doctor, probes"
		}
	}
	switch m {
	case modeVPS:
		return "VPS setup", "Install & configure stack on a server (VPN, DNS, API, bot…)"
	case modeOpenWRT:
		return "OpenWrt / edge", "Router agent, site network, tunnel toward your VPS"
	case modeWorkstation:
		return "Workstation (PC/Mac)", "Build binaries, bootstrap hints, remote helpers"
	default:
		return "Manage node", "Operate installed system: users, fleet, doctor, probes"
	}
}

func parseModeFlags(args []string) runMode {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--mode" && i+1 < len(args) {
			return runMode(args[i+1])
		}
		if strings.HasPrefix(a, "--mode=") {
			return runMode(strings.TrimPrefix(a, "--mode="))
		}
	}
	if v := os.Getenv("NETDUCTOR_MODE"); v != "" {
		return runMode(v)
	}
	return ""
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1)
	subStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	docStyle   = lipgloss.NewStyle().Padding(0, 1)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("236")).Padding(0, 1)
	chipStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("63")).Padding(0, 1).MarginRight(1)
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type menuItem struct {
	title, desc, id string
}

func parseRemoteFlags(args []string) (host, user, key, pass string) {
	user = "root"
	for i := 0; i < len(args); i++ {
		a := args[i]
		take := func() string {
			if i+1 < len(args) {
				i++
				return args[i]
			}
			return ""
		}
		switch {
		case a == "--remote" || a == "--host":
			host = take()
		case strings.HasPrefix(a, "--remote="):
			host = strings.TrimPrefix(a, "--remote=")
		case a == "--remote-user" || a == "--user":
			user = take()
		case strings.HasPrefix(a, "--remote-user="):
			user = strings.TrimPrefix(a, "--remote-user=")
		case a == "--remote-key" || a == "--key":
			key = take()
		case strings.HasPrefix(a, "--remote-key="):
			key = strings.TrimPrefix(a, "--remote-key=")
		case a == "--remote-password" || a == "--password":
			pass = take()
		case strings.HasPrefix(a, "--remote-password="):
			pass = strings.TrimPrefix(a, "--remote-password=")
		}
	}
	return host, user, key, pass
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type screen int

const (
	screenMode screen = iota
	screenMenu
	screenOutput
	screenWizard
)

const (
	tabWizard   = "wizard"
	tabTools    = "tools"
	tabOps      = "ops"
	tabSettings = "settings"
	tabMode     = "mode"
)

type tuiResult struct {
	action string
	mode   runMode
	output string
}

type model struct {
	screen   screen
	mode     runMode
	tab      string
	cursor   int
	output   string
	result   tuiResult
	width    int
	height   int
	quitting bool
	lang     tuiLang
	langPref string
	helpY    int
	hits     []hitRect
	list     list.Model
	wizStep     wizStep
	wizTarget   wizTarget
	wizFields   []wizField
	wizFieldIdx int
	wizInput    string
	wizMsg      string
	remoteHost     string
	remoteUser     string
	remoteKey      string
	remotePassword string
	formAction     string
}

func modeItems(sug runMode, lang tuiLang) []list.Item {
	order := []runMode{modeVPS, modeOpenWRT, modeWorkstation, modeOperator}
	items := make([]list.Item, 0, 4)
	for _, m := range order {
		title, d := describeModeLang(m, lang)
		if m == sug {
			title += " ←"
		}
		items = append(items, menuItem{title: title, desc: d, id: string(m)})
	}
	return items
}

func orDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func capture(fn func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		fn()
		return ""
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	fn()
	_ = w.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	var buf strings.Builder
	tmp := make([]byte, 4096)
	for {
		n, er := r.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if er != nil {
			break
		}
	}
	_ = r.Close()
	return buf.String()
}

func runTUI(args []string) {
	s := loadTUISettings()
	cliHost, cliUser, cliKey, cliPass := parseRemoteFlags(args)
	if cliHost != "" {
		s.RemoteHost = cliHost
	}
	if cliUser != "" {
		s.RemoteUser = cliUser
	}
	if cliKey != "" {
		s.RemoteKey = cliKey
	}
	if cliPass != "" {
		s.RemotePassword = cliPass
	}
	forced := parseModeFlags(args)
	mode := forced
	if mode == "" {
		mode, _ = detectSuggestedMode()
	}
	startMenu := forced == modeVPS || forced == modeOpenWRT || forced == modeWorkstation || forced == modeOperator
	for {
		res := runBubbleSession(mode, startMenu, s.RemoteHost, orDefault(s.RemoteUser, "root"), s.RemoteKey, s.RemotePassword)
		startMenu = true
		if res.mode != "" {
			mode = res.mode
		}
		switch res.action {
		case "", "quit":
			fmt.Println(subStyle.Render("bye"))
			return
		case "vpn-add":
			formVpnAdd()
		case "vpn-rename":
			formVpnRename()
		case "vpn-sub":
			formVpnSub()
		case "backup-peer":
			formBackupPeer()
		case "session":
			formSession()
		case "install":
			formInstall(false)
		case "prepare":
			formInstall(true)
		case "owrt-install":
			formOwrtInstall()
		case "build":
			formBuild()
		case "hostname":
			runHostnameForm()
		case "site-wizard":
			runSiteWizard()
		case "mt-manage":
			runMikroTikManage()
		case "sites-list":
			runSitesListTUI()
		case "sni-live":
			runSNIForm()
		case "ssh-hosts":
			runSSHHostsTUI()
		case "wizard":
			runSetupWizard()
		default:
			return
		}
	}
}
