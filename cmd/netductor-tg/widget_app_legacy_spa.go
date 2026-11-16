package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/netmap"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

// SPA builders for screens that previously used appFromLegacyWidget (separate HTML + JSON).
// Same chat_data contract as the rest of the SPA: Rows / Lines / Nav / Btns.

func widgetStripHTML(s string) string {
	plain := s
	for {
		i := strings.IndexByte(plain, '<')
		if i < 0 {
			break
		}
		j := strings.IndexByte(plain[i:], '>')
		if j < 0 {
			break
		}
		plain = plain[:i] + " " + plain[i+j+1:]
	}
	return strings.Join(strings.Fields(plain), " ")
}

func buildAppChannel() (string, error) {
	d := buildChannelWidgetData()
	rows := [][2]string{
		{appTitle("secondary", "secondary"), d.Sec},
		{appTitle("uplink", "uplink"), d.Up},
		{"face", d.Face},
		{"rtt", d.RTT},
		{appTitle("egress", "egress"), d.Eg},
	}
	st := appState{
		S: "channel", T: d.Title, Rows: rows, H: d.Hint,
		Nav: []string{"secondary", "status", "netmap"},
	}
	if d.Spk != "" && d.Spk != "—" {
		st.Spk = d.Spk
	}
	return appMarshal(st)
}

func buildAppNetmap() (string, error) {
	ru := getLang() != "en"
	body := widgetTruncate(netmap.Text(ru), 500)
	if body == "" {
		body = appTitle("(empty)", "(пусто)")
	}
	return appMarshal(appState{
		S: "netmap", T: appTitle("Netmap", "Карта сети"),
		Lines: body,
		H:     appTitle("topology", "топология"),
		Nav:   []string{"channel", "status", "secondary"},
	})
}

func buildAppSecondarySPA() (string, error) {
	d := fleet.Build()
	rows := [][2]string{
		{"name", "—"},
		{appTitle("online", "online"), "—"},
		{appTitle("version", "версия"), "—"},
		{appTitle("uplink", "uplink"), "—"},
		{"sing-box", "—"},
		{"ip", "—"},
	}
	var lines []string
	if len(d.Secondaries) > 0 {
		s := d.Secondaries[0]
		name := s.Name
		if name == "" {
			name = s.ID
		}
		rows[0][1] = name
		if s.Online {
			rows[1][1] = "online"
		} else {
			rows[1][1] = "offline"
		}
		ver := s.Version
		if ver == "" {
			ver = "—"
		}
		rows[2][1] = ver
		if s.Uplink {
			rows[3][1] = "ok"
		} else {
			rows[3][1] = "down"
		}
		if s.SingBox {
			rows[4][1] = "ok"
		} else {
			rows[4][1] = "down"
		}
		ip := s.IP
		if ip == "" {
			ip = "—"
		}
		rows[5][1] = ip
		for i, sec := range d.Secondaries {
			if i >= 6 {
				break
			}
			n := sec.Name
			if n == "" {
				n = sec.ID
			}
			st := "off"
			if sec.Online {
				st = "on"
			}
			lines = append(lines, fmt.Sprintf("%s %s", n, st))
		}
	}
	st := appState{
		S: "secondary", T: appTitle("Secondary", "Secondary"),
		Rows: rows,
		H:    appTitle("first agent · list → seclist", "первый агент · список → seclist"),
		Nav:  []string{"seclist", "channel", "fleet", "status"},
	}
	if len(lines) > 1 {
		st.Lines = strings.Join(lines, "\n")
	}
	return appMarshal(st)
}

func buildAppProbes() (string, error) {
	out := runND("probe")
	body := widgetTruncate(strings.TrimSpace(out), 500)
	if body == "" {
		body = appTitle("(empty)", "(пусто)")
	}
	return appMarshal(appState{
		S: "probes", T: appTitle("Probes", "Probes"),
		Lines: body,
		H:     appTitle("netductor probe", "netductor probe"),
		Nav:   []string{"doctor", "stack", "adv"},
	})
}

