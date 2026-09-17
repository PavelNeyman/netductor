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
	"github.com/PavelNeyman/netductor/internal/audit"
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
	return describeModeLang(m, detectTUILang())
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
		return "Operator panel", "Day-2 ops: users, sessions, edge, doctor, probes"
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

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type screen int

const (
	screenMode screen = iota
	screenMenu
	screenOutput
)

// result after tea.Quit — forms run outside alt-screen
type tuiResult struct {
	action string
	mode   runMode
	output string
}

type model struct {
	screen   screen
	mode     runMode
	list     list.Model
	output   string
	result   tuiResult
	width    int
	height   int
	quitting bool
	lang     tuiLang
	helpY    int // first row of help bar (for mouse)
}

func modeItems(sug runMode, lang tuiLang) []list.Item {
	loc := l10n(lang)
	order := []runMode{modeVPS, modeOpenWRT, modeWorkstation, modeOperator}
	items := make([]list.Item, 0, 4)
	for _, m := range order {
		title, d := describeModeLang(m, lang)
		if m == sug {
			title += loc.SuggestedMark
		}
		items = append(items, menuItem{title: title, desc: d, id: string(m)})
	}
	return items
}

func menuItemsFor(mode runMode, lang tuiLang) []list.Item {
	loc := l10n(lang)
	wizDesc := "Primary / Secondary / OpenWrt / MikroTik — questions then auto"
	if lang == langRU {
		wizDesc = "Primary / Secondary / OpenWrt / MikroTik — вопросы, затем авто"
	}
	items := []list.Item{
		menuItem{loc.Wizard, wizDesc, "wizard"},
	}
	switch mode {
	case modeVPS:
		items = append(items,
			menuItem{"Install / upgrade stack", "netductor install (idempotent)", "install"},
			menuItem{"Apply Lampac", "on preferred fleet node (usually secondary)", "apply-lampac"},
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
		menuItem{loc.Tools, toolsDesc, "noop"},
		menuItem{"Doctor", "health checks", "doctor"},
		menuItem{"Status", "systemd units", "status"},
		menuItem{"Fleet status", "primary / secondary", "fleet-status"},
		menuItem{"Nodes registry", "list", "nodes-list"},
		menuItem{"Secondary / relay status", "agent online", "relay-status"},
		menuItem{"Backup now", "encrypted + peer", "backup-now"},
		menuItem{"Fleet sync", "data → secondary", "fleet-sync"},
		menuItem{"VPN users", "list", "vpn-list"},
		menuItem{"Refresh VPN links", "prefer secondary", "vpn-refresh"},
		menuItem{"Probes", "connectivity", "probe"},
		menuItem{"SSH known hosts", "TOFU", "ssh-hosts"},
		menuItem{"Audit tail", "events", "audit-tail"},
		menuItem{loc.ChangeMode, "", "change-mode"},
		menuItem{loc.Quit, "", "quit"},
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
	l.SetShowHelp(false) // custom help bar
	return l
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// Full terminal: list fills almost everything; help ~1–2 rows
		listH := msg.Height - 8
		if listH < 6 {
			listH = 6
		}
		m.list.SetSize(max(20, msg.Width-2), listH)
		m.helpY = msg.Height - 2
		return m, nil
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		// Click on list rows: approximate — select index by Y relative to list
		if m.screen != screenOutput && msg.Button == tea.MouseButtonLeft {
			// list content starts ~3–5 rows down after header
			rel := msg.Y - 4
			if rel >= 0 {
				idx := rel / 3 // delegate height 2 + spacing 1
				if idx >= 0 && idx < len(m.list.Items()) {
					m.list.Select(idx)
				}
			}
			// double-click-ish: second press on same — user can Enter
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			m.result = tuiResult{action: "quit", mode: m.mode}
			return m, tea.Quit
		case "l", "L", "ctrl+l":
			if m.lang == langRU {
				m.lang = langEN
			} else {
				m.lang = langRU
			}
			loc := l10n(m.lang)
			if m.screen == screenMode {
				sug, _ := detectSuggestedMode()
				m.list = newList(loc.SelectMode, modeItems(sug, m.lang), max(20, m.width-2), max(6, m.height-8))
			} else if m.screen == screenMenu {
				title, _ := describeModeLang(m.mode, m.lang)
				m.list = newList(loc.AppTitle+" · "+title, menuItemsFor(m.mode, m.lang), max(20, m.width-2), max(6, m.height-8))
			}
			return m, nil
		case "q":
			if m.screen == screenOutput {
				m.screen = screenMenu
				m.output = ""
				return m, nil
			}
			if m.screen == screenMenu {
				sug, _ := detectSuggestedMode()
				loc := l10n(m.lang)
				m.list = newList(loc.SelectMode, modeItems(sug, m.lang), max(20, m.width-2), max(6, m.height-8))
				m.screen = screenMode
				return m, nil
			}
			m.quitting = true
			m.result = tuiResult{action: "quit", mode: m.mode}
			return m, tea.Quit
		case "esc":
			if m.screen == screenOutput {
				m.screen = screenMenu
				m.output = ""
				return m, nil
			}
			if m.screen == screenMenu {
				sug, _ := detectSuggestedMode()
				loc := l10n(m.lang)
				m.list = newList(loc.SelectMode, modeItems(sug, m.lang), max(20, m.width-2), max(6, m.height-8))
				m.screen = screenMode
				return m, nil
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if m.screen == screenOutput {
				break
			}
			n := int(msg.String()[0] - '1')
			if n >= 0 && n < len(m.list.Items()) {
				m.list.Select(n)
				it, ok := m.list.SelectedItem().(menuItem)
				if ok {
					if m.screen == screenMode {
						m.mode = runMode(it.id)
						loc := l10n(m.lang)
						title, _ := describeModeLang(m.mode, m.lang)
						m.list = newList(loc.AppTitle+" · "+title, menuItemsFor(m.mode, m.lang), max(20, m.width-2), max(6, m.height-8))
						m.screen = screenMenu
						return m, nil
					}
					return m.handleAction(it.id)
				}
			}
		case "enter":
			if m.screen == screenOutput {
				m.screen = screenMenu
				m.output = ""
				return m, nil
			}
			it, ok := m.list.SelectedItem().(menuItem)
			if !ok {
				return m, nil
			}
			if m.screen == screenMode {
				m.mode = runMode(it.id)
				loc := l10n(m.lang)
				title, _ := describeModeLang(m.mode, m.lang)
				m.list = newList(loc.AppTitle+" · "+title, menuItemsFor(m.mode, m.lang), max(20, m.width-2), max(6, m.height-8))
				m.screen = screenMenu
				return m, nil
			}
			return m.handleAction(it.id)
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) handleAction(id string) (tea.Model, tea.Cmd) {
	// actions that need Huh forms → leave TUI
	switch id {
	case "quit":
		m.quitting = true
		m.result = tuiResult{action: "quit", mode: m.mode}
		return m, tea.Quit
	case "vpn-add", "vpn-rename", "vpn-sub", "backup-peer", "session", "install", "prepare", "owrt-install", "build", "hostname", "site-wizard", "mt-manage", "sites-list", "sni-live", "ssh-hosts":
		m.result = tuiResult{action: id, mode: m.mode}
		return m, tea.Quit
	case "change-mode":
		sug, _ := detectSuggestedMode()
		m.list = newList("Select mode", modeItems(sug, m.lang), m.width-4, m.height-8)
		m.screen = screenMode
		return m, nil
	case "to-operator":
		m.mode = modeOperator
		t, _ := describeModeLang(m.mode, m.lang)
		m.list = newList("Netductor · "+t, menuItemsFor(m.mode, m.lang), m.width-4, m.height-8)
		return m, nil
	case "noop":
		return m, nil
	case "wizard":
		m.result = tuiResult{action: "wizard", mode: m.mode}
		return m, tea.Quit
	case "fleet-status":
		m.output = capture(func() {
			out, _ := exec.Command("netductor", "fleet", "status").CombinedOutput()
			fmt.Print(string(out))
		})
		m.screen = screenOutput
	case "fleet-sync":
		m.output = capture(func() {
			out, _ := exec.Command("netductor", "fleet", "sync").CombinedOutput()
			fmt.Print(string(out))
		})
		m.screen = screenOutput
	case "apply-lampac":
		m.output = capture(func() {
			out, _ := exec.Command("netductor", "fleet", "apply-lampac").CombinedOutput()
			fmt.Print(string(out))
		})
		m.screen = screenOutput
	case "status":
		m.output = capture(func() { runStatus() })
		m.screen = screenOutput
	case "doctor":
		m.output = capture(func() { _ = runDoctorNative() })
		m.screen = screenOutput
	case "addons-lampac":
		out, _ := exec.Command("netductor", "addons", "lampac").CombinedOutput()
		return m, tea.Printf("%s", string(out))
	case "probe":
		m.output = capture(func() { runProbe(nil) })
		m.screen = screenOutput
	case "collect":
		m.output = capture(func() { _ = runCollect() })
		m.screen = screenOutput
	case "edge-list":
		m.output = capture(runEdgeList)
		m.screen = screenOutput
	case "nodes-list":
		m.output = capture(func() {
			runNodes([]string{"list"})
		})
		m.screen = screenOutput
	case "relay-status":
		m.output = capture(func() { runRelay([]string{"status"}) })
		m.screen = screenOutput
	case "relay-sync":
		m.output = capture(func() { runRelay([]string{"sync"}) })
		m.screen = screenOutput
	case "relay-exit-on":
		m.output = capture(func() { runRelay([]string{"exit", "on"}) })
		m.screen = screenOutput
	case "relay-exit-off":
		m.output = capture(func() { runRelay([]string{"exit", "off"}) })
		m.screen = screenOutput
	case "backup-now":
		m.output = capture(func() { runBackupCmd(nil) })
		m.screen = screenOutput
	case "vpn-refresh":
		m.output = capture(func() { runVPN([]string{"refresh-links"}) })
		m.screen = screenOutput
	case "audit-tail":
		m.output = capture(func() {
			for _, ev := range audit.Tail(40) {
				fmt.Printf("%d\t%s\t%s\t%s\t%s\n", ev.TS, ev.Actor, ev.Action, ev.Target, ev.Detail)
			}
		})
		m.screen = screenOutput
	case "sessions-list":
		m.output = capture(func() { runVPN([]string{"session", "list"}) })
		m.screen = screenOutput
	case "vpn-list":
		m.output = capture(func() {
			users, err := vpn.List()
			if err != nil {
				fmt.Println(err)
				return
			}
			if len(users) == 0 {
				fmt.Println("(no users)")
			}
			for _, u := range users {
				en := "off"
				if u.Enabled {
					en = "on"
				}
				fmt.Printf("%s\t%s\t%s\t%s\n", u.Name, en, u.UUID, u.Note)
			}
		})
		m.screen = screenOutput
	case "bootstrap":
		m.output = `curl -fsSL -o /usr/local/bin/netductor \
  https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64
chmod 755 /usr/local/bin/netductor
netductor tui --mode vps`
		m.screen = screenOutput
	case "agent-help":
		m.output = `mkdir -p /etc/netductor-agent
# SERVER= TOKEN= DEVICE_ID= INTERVAL=60 → /etc/netductor-agent/config
# binary: netductor-agent-linux-arm64|mipsle|arm from GitHub releases
# docs: edge/openwrt/INSTALL.md`
		m.screen = screenOutput
	case "agent-cfg":
		m.output = capture(func() {
			for _, p := range []string{"/etc/netductor-agent/config"} {
				if _, err := os.Stat(p); err == nil {
					fmt.Println("found:", p)
				}
			}
		})
		m.screen = screenOutput
	}
	return m, nil
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

func (m model) View() string {
	loc := l10n(m.lang)
	w, h := m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	if m.quitting && m.result.action == "quit" {
		return subStyle.Render("bye") + "\n"
	}

	helpChips := []string{}
	if m.screen == screenOutput {
		helpChips = []string{"↵ " + loc.Back, "q " + loc.Quit, "L lang"}
	} else {
		helpChips = []string{"↑↓", "↵ open", "1–9", "q back", "L lang", "^C quit"}
	}
	var chips strings.Builder
	for _, c := range helpChips {
		chips.WriteString(chipStyle.Render(c))
	}
	helpLine := helpStyle.Width(w).Render(chips.String())
	helpH := lipgloss.Height(helpLine)
	if helpH < 1 {
		helpH = 1
	}
	// pin help to bottom
	_ = helpH

	if m.screen == screenOutput {
		header := titleStyle.Width(w).Render(loc.Output)
		bodyH := h - lipgloss.Height(header) - helpH - 2
		if bodyH < 3 {
			bodyH = 3
		}
		body := lipgloss.NewStyle().Width(w - 2).Height(bodyH).MaxHeight(bodyH).Render(m.output)
		gap := h - lipgloss.Height(header) - lipgloss.Height(body) - helpH
		if gap < 0 {
			gap = 0
		}
		return header + "\n" + body + strings.Repeat("\n", gap) + helpLine
	}

	langBadge := "EN"
	if m.lang == langRU {
		langBadge = "RU"
	}
	header := titleStyle.Width(w).Render(loc.AppTitle + "  ·  " + langBadge)
	if m.screen == screenMode {
		sug, why := detectSuggestedMode()
		st, _ := describeModeLang(sug, m.lang)
		header += "\n" + subStyle.Width(w).Render(fmt.Sprintf("%s: %s (%s)", loc.Suggested, st, why))
		header += "\n" + subStyle.Width(w).Render(loc.HelpMode)
	}
	listH := h - lipgloss.Height(header) - helpH - 1
	if listH < 5 {
		listH = 5
	}
	// list already sized in Update WindowSize
	mid := m.list.View()
	used := lipgloss.Height(header) + lipgloss.Height(mid) + helpH
	gap := h - used
	if gap < 0 {
		gap = 0
	}
	return header + "\n" + mid + strings.Repeat("\n", gap) + helpLine
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── Huh forms (outside alt-screen) ─────────────────────────

func formVpnAdd() {
	var name, note string
	var ok bool
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("VPN user name").Description("letters, digits, _ -").Value(&name).Validate(func(s string) error {
				if !vpn.ValidName(s) {
					return fmt.Errorf("invalid name")
				}
				return nil
			}),
			huh.NewInput().Title("Note (optional)").Value(&note),
			huh.NewConfirm().Title("Create user and apply sing-box?").Affirmative("Yes").Negative("Cancel").Value(&ok),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		fmt.Println(subStyle.Render("cancelled"))
		return
	}
	out, err := vpn.Add(name, note)
	fmt.Println(out)
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
	} else {
		fmt.Println(okStyle.Render("user created"))
		fmt.Println(subStyle.Render("QR file: "+vpn.QRPath(name)))
		if s, ok := vpn.ReadClient(name, "link-vless.txt", "link.txt"); ok {
			fmt.Println(subStyle.Render("VLESS QR (terminal):"))
			vpn.PrintASCIIQR(s)
		}
	}
}

func formVpnRename() {
	var oldName, newName string
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Current name").Value(&oldName),
			huh.NewInput().Title("New name").Value(&newName),
		),
	)
	if err := f.Run(); err != nil {
		return
	}
	out, err := vpn.Rename(strings.TrimSpace(oldName), strings.TrimSpace(newName))
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(okStyle.Render("renamed → " + out + " (UUID unchanged, links still work)"))
}

func formVpnSub() {
	var name string
	f := huh.NewForm(huh.NewGroup(huh.NewInput().Title("User name").Value(&name)))
	if err := f.Run(); err != nil {
		return
	}
	name = strings.TrimSpace(name)
	sub, ok := vpn.ReadClient(name, "subscription.txt", "link.txt")
	if !ok {
		fmt.Println(errStyle.Render("no subscription for " + name))
		return
	}
	fmt.Println(sub)
}

func formBackupPeer() {
	var target string
	f := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("SCP target").Placeholder("root@RELAY:/var/lib/netductor/backups/peers/core/").Value(&target),
	))
	if err := f.Run(); err != nil {
		return
	}
	if err := install.SetBackupPeer(strings.TrimSpace(target), ""); err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(okStyle.Render(install.BackupPeerStatus()))
}


