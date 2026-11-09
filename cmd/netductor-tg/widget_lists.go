package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

// Phase 2 list widgets. Gate NETDUCTOR_TG_WIDGETS=1.
// Open item → classic card/handler (prompts & destructive stay rich).

const listWidgetPageSize = 12

const listWidgetHTMLShell = `<!DOCTYPE html>
<html><head><meta charset="utf-8"/>
<style>
body{margin:0;padding:10px;font:13px/1.35 system-ui,sans-serif;background:#0f1419;color:#e7ecf3}
h1{margin:0 0 8px;font-size:15px}
.pre{margin:6px 0;padding:8px;background:#1c2733;border-radius:8px;font:11px/1.4 ui-monospace,monospace;white-space:pre-wrap;max-height:220px;overflow:hidden}
.nums{display:flex;flex-wrap:wrap;gap:6px;margin:8px 0}
.nav{display:flex;flex-wrap:wrap;gap:6px;margin-top:8px}
button{border:0;border-radius:8px;padding:7px 10px;font:12px system-ui;background:#1c2733;color:#e7ecf3}
button.primary{background:#2a6df4}.hint{margin-top:6px;color:#8b98a5;font-size:11px}
</style></head><body>
<h1 id="title">List</h1>
<div class="pre" id="body">—</div>
<div class="nums" id="nums"></div>
<div class="nav" id="nav"></div>
<div class="hint" id="hint"></div>
<script>` + widgetBootJS + `
(function(){
var SCREEN="__SCREEN__";
function paint(d){if(!d)return;
var t=document.getElementById('title');if(t)t.textContent=d.title||SCREEN;
var b=document.getElementById('body');if(b)b.textContent=d.body||'—';
var h=document.getElementById('hint');if(h)h.textContent=d.hint||'';
var nums=document.getElementById('nums');nums.innerHTML='';
(d.items||[]).forEach(function(it){
var btn=document.createElement('button');btn.textContent=String(it.i);
btn.setAttribute('data-act',SCREEN+':open:'+it.i);
btn.addEventListener('click',send);nums.appendChild(btn);
});
var nav=document.getElementById('nav');nav.innerHTML='';
function add(act,label,cls){var btn=document.createElement('button');btn.textContent=label;if(cls)btn.className=cls;
btn.setAttribute('data-act',act);btn.addEventListener('click',send);nav.appendChild(btn);}
add(SCREEN+':refresh','↻ Refresh','primary');
if(d.page>0)add(SCREEN+':page:'+(d.page-1),'«');
if(d.more)add(SCREEN+':page:'+(d.page+1),'»');
add(SCREEN+':menu','Menu','');
}
function send(ev){var act=ev.currentTarget.getAttribute('data-act');
if(window.__ndSend)__ndSend(act);else if(window.TelegramWidget&&TelegramWidget.request)TelegramWidget.request(act);}
function data(){return window.__ndData?__ndData():null;}
paint(data());
if(window.TelegramWidget&&TelegramWidget.onEvent){
TelegramWidget.onEvent('chatDataChanged',function(){paint(data());});
}
})();
</script></body></html>`

func listWidgetHTML(screen string) string {
	s := strings.ReplaceAll(listWidgetHTMLShell, "__SCREEN__", screen)
	return strings.Replace(s, "<script>", "<script>"+widgetBootJS, 1)
}

type listWidgetItem struct {
	I int    `json:"i"`
	N string `json:"n"`
	S string `json:"s,omitempty"`
}

type listWidgetData struct {
	Title string           `json:"title"`
	Body  string           `json:"body"`
	Items []listWidgetItem `json:"items"`
	Page  int              `json:"page"`
	More  bool             `json:"more,omitempty"`
	Hint  string           `json:"hint,omitempty"`
	TS    int64            `json:"ts"`
}

func listWidgetChatData(title, hint string, all []listWidgetItem, page int) (string, error) {
	if page < 0 {
		page = 0
	}
	start := page * listWidgetPageSize
	if start > len(all) {
		start = len(all)
	}
	end := start + listWidgetPageSize
	if end > len(all) {
		end = len(all)
	}
	slice := all[start:end]
	// renumber display i as 1..n on page for open index within full list: keep global i
	var lines []string
	items := make([]listWidgetItem, 0, len(slice))
	for _, it := range slice {
		st := it.S
		if st != "" {
			lines = append(lines, fmt.Sprintf("%d %s %s", it.I, it.N, st))
		} else {
			lines = append(lines, fmt.Sprintf("%d %s", it.I, it.N))
		}
		items = append(items, it)
	}
	body := strings.Join(lines, "\n")
	if body == "" {
		body = "(empty)"
	}
	d := listWidgetData{
		Title: title, Body: body, Items: items, Page: page, More: end < len(all), Hint: hint, TS: time.Now().Unix(),
	}
	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		return "", fmt.Errorf("list chat_data %d > %d", len(b), widgetChatDataMax)
	}
	return string(b), nil
}