func buildAppDigest() (string, error) {
	d := fleet.Build()
	pri := d.PrimaryVersion
	if pri == "" {
		pri = ndver.Running()
	}
	upd := appTitle("up to date", "актуально")
	if d.UpdateAvailable && d.LatestRelease != "" {
		upd = "🆕 " + d.LatestRelease
	}
	order := []string{"sing-box", "netductor-api", "netductor-telegram-bot", "netductor-redirect"}
	var svcParts []string
	for _, u := range order {
		st, ok := d.Services[u]
		if !ok {
			continue
		}
		short := u
		if strings.HasPrefix(u, "netductor-") {
			short = strings.TrimPrefix(u, "netductor-")
		}
		mark := "·"
		if st == "active" {
			mark = "✓"
		}
		svcParts = append(svcParts, mark+short)
	}
	svc := strings.Join(svcParts, " ")
	if svc == "" {
		svc = "—"
	}
	on, total := 0, len(d.Secondaries)
	for _, s := range d.Secondaries {
		if s.Online {
			on++
		}
	}
	sec := fmt.Sprintf("%d/%d on", on, total)
	rows := [][2]string{
		{appTitle("primary", "primary"), pri},
		{appTitle("update", "обновление"), upd},
		{appTitle("services", "сервисы"), widgetTruncate(svc, 80)},
		{appTitle("secondary", "secondary"), sec},
	}
	if d.SPOK != nil {
		v := "fail"
		if *d.SPOK {
			v = "ok"
		}
		rows = append(rows, [2]string{"SP", v})
	}
	if d.PSOK != nil {
		v := "fail"
		if *d.PSOK {
			v = "ok"
		}
		rows = append(rows, [2]string{"PS", v})
	}
	return appMarshal(appState{
		S: "digest", T: appTitle("Digest", "Дайджест"),
		Rows: rows,
		H:    appTitle("fleet health snapshot", "снимок здоровья флота"),
		Nav:  []string{"status", "updates", "secondary", "channel"},
	})
}

func buildAppAddons() (string, error) {
	out := runND("addons", "status")
	lampac, blocky := "—", "—"
	low := strings.ToLower(out)
	if strings.Contains(low, "lampac") {
		lampac = "present"
		if strings.Contains(low, "inactive") {
			lampac = "inactive"
		}
	}
	if strings.Contains(low, "blocky") {
		blocky = "present"
	}
	body := widgetTruncate(strings.TrimSpace(out), 450)
	return appMarshal(appState{
		S: "addons", T: appTitle("Addons", "Аддоны"),
		Rows: [][2]string{
			{"lampac", lampac},
			{"blocky", blocky},
		},
		Lines: body,
		H:     appTitle("Full → classic manage", "Full — классическое управление"),
		Nav:   []string{"lampac", "fleet", "media"},
		Btns: [][2]string{
			{"ad_full", appTitle("Full manage", "Управление")},
		},
	})
}

func buildAppLampac() (string, error) {
	plain := widgetTruncate(widgetStripHTML(formatLampacHTML()), 500)
	if plain == "" {
		plain = "—"
	}
	return appMarshal(appState{
		S: "lampac", T: "Lampac",
		Lines: plain,
		H:     appTitle("Full → classic", "Full — классика"),
		Nav:   []string{"addons", "media", "canary"},
		Btns: [][2]string{
			{"lp_full", appTitle("Full manage", "Управление")},
		},
	})
}

func buildAppAudit() (string, error) {
	out := runND("audit", "tail")
	body := widgetTruncate(strings.TrimSpace(out), 500)
	if body == "" {
		body = appTitle("(empty)", "(пусто)")
	}
	return appMarshal(appState{
		S: "audit", T: appTitle("Audit", "Аудит"),
		Lines: body,
		H:     appTitle("journal tail", "хвост журнала"),
		Nav:   []string{"operator", "session", "alerts"},
		Btns: [][2]string{
			{"au_full", appTitle("Full in chat", "Полный в чат")},
		},
	})
}

