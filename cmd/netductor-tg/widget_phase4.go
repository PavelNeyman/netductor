package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

// Phase 4 residual catalog/tools widgets. Main menu stays classic keyboard.

func residualShell(title, fields, body, screen, extraBtns string) string {
	if extraBtns == "" {
		extraBtns = ""
	}
	return `<!DOCTYPE html><html><head><meta charset="utf-8"/><style>` + phase1WidgetCSS + `</style></head><body>
<h1 id="title">` + title + `</h1>
` + fields + `
` + body + `
<div class="nav">
<button class="primary" data-act="` + screen + `:refresh">↻ Refresh</button>
` + extraBtns + `
<button data-act="` + screen + `:menu">Menu</button>
</div>
<div class="hint" id="hint"></div>
<script>` + phase1WidgetJS(screen) + `</script></body></html>`
}

func residualNavFull(token string, q *widgetQuery, admin int64, screen string, full func(token string, chat int64, msgID int)) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"full"}`)
	full(token, chat, q.MessageID)
	return true
}

// --- digest ---

var digestWidgetHTML = residualShell("Fleet health",
	`<div class="row"><span class="k">primary</span><span class="v" data-f="pri">—</span></div>
<div class="row"><span class="k">update</span><span class="v" data-f="upd">—</span></div>
<div class="row"><span class="k">services</span><span class="v" data-f="svc">—</span></div>
<div class="row"><span class="k">secondary</span><span class="v" data-f="sec">—</span></div>`,
	``,
	"digest",
	`<button data-act="digest:full">Full</button><button data-act="digest:status">Status</button>`,
)

type digestWData struct {
	Title string `json:"title"`
	Pri   string `json:"pri"`
	Upd   string `json:"upd"`
	UpdOK *bool  `json:"upd_ok,omitempty"`
	Svc   string `json:"svc"`
	Sec   string `json:"sec"`
	SecOK *bool  `json:"sec_ok,omitempty"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

func digestWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	d := fleet.Build()
	pri := d.PrimaryVersion
	if pri == "" {
		pri = ndver.Running()
	}
	upd := "up to date"
	ok := true
	if d.UpdateAvailable {
		upd = "🆕 " + d.LatestRelease
		ok = false
		if ru {
			upd = "🆕 " + d.LatestRelease
		}
	}
	var svcParts []string
	for u, st := range d.Services {
		short := u
		if strings.HasPrefix(u, "netductor-") {
			short = strings.TrimPrefix(u, "netductor-")
		}
		if st == "active" {
			svcParts = append(svcParts, short+"✓")
		} else {
			svcParts = append(svcParts, short+"✗")
		}
	}
	secLine := "—"
	var secOK *bool
	if len(d.Secondaries) > 0 {
		s := d.Secondaries[0]
		name := s.Name
		if name == "" {
			name = s.ID
		}
		t := s.Online
		secOK = &t
		if s.Online {
			secLine = name + " on"
		} else {
			secLine = name + " off"
		}
	}
	title, hint := "Fleet health", "Full → classic digest"
	if ru {
		title, hint = "Fleet health", "Full — классический digest"
	}
	b, err := json.Marshal(digestWData{
		Title: title, Pri: pri, Upd: upd, UpdOK: &ok, Svc: strings.Join(svcParts, " · "),
		Sec: secLine, SecOK: secOK, Hint: hint, TS: time.Now().Unix(),
	})
	if err != nil {
		return "", err
	}
	if len(b) > 1200 {
		return "", fmt.Errorf("digest data %d", len(b))
	}
	return string(b), nil
}

func handleDigestWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		return false
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	case "status":
		widgetNavStatus(token, q, admin)
		return true
	case "full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, formatFleetDigestHTML(), statusKeyboard())
		return true
	default:
		return false
	}
}

// --- actions ---

var actionsWidgetHTML = residualShell("Actions",
	``,
	`<div class="pre" id="body">—</div>`,
	"actions",
	`<button data-act="actions:full">Full</button><button data-act="actions:status">Status</button>`,
)

type actionsWData struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

func actionsWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Actions", "Full → classic"
	if ru {
		title, hint = "Действия", "Full — классика"
	}
	// strip HTML tags roughly from FormatActionsHTML
	html := fleet.FormatActionsHTML(ru)
	plain := html
	for _, tag := range []string{"<b>", "</b>", "<i>", "</i>", "<code>", "</code>", "<br>", "<br/>", "<table>", "</table>", "<tr>", "</tr>", "<td>", "</td>", "<th>", "</th>", "<tg-button-row>", "</tg-button-row>"} {
		plain = strings.ReplaceAll(plain, tag, " ")
	}
	// remove remaining tags
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
	plain = strings.Join(strings.Fields(plain), " ")
	plain = widgetTruncate(plain, 700)
	if plain == "" {
		plain = "(none)"
	}
	b, err := json.Marshal(actionsWData{Title: title, Body: plain, Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > 1500 {
		return "", fmt.Errorf("actions data %d", len(b))
	}
	return string(b), nil
}

func handleActionsWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		return false
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	case "status":
		widgetNavStatus(token, q, admin)
		return true
	case "full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, fleet.FormatActionsHTML(getLang() != "en"), statusKeyboard())
		return true
	default:
		return false
	}
}

// --- audit ---

var auditWidgetHTML = residualShell("Audit",
	``,
	`<div class="pre" id="body">—</div>`,
	"audit",
	`<button data-act="audit:full">Full</button>`,
)

type auditWData struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

func auditWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Audit", "tail truncated"
	if ru {
		title, hint = "Аудит", "хвост журнала"
	}
	out := runND("audit", "tail")
	if strings.TrimSpace(out) == "" {
		out = "(empty)"
	}
	out = widgetTruncate(out, 700)
	b, err := json.Marshal(auditWData{Title: title, Body: out, Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > 1500 {
		return "", fmt.Errorf("audit data %d", len(b))
	}
	return string(b), nil
}

func handleAuditWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		return false
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	case "full":
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

// --- sessions ---

var sessionsWidgetHTML = residualShell("Sessions",
	``,
	`<div class="pre" id="body">—</div>`,
	"sessions",
	`<button data-act="sessions:full">Full</button>`,
)

func sessionsWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Sessions", "revoke stays on classic Full"
	if ru {
		title, hint = "Сессии", "revoke — на Full"
	}
	out := runND("vpn", "session", "list")
	if strings.TrimSpace(out) == "" {
		out = "(none)"
	}
	out = widgetTruncate(out, 700)
	b, err := json.Marshal(auditWData{Title: title, Body: out, Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > 1500 {
		return "", fmt.Errorf("sessions data %d", len(b))
	}
	return string(b), nil
}

func handleSessionsWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		return false
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	case "full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		// re-enter classic sessions UI via callback path content
		out := runND("vpn", "session", "list")
		if strings.TrimSpace(out) == "" {
			out = "(none)"
		}
		ru := getLang() != "en"
		title := "🔑 <b>Sessions</b>\n<pre>" + esc(out) + "</pre>\n"
		rev := "🗑 Revoke all"
		if ru {
			title = "🔑 <b>Сессии</b>\n<pre>" + esc(out) + "</pre>\n"
			rev = "🗑 Отозвать все"
		}
		title += `<tg-button-row align="left"><tg-button type="callback_data" style="danger" data="m:sessions:revoke">` + rev + `</tg-button></tg-button-row>`
		reply(token, chat, q.MessageID, title, operatorSubKeyboard())
		return true
	default:
		return false
	}
}

// --- routers ---

var routersWidgetHTML = residualShell("Routers",
	``,
	`<div class="pre" id="body">—</div>`,
	"routers",
	`<button data-act="routers:full">Full</button><button data-act="routers:pending">Pending</button>`,
)

func routersWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Routers", "edge list truncated"
	if ru {
		title, hint = "Роутеры", "edge список"
	}
	out := routersText()
	out = widgetTruncate(strings.TrimSpace(out), 700)
	if out == "" {
		out = "(empty)"
	}
	b, err := json.Marshal(auditWData{Title: title, Body: out, Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > 1500 {
		return "", fmt.Errorf("routers data %d", len(b))
	}
	return string(b), nil
}

func handleRoutersWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		return false
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	case "pending":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		if tryReplyWidget(token, chat, q.MessageID, "pending") {
			return true
		}
		t := pendingText()
		reply(token, chat, q.MessageID, formatPendingHTML(t), pendingKeyboard(t))
		return true
	case "full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, routersHubHTML(), routersKeyboard("m:fleet"))
		return true
	default:
		return false
	}
}

// --- lampac ---

var lampacWidgetHTML = residualShell("Lampac",
	``,
	`<div class="pre" id="body">—</div>`,
	"lampac",
	`<button data-act="lampac:full">Full</button><button data-act="lampac:addons">Addons</button>`,
)

func lampacWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Lampac", "Full → classic manage"
	if ru {
		title, hint = "Lampac", "Full — управление"
	}
	// strip to text from HTML formatter
	html := formatLampacHTML()
	plain := html
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
	plain = strings.Join(strings.Fields(plain), " ")
	plain = widgetTruncate(plain, 700)
	b, err := json.Marshal(auditWData{Title: title, Body: plain, Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > 1500 {
		return "", fmt.Errorf("lampac data %d", len(b))
	}
	return string(b), nil
}

func handleLampacWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		return false
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	case "addons":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		if tryReplyWidget(token, chat, q.MessageID, "addons") {
			return true
		}
		reply(token, chat, q.MessageID, formatAddonsHTML(), addonsKeyboard())
		return true
	case "full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		pl := "Addons"
		if getLang() != "en" {
			pl = "Аддоны"
		}
		reply(token, chat, q.MessageID, formatLampacHTML(), navKeyboard("m:addons", pl))
		return true
	default:
		return false
	}
}

// --- canary ---

var canaryWidgetHTML = residualShell("Canary",
	``,
	`<div class="pre" id="body">—</div>`,
	"canary",
	`<button data-act="canary:full">Full</button>`,
)

func canaryWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Canary", "Full → classic canary hub"
	if ru {
		title, hint = "Canary", "Full — хаб"
	}
	html := formatCanaryHub()
	plain := html
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
	plain = strings.Join(strings.Fields(plain), " ")
	plain = widgetTruncate(plain, 700)
	b, err := json.Marshal(auditWData{Title: title, Body: plain, Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > 1500 {
		return "", fmt.Errorf("canary data %d", len(b))
	}
	return string(b), nil
}

func handleCanaryWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		return false
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	case "full":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, formatCanaryHub(), canaryHubKB())
		return true
	default:
		return false
	}
}

func init() {
	registerWidget(&widgetScreen{ID: "digest", HTML: digestWidgetHTML, BuildData: digestWidgetChatDataJSON, HandleAct: handleDigestWidgetAct})
	registerWidget(&widgetScreen{ID: "actions", HTML: actionsWidgetHTML, BuildData: actionsWidgetChatDataJSON, HandleAct: handleActionsWidgetAct})
	registerWidget(&widgetScreen{ID: "audit", HTML: auditWidgetHTML, BuildData: auditWidgetChatDataJSON, HandleAct: handleAuditWidgetAct})
	registerWidget(&widgetScreen{ID: "sessions", HTML: sessionsWidgetHTML, BuildData: sessionsWidgetChatDataJSON, HandleAct: handleSessionsWidgetAct})
	registerWidget(&widgetScreen{ID: "routers", HTML: routersWidgetHTML, BuildData: routersWidgetChatDataJSON, HandleAct: handleRoutersWidgetAct})
	registerWidget(&widgetScreen{ID: "lampac", HTML: lampacWidgetHTML, BuildData: lampacWidgetChatDataJSON, HandleAct: handleLampacWidgetAct})
	registerWidget(&widgetScreen{ID: "canary", HTML: canaryWidgetHTML, BuildData: canaryWidgetChatDataJSON, HandleAct: handleCanaryWidgetAct})
}