func formSession() {
	var hoursStr = "72"
	var ok bool
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Session lifetime (hours)").Value(&hoursStr),
			huh.NewConfirm().Title("Generate admin session token?").Affirmative("Yes").Negative("No").Value(&ok),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		fmt.Println(subStyle.Render("cancelled"))
		return
	}
	hours := 72
	fmt.Sscanf(hoursStr, "%d", &hours)
	tok, exp, err := vpn.CreateSession(hours)
	if err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	fmt.Println(okStyle.Render(tok))
	fmt.Printf("expires_unix=%d hours=%d\n", exp, hours)
}

func formConfirm(title, desc string) bool {
	var ok bool
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(title).Description(desc),
			huh.NewConfirm().Title("Proceed?").Affirmative("Yes").Negative("No").Value(&ok),
		),
	).WithTheme(huh.ThemeCharm())
	_ = f.Run()
	return ok
}

func formInstall(prepare bool) {
	title := "Full install / upgrade"
	cmdHint := "bash install.sh"
	if prepare {
		title = "Prepare only"
		cmdHint = "bash install.sh --prepare"
	}
	if !formConfirm(title, "Runs on this host as current user.\nCommand: "+cmdHint+"\nRequires root.") {
		fmt.Println(subStyle.Render("cancelled"))
		return
	}
	if prepare {
		runInstall([]string{"--prepare"})
	} else {
		runInstall(nil)
	}
}