func buildAppActions() (string, error) {
	ru := getLang() != "en"
	plain := widgetTruncate(widgetStripHTML(fleet.FormatActionsHTML(ru)), 500)
	if plain == "" {
		plain = appTitle("(none)", "(нет)")
	}
	return appMarshal(appState{
		S: "actions", T: appTitle("Actions", "Действия"),
		Lines: plain,
		H:     appTitle("Full → classic keyboard", "Full — классическая клавиатура"),
		Nav:   []string{"status", "incident"},
		Btns: [][2]string{
			{"act_full", appTitle("Full classic", "Классика")},
		},
	})
}

func buildAppRouters() (string, error) {
	out := widgetTruncate(strings.TrimSpace(routersText()), 500)
	if out == "" {
		out = appTitle("(empty)", "(пусто)")
	}
	return appMarshal(appState{
		S: "routers", T: appTitle("Routers", "Роутеры"),
		Lines: out,
		H:     appTitle("edge list", "список edge"),
		Nav:   []string{"pending", "sites", "edge", "edgeguest"},
		Btns: [][2]string{
			{"rt_full", appTitle("Full hub", "Полный хаб")},
		},
	})
}


// buildAppSecondaryCard opens one secondary from seclist (Focus = device id).
func buildAppSecondaryCard(id string) (string, error) {
	if id == "" {
		return appMarshal(appState{S: "seclist", T: "Secondary", B: appTitle("empty", "пусто")})
	}
	d := fleet.Build()
	rows := [][2]string{
		{"id", id},
		{appTitle("online", "online"), "—"},
		{appTitle("version", "версия"), "—"},
		{appTitle("uplink", "uplink"), "—"},
		{"sing-box", "—"},
		{"ip", "—"},
	}
	title := id
	for _, s := range d.Secondaries {
		if s.ID != id && s.Name != id {
			continue
		}
		if s.Name != "" {
			title = s.Name
		}
		rows[0][1] = s.ID
		if s.Online {
			rows[1][1] = "online"
		} else {
			rows[1][1] = "offline"
		}
		ver := s.Version
		if ver == "" {
			ver = "—"
		}
		rows[2][1] = ver
		if s.Uplink {
			rows[3][1] = "ok"
		} else {
			rows[3][1] = "down"
		}
		if s.SingBox {
			rows[4][1] = "ok"
		} else {
			rows[4][1] = "down"
		}
		ip := s.IP
		if ip == "" {
			ip = "—"
		}
		rows[5][1] = ip
		break
	}
	return appMarshal(appState{
		S: "seclist", T: title, Rows: rows,
		H:   appTitle("secondary agent", "агент secondary"),
		Nav: []string{"secondary", "fleet", "channel", "secops"},
		Btns: [][2]string{
			{"up", appTitle("Upgrade", "Обновить")},
		},
	})
}

// handleLegacySPAActs — classic “Full” shortcuts from migrated screens (open HTML/keyboard outside SPA).
func handleLegacySPAActs(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "ad_full":
		// Always classic HTML manage — tryReplyWidget("addons") would re-open SPA.
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, formatAddonsHTML(), addonsKeyboard())
		return true
	case "lp_full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		pl := T("back")
		reply(token, chat, q.MessageID, formatLampacHTML(), navKeyboard("m:addons", pl))
		return true
	case "act_full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, fleet.FormatActionsHTML(getLang() != "en"), statusKeyboard())
		return true
	case "rt_full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, routersHubHTML(), routersKeyboard("m:fleet"))
		return true
	case "au_full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		out := runND("audit", "tail")
		if strings.TrimSpace(out) == "" {
			out = "(empty)"
		}
		title := "📋 <b>Audit</b>\n"
		if getLang() != "en" {
			title = "📋 <b>Аудит</b>\n"
		}
		reply(token, chat, q.MessageID, title+"<pre>"+esc(truncate(out, 3500))+"</pre>", operatorSubKeyboard())
		return true
	default:
		return false
	}
}