// --- collectors ---

func collectUsersListItems() []listWidgetItem {
	raw := runVPN("list")
	var all []listWidgetItem
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 1 {
			parts = strings.Fields(line)
		}
		if len(parts) < 1 || parts[0] == "" || parts[0] == "relay-uplink" || vpn.IsEdgeUser(parts[0]) || vpn.IsDeviceProfile(parts[0]) {
			continue
		}
		n++
		en := ""
		if len(parts) > 1 {
			en = parts[1]
		}
		st := "on"
		if en == "off" {
			st = "off"
		}
		all = append(all, listWidgetItem{I: n, N: parts[0], S: st})
		if n >= 40 {
			break
		}
	}
	return all
}

func collectDevicesListItems() []listWidgetItem {
	names := vpn.ListDeviceNames(false)
	var all []listWidgetItem
	for i, name := range names {
		if i >= 40 {
			break
		}
		all = append(all, listWidgetItem{I: i + 1, N: name})
	}
	return all
}

func collectNodesListItems() []listWidgetItem {
	rows := parseNodesList()
	var all []listWidgetItem
	for i, r := range rows {
		if i >= 40 {
			break
		}
		name := r.Host
		if name == "" {
			name = r.ID
		}
		st := r.Status
		if len(st) > 12 {
			st = st[:12]
		}
		all = append(all, listWidgetItem{I: i + 1, N: name, S: st})
	}
	return all
}

func collectSitesListItems() []listWidgetItem {
	// sites from formatSitesHTML source — edge sites list if available
	raw := runND("edge", "sites")
	var all []listWidgetItem
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// id or name first field
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		n++
		name := parts[0]
		st := ""
		if len(parts) > 1 {
			st = parts[1]
			if len(st) > 16 {
				st = st[:16]
			}
		}
		all = append(all, listWidgetItem{I: n, N: name, S: st})
		if n >= 40 {
			break
		}
	}
	return all
}

func collectPendingListItems() []listWidgetItem {
	raw := pendingText()
	var all []listWidgetItem
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		n++
		name := line
		if len(name) > 40 {
			name = name[:40] + "…"
		}
		all = append(all, listWidgetItem{I: n, N: name})
		if n >= 40 {
			break
		}
	}
	return all
}

func collectSSHHostsItems() []listWidgetItem {
	// best-effort from formatSSHHostsHTML underlying — run edge/ssh list
	raw := runND("ssh", "hosts")
	if strings.TrimSpace(raw) == "" {
		raw = runND("edge", "ssh-hosts")
	}
	var all []listWidgetItem
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		n++
		name := line
		if len(name) > 36 {
			name = name[:36] + "…"
		}
		all = append(all, listWidgetItem{I: n, N: name})
		if n >= 40 {
			break
		}
	}
	return all
}

func collectBackupListItems() []listWidgetItem {
	raw := runND("backup", "list")
	var all []listWidgetItem
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		n++
		name := line
		if len(name) > 36 {
			name = name[:36] + "…"
		}
		all = append(all, listWidgetItem{I: n, N: name})
		if n >= 40 {
			break
		}
	}
	return all
}

func collectSecondaryListItems() []listWidgetItem {
	d := fleet.Build()
	var all []listWidgetItem
	for i, s := range d.Secondaries {
		if i >= 40 {
			break
		}
		name := s.Name
		if name == "" {
			name = s.ID
		}
		st := "off"
		if s.Online {
			st = "on"
		}
		all = append(all, listWidgetItem{I: i + 1, N: name, S: st})
	}
	return all
}

// page memory per screen for widget (process-local; ok for single admin bot)
var listWidgetPage = map[string]int{}

func listPageGet(screen string) int { return listWidgetPage[screen] }
func listPageSet(screen string, p int) {
	if p < 0 {
		p = 0
	}
	listWidgetPage[screen] = p
}

