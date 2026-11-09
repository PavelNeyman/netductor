package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/vpn"
)

// Phase 3 card widgets. Destructive/confirm stays classic reply after answerWidgetQuery.

const cardWidgetCSS = `
body{margin:0;padding:10px;font:13px/1.35 system-ui,sans-serif;background:#0f1419;color:#e7ecf3}
h1{margin:0 0 8px;font-size:15px}.row{display:flex;justify-content:space-between;gap:8px;margin:3px 0}
.k{color:#8b98a5}.v{font-weight:600;text-align:right;word-break:break-word}.ok{color:#3dd68c}.bad{color:#ff6b6b}
.nav{display:flex;flex-wrap:wrap;gap:6px;margin-top:10px}
button{border:0;border-radius:8px;padding:7px 10px;font:12px system-ui;background:#1c2733;color:#e7ecf3}
button.primary{background:#2a6df4}button.danger{background:#a33}.hint{margin-top:8px;color:#8b98a5;font-size:11px}
.pre{margin-top:8px;padding:8px;background:#1c2733;border-radius:8px;font:11px/1.35 ui-monospace,monospace;white-space:pre-wrap;max-height:160px;overflow:hidden}
`

func cardWidgetJS(screen string) string {
	return widgetBootJS + `(function(){
var SCREEN="` + screen + `";
function set(id,t,cls){var el=document.getElementById(id);if(!el)return;el.textContent=t||'—';el.className=(el.className||'').indexOf('pre')>=0?'pre':'v';if(cls)el.className+=' '+cls;}
function paint(d){if(!d)return;if(d.title){var t=document.getElementById('title');if(t)t.textContent=d.title;}
document.querySelectorAll('[data-f]').forEach(function(el){var k=el.getAttribute('data-f');if(!k||d[k]===undefined)return;
var cls='';if(d[k+'_ok']===false)cls='bad';else if(d[k+'_ok']===true)cls='ok';
el.textContent=d[k]||'—';if(el.className.indexOf('pre')<0)el.className='v'+(cls?' '+cls:'');});
if(d.body){var p=document.getElementById('body');if(p)p.textContent=d.body;}
if(d.hint){var h=document.getElementById('hint');if(h)h.textContent=d.hint;}
window.__cardCtx=d.ctx||'';}
function send(ev){var act=ev.currentTarget.getAttribute('data-act');
var ctx=window.__cardCtx||'';
if(act.indexOf(':')===act.lastIndexOf(':')&&ctx&&act!==SCREEN+':refresh'&&act!==SCREEN+':menu'&&act!==SCREEN+':users'&&act!==SCREEN+':nodes'){
  act=act+':'+ctx;
}
if(window.__ndSend)__ndSend(act);else if(window.TelegramWidget&&TelegramWidget.request)TelegramWidget.request(act);}
function data(){return window.__ndData?__ndData():null;}
paint(data());
document.querySelectorAll('[data-act]').forEach(function(btn){btn.addEventListener('click',send);});
if(window.TelegramWidget&&TelegramWidget.onEvent){
TelegramWidget.onEvent('chatDataChanged',function(){paint(data());});
}
})();`
}

// --- user card ---
// ctx in chat_data = username. Acts: user:refresh:name, user:enable:name, ...

var userCardWidgetHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"/><style>` + cardWidgetCSS + `</style></head><body>
<h1 id="title">User</h1>
<div class="row"><span class="k">name</span><span class="v" data-f="name">—</span></div>
<div class="row"><span class="k">state</span><span class="v" data-f="state">—</span></div>
<div class="row"><span class="k">kind</span><span class="v" data-f="kind">—</span></div>
<div class="nav">
<button class="primary" data-act="user:refresh">↻</button>
<button data-act="user:link">🔗</button>
<button data-act="user:enable">▶</button>
<button data-act="user:disable">⏸</button>
<button class="danger" data-act="user:revoke">🗑</button>
<button data-act="user:users">Users</button>
<button data-act="user:menu">Menu</button>
</div>
<div class="hint" id="hint">Revoke/rename use classic confirm where needed</div>
<script>` + cardWidgetJS("user") + `</script></body></html>`

type userCardData struct {
	Title string `json:"title"`
	Name  string `json:"name"`
	State string `json:"state"`
	StateOK *bool `json:"state_ok,omitempty"`
	Kind  string `json:"kind"`
	Ctx   string `json:"ctx"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

