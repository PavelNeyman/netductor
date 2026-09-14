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
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginLeft(1)
	subStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginLeft(1)
	docStyle   = lipgloss.NewStyle().Margin(1, 2)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
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
}

func modeItems(sug runMode) []list.Item {
	order := []runMode{modeVPS, modeOpenWRT, modeWorkstation, modeOperator}
	items := make([]list.Item, 0, 4)
	for _, m := range order {
		t, d := describeMode(m)
		if m == sug {
			t += "  ← suggested"
		}
		items = append(items, menuItem{title: t, desc: d, id: string(m)})
	}
	return items
}

func menuItemsFor(mode runMode) []list.Item {
	switch mode {
	case modeVPS:
		return []list.Item{
			menuItem{"Full install / upgrade", "confirm → bash install.sh", "install"},
			menuItem{"Prepare only", "confirm → install.sh --prepare", "prepare"},
			menuItem{"Doctor", "health checks", "doctor"},
			menuItem{"Status", "systemd units", "status"},
			menuItem{"Operator tools…", "VPN, edge, probes", "to-operator"},
			menuItem{"Set hostname", "nd-<role>-<marker> e.g. nd-core-nl01", "hostname"},
			menuItem{"Nodes registry", "list fleet", "nodes-list"},
			menuItem{"Change mode…", "", "change-mode"},
			menuItem{"Quit", "", "quit"},
		}
	case modeOpenWRT:
		return []list.Item{
			menuItem{"Agent install instructions", "outbound netductor-agent", "agent-help"},
			menuItem{"Run install-openwrt.sh", "confirm if present", "owrt-install"},
			menuItem{"Check agent config", "", "agent-cfg"},
			menuItem{"Change mode…", "", "change-mode"},
			menuItem{"Quit", "", "quit"},
		}
	case modeWorkstation:
		return []list.Item{
			menuItem{"Bootstrap one-liner", "copy-paste for VPS", "bootstrap"},
			menuItem{"Build netductor (local Go)", "", "build"},
			menuItem{"Site setup wizard", "MikroTik + RPi · step by step", "site-wizard"},
			menuItem{"MikroTik manage", "identity / routes / push", "mt-manage"},
			menuItem{"SSH known hosts", "TOFU list / forget", "ssh-hosts"},
			menuItem{"Sites list", "", "sites-list"},
			menuItem{"Operator tools…", "via SSH tunnel", "to-operator"},
			menuItem{"Change mode…", "", "change-mode"},
			menuItem{"Quit", "", "quit"},
		}
	default:
		return []list.Item{
			menuItem{"Status", "systemd units", "status"},
			menuItem{"Doctor", "health checks", "doctor"},
			menuItem{"VPN — list users", "", "vpn-list"},
			menuItem{"VPN — add user", "form: name + note", "vpn-add"},
			menuItem{"VPN — rename user", "old → new (UUID kept)", "vpn-rename"},
			menuItem{"VPN — subscription", "show subscription body", "vpn-sub"},
			menuItem{"Backup peer", "cross-VPS scp target", "backup-peer"},
			menuItem{"Backup now", "local + offsite if set", "backup-now"},
			menuItem{"Session token", "hours form", "session"},
			menuItem{"Edge — list devices", "", "edge-list"},
			menuItem{"Nodes registry", "core + relay fleet", "nodes-list"},
			menuItem{"Relay status", "online / metrics / last cmd", "relay-status"},
			menuItem{"Relay sync", "push user list to relays", "relay-sync"},
			menuItem{"RU exit ON", "core traffic via RU", "relay-exit-on"},
			menuItem{"RU exit OFF", "", "relay-exit-off"},
			menuItem{"Set hostname", "nd-<role>-<marker> e.g. nd-core-nl01", "hostname"},
			menuItem{"Addons — Lampac", "status / health", "addons-lampac"},
			menuItem{"Live probes", "", "probe"},
			menuItem{"Collect metrics", "", "collect"},
			menuItem{"Change mode…", "", "change-mode"},
			menuItem{"Quit", "", "quit"},
		}
	}
}

func newList(title string, items []list.Item, w, h int) list.Model {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(lipgloss.Color("205")).BorderForeground(lipgloss.Color("205"))
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(lipgloss.Color("218"))
	l := list.New(items, d, w, h)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	return l
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-8)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
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
				m.list = newList("Select mode", modeItems(sug), m.width-4, m.height-8)
				m.screen = screenMode
				return m, nil
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
				t, _ := describeMode(m.mode)
				m.list = newList("Netductor · "+t, menuItemsFor(m.mode), m.width-4, m.height-8)
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
		m.list = newList("Select mode", modeItems(sug), m.width-4, m.height-8)
		m.screen = screenMode
		return m, nil
	case "to-operator":
		m.mode = modeOperator
		t, _ := describeMode(m.mode)
		m.list = newList("Netductor · "+t, menuItemsFor(m.mode), m.width-4, m.height-8)
		return m, nil
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
	if m.quitting && m.result.action == "quit" {
		return subStyle.Render("bye") + "\n"
	}
	if m.screen == screenOutput {
		body := lipgloss.NewStyle().Width(max(20, m.width-4)).Render(m.output)
		return docStyle.Render(
			titleStyle.Render("Output") + "\n\n" + body + "\n\n" +
				helpStyle.Render("enter/esc back · ctrl+c quit"),
		)
	}
	header := ""
	if m.screen == screenMode {
		sug, why := detectSuggestedMode()
		st, _ := describeMode(sug)
		header = subStyle.Render(fmt.Sprintf("Suggested: %s (%s)", st, why)) + "\n" +
			subStyle.Render("↑↓ select · enter · nothing installs until you confirm") + "\n\n"
	}
	return docStyle.Render(header + m.list.View())
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
	w, h := 80, 24
	sug, _ := detectSuggestedMode()
	m := model{width: w, height: h, mode: mode}
	if startMenu && mode != "" {
		m.screen = screenMenu
		t, _ := describeMode(mode)
		m.list = newList("Netductor · "+t, menuItemsFor(mode), w-4, h-8)
	} else {
		m.screen = screenMode
		m.list = newList("Select mode", modeItems(sug), w-4, h-8)
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
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
		default:
			return
		}
	}
}