func makeListBuildData(screen, titleEN, titleRU, hintEN, hintRU string, collect func() []listWidgetItem) func() (string, error) {
	return func() (string, error) {
		ru := getLang() != "en"
		title, hint := titleEN, hintEN
		if ru {
			title, hint = titleRU, hintRU
		}
		return listWidgetChatData(title, hint, collect(), listPageGet(screen))
	}
}

func handleListWidgetAct(screen string, collect func() []listWidgetItem, openFn func(token string, chat int64, msgID int, item listWidgetItem)) func(token string, q *widgetQuery, admin int64, act string) bool {
	return func(token string, q *widgetQuery, admin int64, act string) bool {
		chat := admin
		if q.Chat != nil {
			chat = q.Chat.ID
		}
		switch {
		case act == "refresh" || act == "":
			return false
		case act == "menu":
			widgetNavMenu(token, q, admin)
			return true
		case strings.HasPrefix(act, "page:"):
			var p int
			fmt.Sscanf(strings.TrimPrefix(act, "page:"), "%d", &p)
			listPageSet(screen, p)
			s := widgetRegistry[screen]
			if s != nil && s.BuildData != nil {
				cd, err := s.BuildData()
				if err != nil {
					answerWidgetQuery(token, q.ID, `{"err":"build"}`)
					return true
				}
				if q.MessageID != 0 && q.Chat != nil {
					_ = editWidgetScreenData(token, q.Chat.ID, q.MessageID, screen)
				}
				answerWidgetQuery(token, q.ID, `{"ok":true,"d":`+cd+`}`)
				return true
			}
			return false
		case strings.HasPrefix(act, "open:"):
			var idx int
			fmt.Sscanf(strings.TrimPrefix(act, "open:"), "%d", &idx)
			all := collect()
			var found *listWidgetItem
			for i := range all {
				if all[i].I == idx {
					found = &all[i]
					break
				}
			}
			if found == nil {
				answerWidgetQuery(token, q.ID, `{"err":"missing"}`)
				return true
			}
			answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"open"}`)
			if openFn != nil {
				openFn(token, chat, q.MessageID, *found)
			}
			return true
		default:
			return false
		}
	}
}

func init() {
	registerWidget(&widgetScreen{
		ID: "users", HTML: listWidgetHTML("users"),
		BuildData: makeListBuildData("users", "Users", "Users", "# open card · ↻ refresh", "# карточка · ↻ refresh", collectUsersListItems),
		HandleAct: handleListWidgetAct("users", collectUsersListItems, func(token string, chat int64, msgID int, item listWidgetItem) {
			if tryReplyUserWidget(token, chat, msgID, item.N) {
				return
			}
			reply(token, chat, msgID, formatUserHubHTML(item.N), userHubKeyboard(item.N))
		}),
	})
	registerWidget(&widgetScreen{
		ID: "devices", HTML: listWidgetHTML("devices"),
		BuildData: makeListBuildData("devices", "Devices", "Devices", "# open", "# открыть", collectDevicesListItems),
		HandleAct: handleListWidgetAct("devices", collectDevicesListItems, func(token string, chat int64, msgID int, item listWidgetItem) {
			// device profile card if exists
			reply(token, chat, msgID, formatUserHubHTML(item.N), userHubKeyboard(item.N))
		}),
	})
	registerWidget(&widgetScreen{
		ID: "nodes", HTML: listWidgetHTML("nodes"),
		BuildData: makeListBuildData("nodes", "Nodes", "Nodes", "# open node", "# узел", collectNodesListItems),
		HandleAct: handleListWidgetAct("nodes", collectNodesListItems, func(token string, chat int64, msgID int, item listWidgetItem) {
			// resolve id from parseNodesList by index
			rows := parseNodesList()
			id := item.N
			for _, r := range rows {
				host := r.Host
				if host == "" {
					host = r.ID
				}
				if host == item.N || r.ID == item.N {
					id = r.ID
					break
				}
			}
			if id != "" {
				if tryReplyNodeWidget(token, chat, msgID, id) {
					return
				}
				reply(token, chat, msgID, formatNodeDetailHTML(id), nodeCardKeyboard(id))
			}
		}),
	})
	registerWidget(&widgetScreen{
		ID: "sites", HTML: listWidgetHTML("sites"),
		BuildData: makeListBuildData("sites", "Sites", "Sites", "list", "список", collectSitesListItems),
		HandleAct: handleListWidgetAct("sites", collectSitesListItems, func(token string, chat int64, msgID int, item listWidgetItem) {
			reply(token, chat, msgID, T("sites_title")+string([]byte{10, 10})+formatSitesHTML(), sitesListKeyboard())
		}),
	})
	registerWidget(&widgetScreen{
		ID: "pending", HTML: listWidgetHTML("pending"),
		BuildData: makeListBuildData("pending", "Pending", "Pending", "edge pending", "ожидание", collectPendingListItems),
		HandleAct: handleListWidgetAct("pending", collectPendingListItems, func(token string, chat int64, msgID int, item listWidgetItem) {
			t := pendingText()
			reply(token, chat, msgID, formatPendingHTML(t), pendingKeyboard(t))
		}),
	})
	registerWidget(&widgetScreen{
		ID: "sshhosts", HTML: listWidgetHTML("sshhosts"),
		BuildData: makeListBuildData("sshhosts", "SSH hosts", "SSH hosts", "hosts", "хосты", collectSSHHostsItems),
		HandleAct: handleListWidgetAct("sshhosts", collectSSHHostsItems, func(token string, chat int64, msgID int, item listWidgetItem) {
			reply(token, chat, msgID, formatSSHHostsHTML(), sshHostsKeyboard())
		}),
	})
	registerWidget(&widgetScreen{
		ID: "backup", HTML: listWidgetHTML("backup"),
		BuildData: makeListBuildData("backup", "Backups", "Backups", "# → classic backup hub", "# → backup", collectBackupListItems),
		HandleAct: handleListWidgetAct("backup", collectBackupListItems, func(token string, chat int64, msgID int, item listWidgetItem) {
			handleBackupCB(token, chat, msgID, "m:backup")
		}),
	})
	registerWidget(&widgetScreen{
		ID: "seclist", HTML: listWidgetHTML("seclist"),
		BuildData: makeListBuildData("seclist", "Secondaries", "Secondaries", "# → secondary status", "# → secondary", collectSecondaryListItems),
		HandleAct: handleListWidgetAct("seclist", collectSecondaryListItems, func(token string, chat int64, msgID int, item listWidgetItem) {
			if tryReplyWidget(token, chat, msgID, "secondary") {
				return
			}
			out := runND("secondary", "status")
			reply(token, chat, msgID, "🖥 <b>Secondary</b>\n<pre>"+esc(truncate(out, 2000))+"</pre>", map[string]any{"inline_keyboard": [][]map[string]any{
				{btn("⬅️", "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
			}})
		}),
	})
	// fleet hub as short static list of sections
	registerWidget(&widgetScreen{
		ID: "fleet", HTML: listWidgetHTML("fleet"),
		BuildData: func() (string, error) {
			ru := getLang() != "en"
			title, hint := "Fleet", "open section #"
			if ru {
				title, hint = "Fleet", "раздел #"
			}
			items := []listWidgetItem{
				{1, "nodes", ""}, {2, "sites", ""}, {3, "routers", ""}, {4, "pending", ""}, {5, "secondaries", ""},
			}
			return listWidgetChatData(title, hint, items, 0)
		},
		HandleAct: handleListWidgetAct("fleet", func() []listWidgetItem {
			return []listWidgetItem{
				{1, "nodes", ""}, {2, "sites", ""}, {3, "routers", ""}, {4, "pending", ""}, {5, "secondaries", ""},
			}
		}, func(token string, chat int64, msgID int, item listWidgetItem) {
			switch item.N {
			case "nodes":
				if tryReplyWidget(token, chat, msgID, "nodes") {
					return
				}
				reply(token, chat, msgID, formatNodesListHTML(), nodesListKeyboard())
			case "sites":
				if tryReplyWidget(token, chat, msgID, "sites") {
					return
				}
				reply(token, chat, msgID, formatSitesHTML(), sitesListKeyboard())
			case "pending":
				if tryReplyWidget(token, chat, msgID, "pending") {
					return
				}
				t := pendingText()
				reply(token, chat, msgID, formatPendingHTML(t), pendingKeyboard(t))
			case "secondaries":
				if tryReplyWidget(token, chat, msgID, "seclist") {
					return
				}
				reply(token, chat, msgID, formatRelayListHTML(), relayKeyboard())
			default:
				reply(token, chat, msgID, fleetHubHTML(), fleetKeyboard())
			}
		}),
	})
}
