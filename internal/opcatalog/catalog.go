// Package opcatalog is the shared day-2 action registry for Web, TG, CLI, TUI.
// Deploy (fleet/primary/…) lives on Mac operator — not listed here.
package opcatalog

import "strings"

// Surfaces: web (Control), tg (Tools/catalog), cli (netductor …), tui (Ops remote).
const (
	SurfWeb = "web"
	SurfTG  = "tg"
	SurfCLI = "cli"
	SurfTUI = "tui"
)

// Action is one session-API operation exposed on operator surfaces.
type Action struct {
	ID       string   `json:"id"`
	Section  string   `json:"section"` // overview|vpn|nodes|edge|nvr|git|backup|dns|probes
	Method   string   `json:"method"`  // GET|POST
	Path     string   `json:"path"`
	Body     string   `json:"body,omitempty"`
	CLI      string   `json:"cli,omitempty"` // approximate CLI verb (documentation)
	LabelEN  string   `json:"label_en"`
	LabelRU  string   `json:"label_ru"`
	Surfaces []string `json:"surfaces"` // web, tg, cli, tui
}

func allSurfaces() []string { return []string{SurfWeb, SurfTG, SurfCLI, SurfTUI} }

// All returns the static catalog. New day-2 API → add here first, then UI.
func All() []Action {
	a := func(id, sec, method, path, body, cli, en, ru string, surfaces ...string) Action {
		if len(surfaces) == 0 {
			surfaces = allSurfaces()
		}
		return Action{ID: id, Section: sec, Method: method, Path: path, Body: body, CLI: cli, LabelEN: en, LabelRU: ru, Surfaces: surfaces}
	}
	return []Action{
		// overview
		a("health", "overview", "GET", "/health", "", "—", "Health", "Health", SurfWeb, SurfCLI),
		a("doctor", "overview", "GET", "/api/doctor", "", "doctor", "Doctor", "Doctor"),
		a("domain", "overview", "GET", "/api/domain", "", "domain show", "Domain", "Домен"),
		a("bot", "overview", "GET", "/api/bot-status", "", "—", "Bot status", "Статус бота"),
		a("status", "overview", "GET", "/api/status", "", "status", "Status", "Статус"),
		// metrics: Web/CLI/TUI; TG uses Status (no duplicate button)
		a("metrics", "overview", "GET", "/api/metrics", "", "—", "Metrics", "Метрики", SurfWeb, SurfCLI, SurfTUI),
		a("metrics-hist", "overview", "GET", "/api/metrics/history", "", "—", "Metrics history", "История метрик", SurfWeb, SurfCLI),
		a("addons", "overview", "GET", "/api/addons", "", "addons list", "Addons", "Дополнения"),
		a("addons-lampac", "overview", "GET", "/api/addons/lampac", "", "—", "Lampac", "Lampac"),
		a("sni", "overview", "GET", "/api/sni", "", "—", "SNI", "SNI"),
		a("sni-presets", "overview", "GET", "/api/sni-presets", "", "—", "SNI presets", "Пресеты SNI"),
		a("latest", "overview", "GET", "/api/latest", "", "—", "Latest metrics", "Latest"),
		a("sessions", "overview", "GET", "/api/sessions", "", "—", "Sessions", "Sessions"),

		// vpn
		a("vpn-users", "vpn", "GET", "/vpn/users", "", "vpn list", "List users", "Список users"),
		a("vpn-refresh", "vpn", "POST", "/api/vpn/refresh-links", "{}", "vpn refresh-links", "Refresh links", "Обновить ссылки"),

		// nodes
		a("nodes", "nodes", "GET", "/api/nodes", "", "nodes list", "Nodes", "Ноды"),
		a("nodes-self", "nodes", "GET", "/api/nodes/self", "", "—", "Self", "Self"),
		a("secondary", "nodes", "GET", "/api/secondary/status", "", "secondary status", "Secondary status", "Статус secondary"),
		a("secondary-links", "nodes", "GET", "/api/secondary/links", "", "—", "Secondary links", "Ссылки secondary"),
		a("ssh-hosts", "nodes", "GET", "/api/ssh-hosts", "", "ssh-hosts list", "SSH hosts", "SSH hosts"),
		a("ssh-clear", "nodes", "POST", "/api/ssh-hosts/clear", "{}", "ssh-hosts clear", "SSH hosts clear", "Очистить SSH hosts"),
		a("mtls-certs", "nodes", "GET", "/api/mtls/certs", "", "mtls list", "mTLS certs", "mTLS сертификаты"),
		a("sites", "nodes", "GET", "/api/sites", "", "—", "Sites", "Сайты"),

		// edge
		a("edge-pending", "edge", "GET", "/api/edge/pending", "", "edge pending", "Pending", "Pending"),
		a("edge-devices", "edge", "GET", "/api/edge/devices", "", "edge list", "Devices", "Devices"),
		a("edge-metrics", "edge", "GET", "/api/edge/metrics", "", "—", "Edge metrics", "Edge metrics"),
		a("edge-guest-st", "edge", "GET", "/api/edge/guest/status", "", "—", "Guest status", "Guest status"),

		// nvr
		a("nvr-cameras", "nvr", "GET", "/api/nvr/cameras", "", "nvr cameras", "Cameras", "Cameras"),
		a("nvr-config", "nvr", "GET", "/api/nvr/config", "", "nvr status", "Config", "Config"),
		a("nvr-storage", "nvr", "GET", "/api/nvr/storage", "", "—", "Storage", "Storage"),
		a("nvr-events", "nvr", "GET", "/api/nvr/events", "", "—", "Events", "Events"),
		a("nvr-segments", "nvr", "GET", "/api/nvr/segments", "", "—", "Segments", "Segments"),
		a("nvr-retention", "nvr", "POST", "/api/nvr/retention/run", "{}", "—", "Retention run", "Retention"),
		a("nvr-go2rtc", "nvr", "POST", "/api/nvr/go2rtc", "{}", "nvr go2rtc", "go2rtc write", "go2rtc"),

		// git
		a("git-repos", "git", "GET", "/api/git/repos", "", "git list", "Repos", "Repos"),
		a("git-pipelines", "git", "GET", "/api/git/pipelines", "", "git pipelines", "Pipelines", "Pipelines"),
		a("reg-status", "git", "GET", "/api/registry/status", "", "registry status", "Registry status", "Registry"),
		a("reg-ensure", "git", "POST", "/api/registry/ensure", "{}", "—", "Registry ensure", "Registry ensure"),

		// dns
		a("dns-lists", "dns", "GET", "/api/dns/lists", "", "dns list", "DNS lists", "DNS списки"),
		a("dns-set", "dns", "POST", "/api/dns/set", `{"id":"","enabled":true}`, "dns set", "DNS set list", "DNS вкл/выкл список"),
		a("dns-reload", "dns", "POST", "/api/dns/reload", "{}", "dns reload", "DNS reload", "DNS reload"),

		// backup
		a("backup-schedule", "backup", "GET", "/api/backup/schedule", "", "backup schedule", "Backup schedule", "Расписание бэкапа"),
		a("backup-list", "backup", "GET", "/api/backup/list", "", "backup list", "Backup files", "Файлы бэкапа"),
		a("backup-peer", "backup", "GET", "/api/backup/peer", "", "—", "Peer", "Peer"),
		a("backup-run", "backup", "POST", "/api/backup/run", "{}", "backup now", "Run now", "Бэкап сейчас"),
		a("sec-export", "backup", "GET", "/api/secondary/export", "", "—", "Secondary export", "Secondary export"),

		// probes / audit
		a("probes", "probes", "GET", "/api/probes", "", "probe", "Probes", "Probes"),
		a("probes-cfg", "probes", "GET", "/api/probes/config", "", "—", "Probes config", "Probes config"),
		a("audit", "probes", "GET", "/api/audit", "", "audit tail", "Audit", "Audit"),
	}
}

