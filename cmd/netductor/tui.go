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
	screenWizard
)

// top tabs (LAG-style)
const (
	tabWizard = "wizard"
	tabTools  = "tools"
	tabOps    = "ops"
	tabMode   = "mode"
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
	tab      string
	cursor   int
	output   string
	result   tuiResult
	width    int
	height   int
	quitting bool
	lang     tuiLang
	helpY    int
	hits     []hitRect
	list     list.Model // kept for compatibility; main UI is custom split
	// wizard state
	wizStep     wizStep
	wizTarget   wizTarget
	wizFields   []wizField
	wizFieldIdx int
	wizInput    string
	wizMsg      string
	remoteHost  string
	remoteUser  string
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
		menuItem{"Secondary / relay status", "agent online", "relay-status"},
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
	l.SetShowHelp(false) // custom help bar
	return l
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.screen == screenWizard {
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			m.width, m.height = ws.Width, ws.Height
			return m, nil
		}
		return m.updateWizard(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.helpY = msg.Height - 2
		return m, nil
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		return m.handleMouse(msg.X, msg.Y)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			m.result = tuiResult{action: "quit", mode: m.mode}
			return m, tea.Quit
		case "l", "L", "ctrl+l":
			return m.toggleLang()
		case "tab":
			order := []string{tabWizard, tabTools, tabOps, tabMode}
			for i, tname := range order {
				if tname == m.tab {
					m.tab = order[(i+1)%len(order)]
					break
				}
			}
			m.cursor = 0
			if m.tab == tabWizard {
				// stay in main menu UI listing wizard targets (not a separate app)
				m.screen = screenMenu
				m.wizStep = wizStepTarget
				return m, nil
			}
			if m.tab == tabMode {
				m.screen = screenMode
			} else {
				m.screen = screenMenu
			}
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.currentEntries())-1 {
				m.cursor++
			}
			return m, nil
		case "esc", "q":
			if m.screen == screenOutput {
				m.screen = screenMenu
				m.output = ""
				return m, nil
			}
			if m.tab != tabMode && m.screen == screenMenu {
				m.tab = tabMode
				m.screen = screenMode
				m.cursor = 0
				return m, nil
			}
			// mode screen: esc = quit confirm via quit action soft
			m.quitting = true
			m.result = tuiResult{action: "quit", mode: m.mode}
			return m, tea.Quit
		case "enter":
			if m.screen == screenOutput {
				m.screen = screenMenu
				m.output = ""
				return m, nil
			}
			return m.activateCursor()
		}
	}
	return m, nil
}

func (m model) toggleLang() (tea.Model, tea.Cmd) {
	if m.lang == langRU {
		m.lang = langEN
	} else {
		m.lang = langRU
	}
	return m, nil
}

func (m model) activateCursor() (tea.Model, tea.Cmd) {
	ents := m.currentEntries()
	if m.cursor < 0 || m.cursor >= len(ents) {
		return m, nil
	}
	id := ents[m.cursor].ID
	if m.screen == screenMode || m.tab == tabMode {
		m.mode = runMode(id)
		m.tab = tabTools
		m.screen = screenMenu
		m.cursor = 0
		return m, nil
	}
	if m.tab == tabWizard {
		// start in-TUI wizard for selected target
		m.wizTarget = wizTarget(id)
		m.wizBuildFields()
		m.wizStep = wizStepFields
		m.screen = screenWizard
		return m, nil
	}
	return m.handleAction(id)
}

func (m model) handleAction(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "quit":
		m.quitting = true
		m.result = tuiResult{action: "quit", mode: m.mode}
		return m, tea.Quit
	case "change-mode":
		m.tab = tabMode
		m.screen = screenMode
		m.cursor = 0
		return m, nil
	case "remote-set":
		m.wizTarget = "remote"
		m.wizFields = []wizField{
			{Key: "host", Label: map[bool]string{true: "VPS host / IP", false: "VPS host / IP"}[m.lang==langRU], Value: m.remoteHost, Placeholder: "2.27.118.70"},
			{Key: "user", Label: "SSH user", Value: orDefault(m.remoteUser, "root")},
		}
		m.wizFieldIdx = 0
		m.wizInput = m.wizFields[0].Value
		m.wizStep = wizStepFields
		m.screen = screenWizard
		return m, nil
	case "remote-clear":
		m.remoteHost, m.remoteUser = "", "root"
		m.output = "remote cleared — commands run locally"
		m.screen = screenOutput
		return m, nil
	case "fleet-status":
		m.showCmd("fleet", "status")
	case "relay-sync", "fleet-sync":
		m.showCmd("relay", "sync")
	case "apply-lampac":
		m.showCmd("install", "lampac")
	case "disable-legacy":
		m.showCmd("fleet", "disable-legacy")
	case "status":
		if m.hasRemote() {
			m.showCmd("status")
		} else {
			m.output = capture(func() { runStatus() })
			m.screen = screenOutput
		}
	case "doctor":
		if m.hasRemote() {
			m.showCmd("doctor")
		} else {
			m.output = capture(func() { _ = runDoctorNative() })
			m.screen = screenOutput
		}
	case "vpn-list":
		m.showCmd("vpn", "list")
	case "relay-status":
		m.showCmd("relay", "status")
	case "nodes-list":
		m.showCmd("nodes", "list")
	case "edge-list":
		m.showCmd("edge", "list")
	case "probe":
		if m.hasRemote() {
			m.showCmd("probe")
		} else {
			m.output = capture(func() { runProbe(nil) })
			m.screen = screenOutput
		}
	case "backup-now":
		m.showCmd("backup", "now")
	case "audit-tail":
		m.showCmd("audit", "tail")
	case "install":
		if m.hasRemote() {
			m.showCmd("install")
			return m, nil
		}
		m.result = tuiResult{action: id, mode: m.mode}
		return m, tea.Quit
	case "prepare", "build", "owrt-install", "vpn-add", "vpn-rename", "hostname", "site-wizard", "mt-manage", "sites-list", "sni-live", "ssh-hosts", "session", "backup-peer", "vpn-sub":
		m.result = tuiResult{action: id, mode: m.mode}
		return m, tea.Quit
	default:
		m.output = "unknown action: " + id
		m.screen = screenOutput
	}
	return m, nil
}

func orDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
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
			huh.NewConfirm().Title("Proceed? (Esc = cancel)").Affirmative("Yes").Negative("No / Esc").Value(&ok),
		),
	).WithTheme(huh.ThemeCharm())
	if err := f.Run(); err != nil {
		return false // Esc / interrupt
	}
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
	lang := detectLang()
	rh, ru := loadRemoteFromEnv()
	m := model{width: w, height: h, mode: mode, lang: lang, tab: tabMode, cursor: 0, remoteHost: rh, remoteUser: ru}
	if startMenu && mode != "" {
		m.screen = screenMenu
		m.tab = tabTools
	} else {
		m.screen = screenMode
		m.tab = tabMode
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
			// in-TUI wizard preferred; legacy fallback
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
					huh.NewOption("Secondary VPS (RU VPN entry only)", "secondary"),
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