// activeUserCardCtx set when opening a user widget (single-admin bot).
var activeUserCardCtx string

func buildUserCardData(name string) userCardData {
	ru := getLang() != "en"
	if name == "" {
		name = activeUserCardCtx
	}
	en := "?"
	raw := runVPN("list")
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) == 0 {
			parts = strings.Fields(line)
		}
		if len(parts) > 0 && parts[0] == name {
			if len(parts) > 1 {
				en = parts[1]
			} else {
				en = "on"
			}
			break
		}
	}
	st := en
	ok := en != "off" && en != "?"
	kind := "user"
	if vpn.IsDeviceProfile(name) {
		kind = "device"
	}
	title := "User"
	hint := "▶ enable · ⏸ disable · 🗑 revoke → classic list"
	if ru {
		title, hint = "User", "▶ вкл · ⏸ выкл · 🗑 revoke"
	}
	return userCardData{Title: title, Name: name, State: st, StateOK: &ok, Kind: kind, Ctx: name, Hint: hint, TS: time.Now().Unix()}
}

func userCardChatDataJSON() (string, error) {
	b, err := json.Marshal(buildUserCardData(activeUserCardCtx))
	if err != nil {
		return "", err
	}
	if len(b) > 1024 {
		return "", fmt.Errorf("user card data %d", len(b))
	}
	return string(b), nil
}

func tryReplyUserWidget(token string, chat int64, msgID int, name string) bool {
	if !widgetsEnabled() || name == "" {
		return false
	}
	activeUserCardCtx = name
	return tryReplyWidget(token, chat, msgID, "user")
}

func handleUserCardAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	// act may be "refresh", "enable:name", "refresh:name"
	name := activeUserCardCtx
	if i := strings.IndexByte(act, ':'); i >= 0 {
		name = act[i+1:]
		act = act[:i]
	}
	if name == "" {
		name = activeUserCardCtx
	}
	if name != "" {
		activeUserCardCtx = name
	}
	switch act {
	case "refresh", "":
		return false
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	case "users":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"users"}`)
		if tryReplyWidget(token, chat, q.MessageID, "users") {
			return true
		}
		reply(token, chat, q.MessageID, formatUsersListHTML(), usersListKeyboard())
		return true
	case "link":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"link"}`)
		showUserAccess(token, chat, q.MessageID, name, "vless")
		return true
	case "enable":
		out := runVPN("enable", name)
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		if tryReplyUserWidget(token, chat, q.MessageID, name) {
			_ = out
			return true
		}
		reply(token, chat, q.MessageID, "✅ <pre>"+esc(out)+"</pre>\n\n"+formatUserHubHTML(name), userHubKeyboard(name))
		return true
	case "disable":
		out := runVPN("disable", name)
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		if tryReplyUserWidget(token, chat, q.MessageID, name) {
			return true
		}
		reply(token, chat, q.MessageID, "🚫 <pre>"+esc(out)+"</pre>\n\n"+formatUserHubHTML(name), userHubKeyboard(name))
		return true
	case "revoke":
		// classic — revoke leaves list
		out := runVPN("revoke", name)
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"users"}`)
		reply(token, chat, q.MessageID, "🗑 <pre>"+esc(out)+"</pre>", usersListKeyboard())
		return true
	default:
		return false
	}
}

// --- node card ---

var activeNodeCardCtx string

var nodeCardWidgetHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"/><style>` + cardWidgetCSS + `</style></head><body>
<h1 id="title">Node</h1>
<div class="row"><span class="k">host</span><span class="v" data-f="host">—</span></div>
<div class="row"><span class="k">role</span><span class="v" data-f="role">—</span></div>
<div class="row"><span class="k">status</span><span class="v" data-f="status">—</span></div>
<div class="row"><span class="k">version</span><span class="v" data-f="ver">—</span></div>
<div class="row"><span class="k">ip</span><span class="v" data-f="ip">—</span></div>
<div class="nav">
<button class="primary" data-act="node:refresh">↻</button>
<button data-act="node:journal">📋</button>
<button data-act="node:upgrade">⬆️</button>
<button data-act="node:restart">♻️ sb</button>
<button class="danger" data-act="node:reboot">Reboot</button>
<button data-act="node:nodes">Nodes</button>
<button data-act="node:menu">Menu</button>
</div>
<div class="hint" id="hint">Reboot → classic challenge</div>
<script>` + cardWidgetJS("node") + `</script></body></html>`

