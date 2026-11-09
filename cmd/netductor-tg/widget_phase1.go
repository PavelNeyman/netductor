package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/netmap"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

// Phase 1 read-only widgets: netmap, secondary, probes, versions.
// Gate: NETDUCTOR_TG_WIDGETS=1. Metrics stay folded into status.

const phase1WidgetCSS = `
body{margin:0;padding:10px;font:13px/1.35 system-ui,sans-serif;background:#0f1419;color:#e7ecf3}
h1{margin:0 0 8px;font-size:15px}.row{display:flex;justify-content:space-between;margin:3px 0;gap:8px}
.k{color:#8b98a5;flex-shrink:0}.v{font-weight:600;text-align:right;word-break:break-word}.ok{color:#3dd68c}.bad{color:#ff6b6b}
.pre{margin-top:8px;padding:8px;background:#1c2733;border-radius:8px;font:11px/1.35 ui-monospace,monospace;white-space:pre-wrap;max-height:280px;overflow:hidden}
.nav{display:flex;flex-wrap:wrap;gap:6px;margin-top:10px}
button{border:0;border-radius:8px;padding:7px 10px;font:12px system-ui;background:#1c2733;color:#e7ecf3}
button.primary{background:#2a6df4}.hint{margin-top:8px;color:#8b98a5;font-size:11px}
`

func phase1WidgetJS(screen string) string {
	return widgetBootJS + `
(function(){
var SCREEN="` + screen + `";
function set(id,t,cls){var el=document.getElementById(id);if(!el)return;el.textContent=t||'—';if(cls!==undefined)el.className='v'+(cls?' '+cls:'');}
function paint(d){if(!d)return;if(d.title){var t=document.getElementById('title');if(t)t.textContent=d.title;}
document.querySelectorAll('[data-f]').forEach(function(el){var k=el.getAttribute('data-f');if(!k||d[k]===undefined)return;
var cls='';if(d[k+'_ok']===false)cls='bad';else if(d[k+'_ok']===true)cls='ok';
el.textContent=d[k]||'—';el.className='v'+(cls?' '+cls:'');});
if(d.body){var p=document.getElementById('body');if(p)p.textContent=d.body;}
if(d.hint){var h=document.getElementById('hint');if(h)h.textContent=d.hint;}}
__ndBind(paint);
document.querySelectorAll('[data-act]').forEach(function(btn){
btn.addEventListener('click',function(){__ndSend(btn.getAttribute('data-act')||SCREEN+':refresh');});
});
})();`
}

func phase1Shell(title, fieldsHTML, extraBody, screen, extraBtns string) string {
	if extraBtns == "" {
		extraBtns = ""
	}
	return `<!DOCTYPE html><html><head><meta charset="utf-8"/><style>` + phase1WidgetCSS + `</style></head><body>
<h1 id="title">` + title + `</h1>
` + fieldsHTML + `
` + extraBody + `
<div class="nav">
<button class="primary" data-act="` + screen + `:refresh">↻ Refresh</button>
` + extraBtns + `
<button data-act="` + screen + `:menu">Menu</button>
</div>
<div class="hint" id="hint"></div>
<script>` + phase1WidgetJS(screen) + `</script></body></html>`
}

func widgetNavMenu(token string, q *widgetQuery, admin int64) {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"menu"}`)
	reply(token, chat, q.MessageID, T("main_menu"), mainKeyboard())
}

func widgetNavStatus(token string, q *widgetQuery, admin int64) {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"status"}`)
	if tryReplyWidget(token, chat, q.MessageID, "status") {
		return
	}
	reply(token, chat, q.MessageID, formatStatusPretty(), statusKeyboard())
}

func widgetTruncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// --- netmap ---

var netmapWidgetHTML = phase1Shell("Netmap",
	``,
	`<div class="pre" id="body">—</div>`,
	"netmap",
	`<button data-act="netmap:status">Status</button>`,
)

type netmapWidgetData struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

func netmapWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Netmap", "Topology text"
	if ru {
		title, hint = "Сеть", "Топология"
	}
	body := widgetTruncate(netmap.Text(ru), 700)
	b, err := json.Marshal(netmapWidgetData{Title: title, Body: body, Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		return "", fmt.Errorf("widget chat_data %d > widgetChatDataMax", len(b))
	}
	return string(b), nil
}

func handleNetmapWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	switch act {
	case "refresh", "":
		return false
	case "status":
		widgetNavStatus(token, q, admin)
		return true
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	default:
		return false
	}
}

// --- secondary ---

var secondaryWidgetHTML = phase1Shell("Secondary",
	`<div class="row"><span class="k">name</span><span class="v" data-f="name">—</span></div>
<div class="row"><span class="k">online</span><span class="v" data-f="online">—</span></div>
<div class="row"><span class="k">version</span><span class="v" data-f="ver">—</span></div>
<div class="row"><span class="k">uplink</span><span class="v" data-f="up">—</span></div>
<div class="row"><span class="k">sing-box</span><span class="v" data-f="sb">—</span></div>
<div class="row"><span class="k">ip</span><span class="v" data-f="ip">—</span></div>`,
	``,
	"secondary",
	`<button data-act="secondary:status">Status</button>`,
)

type secondaryWidgetData struct {
	Title  string `json:"title"`
	Name   string `json:"name"`
	Online string `json:"online"`
	OnlineOK *bool `json:"online_ok,omitempty"`
	Ver    string `json:"ver"`
	Up     string `json:"up"`
	UpOK   *bool  `json:"up_ok,omitempty"`
	Sb     string `json:"sb"`
	SbOK   *bool  `json:"sb_ok,omitempty"`
	IP     string `json:"ip"`
	Hint   string `json:"hint,omitempty"`
	TS     int64  `json:"ts"`
}

func secondaryWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Secondary", "First secondary agent"
	if ru {
		title, hint = "Secondary", "Первый secondary"
	}
	d := fleet.Build()
	data := secondaryWidgetData{Title: title, Hint: hint, TS: time.Now().Unix(), Name: "—", Online: "—", Ver: "—", Up: "—", Sb: "—", IP: "—"}
	if len(d.Secondaries) > 0 {
		s := d.Secondaries[0]
		name := s.Name
		if name == "" {
			name = s.ID
		}
		data.Name = name
		on := s.Online
		data.OnlineOK = &on
		if s.Online {
			data.Online = "online"
		} else {
			data.Online = "offline"
		}
		data.Ver = s.Version
		if data.Ver == "" {
			data.Ver = "—"
		}
		up := s.Uplink
		data.UpOK = &up
		if s.Uplink {
			data.Up = "ok"
		} else {
			data.Up = "down"
		}
		sb := s.SingBox
		data.SbOK = &sb
		if s.SingBox {
			data.Sb = "ok"
		} else {
			data.Sb = "down"
		}
		data.IP = s.IP
		if data.IP == "" {
			data.IP = "—"
		}
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		return "", fmt.Errorf("widget chat_data %d > widgetChatDataMax", len(b))
	}
	return string(b), nil
}

func handleSecondaryWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	switch act {
	case "refresh", "":
		return false
	case "status":
		widgetNavStatus(token, q, admin)
		return true
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	default:
		return false
	}
}

// --- probes ---

var probesWidgetHTML = phase1Shell("Probes",
	``,
	`<div class="pre" id="body">—</div>`,
	"probes",
	`<button data-act="probes:status">Status</button>`,
)

type probesWidgetData struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

func probesWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Probes", "netductor probe (truncated)"
	if ru {
		title, hint = "Probes", "netductor probe (кратко)"
	}
	out := runND("probe")
	body := widgetTruncate(strings.TrimSpace(out), 700)
	if body == "" {
		body = "(empty)"
	}
	b, err := json.Marshal(probesWidgetData{Title: title, Body: body, Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		return "", fmt.Errorf("widget chat_data %d > widgetChatDataMax", len(b))
	}
	return string(b), nil
}

func handleProbesWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	switch act {
	case "refresh", "":
		return false
	case "status":
		widgetNavStatus(token, q, admin)
		return true
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	default:
		return false
	}
}

// --- versions ---

var versionsWidgetHTML = phase1Shell("Versions",
	`<div class="row"><span class="k">primary</span><span class="v" data-f="pri">—</span></div>
<div class="row"><span class="k">secondary</span><span class="v" data-f="sec">—</span></div>
<div class="row"><span class="k">latest</span><span class="v" data-f="latest">—</span></div>
<div class="row"><span class="k">update</span><span class="v" data-f="upd">—</span></div>`,
	``,
	"versions",
	`<button data-act="versions:full">Full</button><button data-act="versions:status">Status</button>`,
)

type versionsWidgetData struct {
	Title  string `json:"title"`
	Pri    string `json:"pri"`
	Sec    string `json:"sec"`
	Latest string `json:"latest"`
	Upd    string `json:"upd"`
	UpdOK  *bool  `json:"upd_ok,omitempty"`
	Hint   string `json:"hint,omitempty"`
	TS     int64  `json:"ts"`
}

func versionsWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Versions", "Full opens classic Apply UI"
	if ru {
		title, hint = "Версии", "Full — классический Apply"
	}
	d := fleet.Build()
	pri := d.PrimaryVersion
	if pri == "" {
		pri = ndver.Running()
	}
	sec := "—"
	if len(d.Secondaries) > 0 && d.Secondaries[0].Version != "" {
		sec = d.Secondaries[0].Version
	}
	latest := d.LatestRelease
	if latest == "" {
		if t, err := ndupdate.LatestReleaseTag(); err == nil {
			latest = t
		}
	}
	if latest == "" {
		latest = "—"
	}
	upd := "ok"
	ok := true
	if latest != "—" && ndupdate.Newer(latest, pri) {
		upd = "available"
		ok = false
		if ru {
			upd = "доступно"
		}
	}
	b, err := json.Marshal(versionsWidgetData{
		Title: title, Pri: pri, Sec: sec, Latest: latest, Upd: upd, UpdOK: &ok, Hint: hint, TS: time.Now().Unix(),
	})
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		return "", fmt.Errorf("widget chat_data %d > widgetChatDataMax", len(b))
	}
	return string(b), nil
}

func handleVersionsWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		return false
	case "full":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"versions_full"}`)
		reply(token, chat, q.MessageID, formatVersionsHTML(false), versionsKeyboard())
		return true
	case "status":
		widgetNavStatus(token, q, admin)
		return true
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	default:
		return false
	}
}

func init() {
	registerWidget(&widgetScreen{ID: "netmap", HTML: netmapWidgetHTML, BuildData: netmapWidgetChatDataJSON, HandleAct: handleNetmapWidgetAct})
	registerWidget(&widgetScreen{ID: "secondary", HTML: secondaryWidgetHTML, BuildData: secondaryWidgetChatDataJSON, HandleAct: handleSecondaryWidgetAct})
	registerWidget(&widgetScreen{ID: "probes", HTML: probesWidgetHTML, BuildData: probesWidgetChatDataJSON, HandleAct: handleProbesWidgetAct})
	registerWidget(&widgetScreen{ID: "versions", HTML: versionsWidgetHTML, BuildData: versionsWidgetChatDataJSON, HandleAct: handleVersionsWidgetAct})
}