func formOwrtInstall() {
	if !formConfirm("OpenWrt installer", "Looks for install-openwrt.sh and runs it.") {
		fmt.Println(subStyle.Render("cancelled"))
		return
	}
	for _, c := range []string{"install-openwrt.sh", "/opt/netductor/install-openwrt.sh"} {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			cmd := exec.Command("sh", c)
			cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
			_ = cmd.Run()
			return
		}
	}
	fmt.Println(errStyle.Render("install-openwrt.sh not found"))
}

func formBuild() {
	if !formConfirm("Local Go build", "go build -o netductor ./cmd/netductor") {
		fmt.Println(subStyle.Render("cancelled"))
		return
	}
	cmd := exec.Command("go", "build", "-o", "netductor", "./cmd/netductor")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(errStyle.Render(err.Error()))
	} else {
		fmt.Println(okStyle.Render("ok: ./netductor"))
	}
}

func runBubbleSession(mode runMode, startMenu bool) tuiResult {
	w, h := 120, 40
	lang := detectTUILang()
	loc := l10n(lang)
	sug, _ := detectSuggestedMode()
	m := model{width: w, height: h, mode: mode, lang: lang}
	if startMenu && mode != "" {
		m.screen = screenMenu
		title, _ := describeModeLang(mode, lang)
		m.list = newList(loc.AppTitle+" · "+title, menuItemsFor(mode, lang), w-2, h-10)
	} else {
		m.screen = screenMode
		m.list = newList(loc.SelectMode, modeItems(sug, lang), w-2, h-10)
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	final, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return tuiResult{action: "quit"}
	}
	fm, ok := final.(model)
	if !ok {
		return tuiResult{action: "quit"}
	}
	return fm.result
}