type nodeCardWData struct {
	Title  string `json:"title"`
	Host   string `json:"host"`
	Role   string `json:"role"`
	Status string `json:"status"`
	StatusOK *bool `json:"status_ok,omitempty"`
	Ver    string `json:"ver"`
	IP     string `json:"ip"`
	Ctx    string `json:"ctx"`
	Hint   string `json:"hint,omitempty"`
	TS     int64  `json:"ts"`
}

func buildNodeCardWData(id string) nodeCardWData {
	if id == "" {
		id = activeNodeCardCtx
	}
	c := loadNodeCard(id)
	host := c.Host
	if host == "" {
		host = id
	}
	on := strings.EqualFold(c.Status, "online")
	ru := getLang() != "en"
	title, hint := "Node", "Upgrade queues · Reboot needs code"
	if ru {
		title, hint = "Узел", "Upgrade в очередь · Reboot с кодом"
	}
	return nodeCardWData{
		Title: title, Host: host, Role: displayRole(c.Role), Status: c.Status, StatusOK: &on,
		Ver: c.Version, IP: c.IP, Ctx: id, Hint: hint, TS: time.Now().Unix(),
	}
}

func nodeCardChatDataJSON() (string, error) {
	b, err := json.Marshal(buildNodeCardWData(activeNodeCardCtx))
	if err != nil {
		return "", err
	}
	if len(b) > 1024 {
		return "", fmt.Errorf("node card data %d", len(b))
	}
	return string(b), nil
}

func tryReplyNodeWidget(token string, chat int64, msgID int, id string) bool {
	if !widgetsEnabled() || id == "" {
		return false
	}
	activeNodeCardCtx = id
	return tryReplyWidget(token, chat, msgID, "node")
}

func handleNodeCardAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	id := activeNodeCardCtx
	if i := strings.IndexByte(act, ':'); i >= 0 {
		id = act[i+1:]
		act = act[:i]
	}
	if id == "" {
		id = activeNodeCardCtx
	}
	if id != "" {
		activeNodeCardCtx = id
	}
	switch act {
	case "refresh", "":
		return false
	case "menu":
		widgetNavMenu(token, q, admin)
		return true
	case "nodes":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"nodes"}`)
		if tryReplyWidget(token, chat, q.MessageID, "nodes") {
			return true
		}
		reply(token, chat, q.MessageID, formatNodesListHTML(), nodesListKeyboard())
		return true
	case "journal":
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, formatJournalHTML(id), nodeCardKeyboard(id))
		return true
	case "upgrade":
		_ = enqueueNodeCmd(id, "upgrade")
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, formatCmdQueuedHTML("upgrade", id, ""), nodeCardKeyboard(id))
		return true
	case "restart":
		out := restartSingBox(id)
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, "♻️ <b>Restart sing-box</b>\n<pre>"+esc(out)+"</pre>", nodeCardKeyboard(id))
		return true
	case "reboot":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"reboot"}`)
		handleRebootCB(token, chat, q.MessageID, "m:n:reboot:"+id)
		return true
	default:
		return false
	}
}