// BySection groups actions for Web Control.
func BySection() map[string][]Action {
	m := map[string][]Action{}
	for _, x := range All() {
		m[x.Section] = append(m[x.Section], x)
	}
	return m
}

// ForSurface filters by surface name (web|tg|cli|tui).
func ForSurface(surface string) []Action {
	var out []Action
	for _, x := range All() {
		for _, s := range x.Surfaces {
			if s == surface {
				out = append(out, x)
				break
			}
		}
	}
	return out
}

// HasSurface reports whether action is on surface.
func (a Action) HasSurface(surface string) bool {
	for _, s := range a.Surfaces {
		if s == surface {
			return true
		}
	}
	return false
}

// Label returns EN or RU label.
func (a Action) Label(lang string) string {
	if lang == "ru" && a.LabelRU != "" {
		return a.LabelRU
	}
	if a.LabelEN != "" {
		return a.LabelEN
	}
	return a.ID
}

// Get returns action by id or false.
func Get(id string) (Action, bool) {
	for _, a := range All() {
		if a.ID == id {
			return a, true
		}
	}
	return Action{}, false
}

// Sections returns unique section names in stable order.
func Sections() []string {
	order := []string{"overview", "vpn", "nodes", "edge", "nvr", "git", "dns", "backup", "probes"}
	have := map[string]bool{}
	for _, a := range All() {
		have[a.Section] = true
	}
	var out []string
	for _, s := range order {
		if have[s] {
			out = append(out, s)
		}
	}
	return out
}

// MatrixMarkdown renders UI↔API matrix for docs / CLI.
func MatrixMarkdown() string {
	var b strings.Builder
	b.WriteString("# OpCatalog matrix (day-2)\n\n")
	b.WriteString("Deploy (fleet/primary/secondary/edge) is Mac-only — see [DEPLOY-PARITY.md](DEPLOY-PARITY.md).\n\n")
	b.WriteString("| ID | API | CLI hint | Web | TG | CLI | TUI |\n")
	b.WriteString("|----|-----|----------|-----|----|-----|-----|\n")
	for _, a := range All() {
		mark := func(s string) string {
			if a.HasSurface(s) {
				return "✅"
			}
			return "—"
		}
		cli := a.CLI
		if cli == "" {
			cli = "—"
		}
		b.WriteString("| `" + a.ID + "` | `" + a.Method + " " + a.Path + "` | `" + cli + "` | " +
			mark(SurfWeb) + " | " + mark(SurfTG) + " | " + mark(SurfCLI) + " | " + mark(SurfTUI) + " |\n")
	}
	b.WriteString("\n**Rule:** new day-2 capability → entry in `internal/opcatalog` **before** UI-only code.\n")
	return b.String()
}
