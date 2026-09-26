// Package opcatalog is the shared day-2 action registry for Web Control and TG.
package opcatalog

// Action is one session-API operation exposed on operator surfaces.
type Action struct {
	ID       string   `json:"id"`
	Section  string   `json:"section"` // overview|vpn|nodes|edge|nvr|git|backup|probes
	Method   string   `json:"method"`  // GET|POST
	Path     string   `json:"path"`
	Body     string   `json:"body,omitempty"` // default POST body
	LabelEN  string   `json:"label_en"`
	LabelRU  string   `json:"label_ru"`
	Surfaces []string `json:"surfaces"` // web, tg
}

// All returns the static catalog (extend here — Web/TG consume the same list).
func All() []Action {
	s := func(en, ru string) (string, string) { return en, ru }
	a := func(id, sec, method, path, body, en, ru string, surfaces ...string) Action {
		if len(surfaces) == 0 {
			surfaces = []string{"web", "tg"}
		}
		return Action{ID: id, Section: sec, Method: method, Path: path, Body: body, LabelEN: en, LabelRU: ru, Surfaces: surfaces}
	}
	_ = s
	return []Action{
		a("health", "overview", "GET", "/health", "", "Health", "Health"),
		a("doctor", "overview", "GET", "/api/doctor", "", "Doctor", "Doctor"),
		a("domain", "overview", "GET", "/api/domain", "", "Domain", "Домен"),
		a("bot", "overview", "GET", "/api/bot-status", "", "Bot status", "Статус бота"),
		a("status", "overview", "GET", "/api/status", "", "Status", "Статус"),
		a("metrics", "overview", "GET", "/api/metrics", "", "Metrics", "Метрики"),
		a("metrics-hist", "overview", "GET", "/api/metrics/history", "", "Metrics history", "История метрик"),
		a("addons", "overview", "GET", "/api/addons", "", "Addons", "Дополнения"),
		a("addons-lampac", "overview", "GET", "/api/addons/lampac", "", "Lampac", "Lampac"),
		a("sni", "overview", "GET", "/api/sni", "", "SNI", "SNI"),
		a("sni-presets", "overview", "GET", "/api/sni-presets", "", "SNI presets", "Пресеты SNI"),
		a("latest", "overview", "GET", "/api/latest", "", "Latest", "Latest"),
		a("sessions", "overview", "GET", "/api/sessions", "", "Sessions", "Sessions"),

		a("vpn-users", "vpn", "GET", "/vpn/users", "", "List users", "Список users"),
		a("vpn-refresh", "vpn", "POST", "/api/vpn/refresh-links", "{}", "Refresh links", "Обновить ссылки"),

		a("nodes", "nodes", "GET", "/api/nodes", "", "Nodes", "Ноды"),
		a("nodes-self", "nodes", "GET", "/api/nodes/self", "", "Self", "Self"),
		a("secondary", "nodes", "GET", "/api/secondary/status", "", "Secondary status", "Статус secondary"),
		a("secondary-links", "nodes", "GET", "/api/secondary/links", "", "Secondary links", "Ссылки secondary"),
		a("ssh-hosts", "nodes", "GET", "/api/ssh-hosts", "", "SSH hosts", "SSH hosts"),
		a("ssh-clear", "nodes", "POST", "/api/ssh-hosts/clear", "{}", "SSH hosts clear", "Очистить SSH hosts"),
		a("mtls-certs", "nodes", "GET", "/api/mtls/certs", "", "mTLS certs", "mTLS сертификаты"),
		a("sites", "nodes", "GET", "/api/sites", "", "Sites", "Сайты"),

		a("edge-pending", "edge", "GET", "/api/edge/pending", "", "Pending", "Pending"),
		a("edge-devices", "edge", "GET", "/api/edge/devices", "", "Devices", "Devices"),
		a("edge-metrics", "edge", "GET", "/api/edge/metrics", "", "Metrics", "Metrics"),
		a("edge-guest-st", "edge", "GET", "/api/edge/guest/status", "", "Guest status", "Guest status"),

		a("nvr-cameras", "nvr", "GET", "/api/nvr/cameras", "", "Cameras", "Cameras"),
		a("nvr-config", "nvr", "GET", "/api/nvr/config", "", "Config", "Config"),
		a("nvr-storage", "nvr", "GET", "/api/nvr/storage", "", "Storage", "Storage"),
		a("nvr-events", "nvr", "GET", "/api/nvr/events", "", "Events", "Events"),
		a("nvr-segments", "nvr", "GET", "/api/nvr/segments", "", "Segments", "Segments"),
		a("nvr-retention", "nvr", "POST", "/api/nvr/retention/run", "{}", "Retention run", "Retention"),
		a("nvr-go2rtc", "nvr", "POST", "/api/nvr/go2rtc", "{}", "go2rtc write", "go2rtc"),

		a("git-repos", "git", "GET", "/api/git/repos", "", "Repos", "Repos"),
		a("git-pipelines", "git", "GET", "/api/git/pipelines", "", "Pipelines", "Pipelines"),
		a("reg-status", "git", "GET", "/api/registry/status", "", "Registry status", "Registry"),
		a("reg-ensure", "git", "POST", "/api/registry/ensure", "{}", "Registry ensure", "Registry ensure"),

		
		a("dns-lists", "dns", "GET", "/api/dns/lists", "", "DNS lists", "DNS списки"),
		a("dns-set", "dns", "POST", "/api/dns/set", `{"id":"","enabled":true}`, "DNS set list", "DNS вкл/выкл список"),
		a("dns-reload", "dns", "POST", "/api/dns/reload", "{}", "DNS reload", "DNS reload"),
		a("backup-schedule", "backup", "GET", "/api/backup/schedule", "", "Backup schedule", "Расписание бэкапа"),
		a("backup-list", "backup", "GET", "/api/backup/list", "", "Backup files", "Файлы бэкапа"),
		a("backup-peer", "backup", "GET", "/api/backup/peer", "", "Peer", "Peer"),
		a("backup-run", "backup", "POST", "/api/backup/run", "{}", "Run now", "Бэкап сейчас"),
		a("sec-export", "backup", "GET", "/api/secondary/export", "", "Secondary export", "Secondary export"),

		a("probes", "probes", "GET", "/api/probes", "", "Probes", "Probes"),
		a("probes-cfg", "probes", "GET", "/api/probes/config", "", "Probes config", "Probes config"),
		a("audit", "probes", "GET", "/api/audit", "", "Audit", "Audit"),
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

// ForSurface filters by surface name (web|tg).
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
	order := []string{"overview", "vpn", "nodes", "edge", "nvr", "git", "backup", "dns", "probes"}
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