// runTUI — Bubble Tea menus + Huh forms for confirm/add/session.

func runHostnameForm() {
	cur := ""
	if b, err := os.ReadFile("/etc/netductor/node_id"); err == nil {
		cur = strings.TrimSpace(string(b))
	}
	name := cur
	ok := false
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Hostname").Description("Format nd-<role>-<marker> · core|edge|lab · e.g. nd-core-nl01 · a-z0-9- only").Value(&name),
			huh.NewConfirm().Title("Apply hostname on this host?").Affirmative("Yes").Negative("Skip").Value(&ok),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil || !ok {
		return
	}
	name = nodes.NormalizeHostname(name)
	if name == "" {
		fmt.Println("empty name, skip")
		return
	}
	_ = os.MkdirAll("/etc/netductor", 0o755)
	_ = os.WriteFile("/etc/netductor/node_id", []byte(name+string([]byte{10})), 0o644)
	_ = os.WriteFile("/etc/hostname", []byte(name+string([]byte{10})), 0o644)
	_ = exec.Command("hostnamectl", "set-hostname", name).Run()
	ip := ""
	if b, err := os.ReadFile("/etc/netductor/public_ip"); err == nil {
		ip = strings.TrimSpace(string(b))
	}
	_ = nodes.SelfRegisterLocal(name, "core", ip)
	fmt.Println("hostname:", name)
}