// --- operator hub (read + nav to classic toggles) ---

var operatorWidgetHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"/><style>` + cardWidgetCSS + `</style></head><body>
<h1 id="title">Operator</h1>
<div class="row"><span class="k">API WAN</span><span class="v" data-f="api">—</span></div>
<div class="row"><span class="k">Users→SP</span><span class="v" data-f="usp">—</span></div>
<div class="row"><span class="k">SP / PS</span><span class="v" data-f="plane">—</span></div>
<div class="nav">
<button class="primary" data-act="operator:refresh">↻</button>
<button data-act="operator:full">Full</button>
<button data-act="operator:svc">Svc paths</button>
<button data-act="operator:menu">Menu</button>
</div>
<div class="hint" id="hint">Toggles open classic operator UI</div>
<script>` + cardWidgetJS("operator") + `</script></body></html>`

type operatorWData struct {
	Title string `json:"title"`
	API   string `json:"api"`
	APIOK *bool  `json:"api_ok,omitempty"`
	Usp   string `json:"usp"`
	UspOK *bool  `json:"usp_ok,omitempty"`
	Plane string `json:"plane"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

func operatorWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	apiArmed := strings.Contains(runND("api-public", "status"), "armed=true")
	pol := runND("svc-paths", "failover", "status")
	usersSP := strings.Contains(pol, `"users_to_sp_on_vless_down": true`) || strings.Contains(pol, `"users_to_sp_on_vless_down":true`)
	api, usp := "closed", "off"
	apiOK, uspOK := true, usersSP
	if apiArmed {
		api, apiOK = "OPEN", false
	}
	if usersSP {
		usp = "on"
	}
	stJSON := runND("svc-paths", "status")
	plane := "SP/PS ?"
	if strings.Contains(stJSON, `"peer_ping_ok":true`) || strings.Contains(stJSON, `"peer_ping_ok": true`) {
		plane = "plane ok"
	}
	title, hint := "Operator", "Full → classic toggles"
	if ru {
		title, hint = "Оператор", "Full — классические тумблеры"
		if apiArmed {
			api = "ОТКРЫТ"
		} else {
			api = "закрыт"
		}
	}
	b, err := json.Marshal(operatorWData{
		Title: title, API: api, APIOK: &apiOK, Usp: usp, UspOK: &uspOK, Plane: plane, Hint: hint, TS: time.Now().Unix(),
	})
	if err != nil {
		return "", err
	}
	if len(b) > 1024 {
		return "", fmt.Errorf("operator data %d", len(b))
	}
	return string(b), nil
}

func handleOperatorWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
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
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"operator"}`)
		reply(token, chat, q.MessageID, operatorHubHTML(), operatorKeyboard())
		return true
	case "svc":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"svc"}`)
		if tryReplyWidget(token, chat, q.MessageID, "svcpaths") {
			return true
		}
		out := runND("svc-paths", "status")
		reply(token, chat, q.MessageID, "🔗 <b>Service paths</b>\n<pre>"+esc(out)+"</pre>", operatorSubKeyboard())
		return true
	default:
		return false
	}
}

// --- addons ---

var addonsWidgetHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"/><style>` + cardWidgetCSS + `</style></head><body>
<h1 id="title">Addons</h1>
<div class="row"><span class="k">lampac</span><span class="v" data-f="lampac">—</span></div>
<div class="row"><span class="k">blocky</span><span class="v" data-f="blocky">—</span></div>
<div class="pre" id="body">—</div>
<div class="nav">
<button class="primary" data-act="addons:refresh">↻</button>
<button data-act="addons:full">Full</button>
<button data-act="addons:menu">Menu</button>
</div>
<div class="hint" id="hint">Update stays on classic screen</div>
<script>` + cardWidgetJS("addons") + `</script></body></html>`

type addonsWData struct {
	Title  string `json:"title"`
	Lampac string `json:"lampac"`
	Blocky string `json:"blocky"`
	Body   string `json:"body"`
	Hint   string `json:"hint,omitempty"`
	TS     int64  `json:"ts"`
}

func addonsWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Addons", "Full → manage / update"
	if ru {
		title, hint = "Аддоны", "Full — управление"
	}
	out := runND("addons", "status")
	if len(out) > 600 {
		out = out[:600] + "…"
	}
	lampac, blocky := "—", "—"
	low := strings.ToLower(out)
	if strings.Contains(low, "lampac") {
		lampac = "present"
		if strings.Contains(low, "lampac") && strings.Contains(low, "inactive") {
			lampac = "inactive"
		}
	}
	if strings.Contains(low, "blocky") {
		blocky = "present"
	}
	b, err := json.Marshal(addonsWData{Title: title, Lampac: lampac, Blocky: blocky, Body: strings.TrimSpace(out), Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		return "", fmt.Errorf("addons data %d", len(b))
	}
	return string(b), nil
}

func handleAddonsWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
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
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"addons"}`)
		reply(token, chat, q.MessageID, formatAddonsHTML(), addonsKeyboard())
		return true
	default:
		return false
	}
}

// --- svc paths ---

var svcpathsWidgetHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"/><style>` + cardWidgetCSS + `</style></head><body>
<h1 id="title">Service paths</h1>
<div class="pre" id="body">—</div>
<div class="nav">
<button class="primary" data-act="svcpaths:refresh">↻</button>
<button data-act="svcpaths:apply">Apply</button>
<button data-act="svcpaths:operator">Operator</button>
<button data-act="svcpaths:menu">Menu</button>
</div>
<div class="hint" id="hint">Apply runs svc-paths apply</div>
<script>` + cardWidgetJS("svcpaths") + `</script></body></html>`

type svcpathsWData struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

func svcpathsWidgetChatDataJSON() (string, error) {
	ru := getLang() != "en"
	title, hint := "Service paths", "Apply → classic result"
	if ru {
		title, hint = "Служебные каналы", "Apply — результат"
	}
	out := runND("svc-paths", "status")
	out = widgetTruncate(strings.TrimSpace(out), 700)
	b, err := json.Marshal(svcpathsWData{Title: title, Body: out, Hint: hint, TS: time.Now().Unix()})
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		return "", fmt.Errorf("svcpaths data %d", len(b))
	}
	return string(b), nil
}

func handleSvcpathsWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
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
	case "operator":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"operator"}`)
		if tryReplyWidget(token, chat, q.MessageID, "operator") {
			return true
		}
		reply(token, chat, q.MessageID, operatorHubHTML(), operatorKeyboard())
		return true
	case "apply":
		out := runND("svc-paths", "apply")
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		reply(token, chat, q.MessageID, "svc-paths apply\n<pre>"+esc(truncate(out, 2500))+"</pre>", operatorSubKeyboard())
		return true
	default:
		return false
	}
}

func init() {
	registerWidget(&widgetScreen{ID: "user", HTML: userCardWidgetHTML, BuildData: userCardChatDataJSON, HandleAct: handleUserCardAct})
	registerWidget(&widgetScreen{ID: "node", HTML: nodeCardWidgetHTML, BuildData: nodeCardChatDataJSON, HandleAct: handleNodeCardAct})
	registerWidget(&widgetScreen{ID: "operator", HTML: operatorWidgetHTML, BuildData: operatorWidgetChatDataJSON, HandleAct: handleOperatorWidgetAct})
	registerWidget(&widgetScreen{ID: "addons", HTML: addonsWidgetHTML, BuildData: addonsWidgetChatDataJSON, HandleAct: handleAddonsWidgetAct})
	registerWidget(&widgetScreen{ID: "svcpaths", HTML: svcpathsWidgetHTML, BuildData: svcpathsWidgetChatDataJSON, HandleAct: handleSvcpathsWidgetAct})
}