func runTUI(args []string) {
	forced := parseModeFlags(args)
	mode := forced
	startMenu := forced == modeVPS || forced == modeOpenWRT || forced == modeWorkstation || forced == modeOperator

	for {
		res := runBubbleSession(mode, startMenu)
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

// runSetupWizard asks what to configure, then runs automation.
func runSetupWizard() {
	var target string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Setup wizard — what are we configuring?").
				Description("Questions first, then fully automatic apply").
				Options(
					huh.NewOption("Primary VPS (abroad control plane)", "primary"),
					huh.NewOption("Secondary VPS (RU entry / warm services)", "secondary"),
					huh.NewOption("OpenWrt router / RPi (edge agent)", "openwrt"),
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
	case "mikrotik":
		runSiteWizard()
	}
}

func wizardPrimary() {
	var sni string
	var doInstall bool
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Run netductor install on this host?").
				Description("Idempotent. VPN, DNS, API, bot, SSH key-only.").
				Value(&doInstall),
			huh.NewInput().
				Title("Reality SNI").
				Placeholder("api.vk.me").
				Value(&sni),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if sni == "" {
		sni = "api.vk.me"
	}
	if doInstall {
		fmt.Println(okStyle.Render("→ netductor install"))
		cmd := exec.Command("netductor", "install")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		_ = cmd.Run()
	}
	fmt.Println(okStyle.Render("→ set SNI " + sni))
	_ = exec.Command("netductor", "vpn", "set-sni", sni).Run()
	_ = exec.Command("netductor", "fleet", "bootstrap").Run()
	out, _ := exec.Command("netductor", "doctor").CombinedOutput()
	fmt.Print(string(out))
	fmt.Println(subStyle.Render("Next: wizard → Secondary"))
}

func wizardSecondary() {
	var host, user, pass, sni string
	user = "root"
	sni = "api.vk.me"
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Secondary IP / host").Value(&host),
			huh.NewInput().Title("SSH user").Value(&user),
			huh.NewInput().Title("SSH password (first login only)").EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewInput().Title("Reality SNI").Placeholder("api.vk.me").Value(&sni),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if err != nil || host == "" || pass == "" {
		fmt.Println(errStyle.Render("host and password required"))
		return
	}
	if sni == "" {
		sni = "api.vk.me"
	}
	fmt.Println(okStyle.Render("→ fleet provision-secondary (automatic)"))
	cmd := exec.Command("netductor", "fleet", "provision-secondary",
		"--host", host, "--user", user, "--password", pass, "--sni", sni)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(errStyle.Render(err.Error()))
		return
	}
	out, _ := exec.Command("netductor", "fleet", "status").CombinedOutput()
	fmt.Print(string(out))
}

func wizardOpenWrt() {
	var host, user, pass string
	user = "root"
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("OpenWrt host / IP").Value(&host),
			huh.NewInput().Title("SSH user").Value(&user),
			huh.NewInput().Title("SSH password (optional if key)").EchoMode(huh.EchoModePassword).Value(&pass),
			huh.NewNote().Title("Note").Description("Agent enrolls outbound to primary :8788. Approve in Admin/TG."),
		),
	).WithTheme(huh.ThemeCharm()).Run()
	if host == "" {
		fmt.Println(errStyle.Render("host required"))
		return
	}
	fmt.Println(okStyle.Render("→ edge enroll path for " + host))
	fmt.Println(subStyle.Render("On router: install netductor-agent; CORE=http://PRIMARY:8788"))
	fmt.Println(subStyle.Render("Bootstrap token: /etc/netductor/secrets/edge_bootstrap_token on primary"))
}
