package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
)

// Single in-message SPA: one HTML, all navigation via chat_data.s.
// Acts (≤64 bytes): r | g:<screen> | o:<n> | b | m

const appWidgetBoot = `(function(){
var W=window.TelegramWidget;
function read(){
  try{if(W&&W.chatData){var x=W.chatData;return typeof x==='string'?JSON.parse(x):x;}}catch(e){}
  return null;
}
function paint(d){
  if(!d){d={s:'home',t:'Menu',b:'no data'};}
  var title=document.getElementById('t');
  var body=document.getElementById('b');
  var hint=document.getElementById('h');
  var nav=document.getElementById('n');
  if(title)title.textContent=d.t||d.s||'Netductor';
  if(hint)hint.textContent=d.h||'';
  if(body){
    if(d.rows&&d.rows.length){
      var html='';
      for(var i=0;i<d.rows.length;i++){
        var r=d.rows[i];
        html+='<div class="row"><span class="k">'+(r[0]||'')+'</span><span class="v">'+(r[1]||'—')+'</span></div>';
      }
      body.innerHTML=html;
    }else if(d.lines){
      body.innerHTML='<div class="pre"></div>';
      body.firstChild.textContent=d.lines;
    }else{
      body.innerHTML='<div class="pre"></div>';
      body.firstChild.textContent=d.b||'—';
    }
  }
  if(nav){
    nav.innerHTML='';
    function add(act,label,cls){
      var btn=document.createElement('button');
      btn.textContent=label;
      if(cls)btn.className=cls;
      btn.addEventListener('click',function(){send(act);});
      nav.appendChild(btn);
    }
    add('r','↻','primary');
    if(d.s&&d.s!=='home')add('b','←');
    if(d.s==='home'&&d.nav){
      for(var j=0;j<d.nav.length;j++){
        var id=d.nav[j];
        add('g:'+id,id);
      }
    }else if(d.ids){
      var ids=String(d.ids).split(',');
      for(var k=0;k<ids.length&&k<12;k++){
        if(ids[k])add('o:'+(k+1),String(k+1));
      }
    }
    if(d.s!=='home')add('g:home','Menu');
  }
}
function send(act){
  act=String(act||'r');
  if(act.length>64)act=act.slice(0,64);
  if(!(W&&W.request))return;
  W.request(act).then(function(ans){
    if(ans&&ans.result){
      try{paint(JSON.parse(ans.result));}catch(e){}
    }
  },function(){});
}
function run(){paint(read());}
run();
if(W&&W.onEvent)W.onEvent('chatDataChanged',run);
})();`

func appWidgetHTML() string {
	return `<!DOCTYPE html><html><head><meta charset="utf-8"/>
<style>
body{margin:0;padding:10px;font:13px/1.35 system-ui,sans-serif;background:var(--tg-theme-bg-color,#0f1419);color:var(--tg-theme-text-color,#e7ecf3)}
h1{margin:0 0 8px;font-size:15px}
.row{display:flex;justify-content:space-between;gap:8px;margin:3px 0}
.k{color:var(--tg-theme-hint-color,#8b98a5);flex-shrink:0}
.v{font-weight:600;text-align:right;word-break:break-word}
.pre{margin:6px 0;padding:8px;background:var(--tg-theme-secondary-bg-color,#1c2733);border-radius:8px;font:11px/1.4 ui-monospace,monospace;white-space:pre-wrap;max-height:240px;overflow:hidden}
#n{display:flex;flex-wrap:wrap;gap:6px;margin-top:10px}
button{border:0;border-radius:8px;padding:7px 10px;font:12px system-ui;background:var(--tg-theme-secondary-bg-color,#1c2733);color:inherit}
button.primary{background:var(--tg-theme-button-color,#2a6df4);color:var(--tg-theme-button-text-color,#fff)}
#h{margin-top:8px;color:var(--tg-theme-hint-color,#8b98a5);font-size:11px}
</style></head><body>
<h1 id="t">Netductor</h1>
<div id="b"><div class="pre">—</div></div>
<div id="n"></div>
<div id="h"></div>
<script>` + appWidgetBoot + `</script></body></html>`
}

// app session: current screen for list open / back
var appScreen = "home"
var appListIDs []string // names for o:N on current list

type appState struct {
	S    string     `json:"s"`
	T    string     `json:"t"`
	B    string     `json:"b,omitempty"`
	H    string     `json:"h,omitempty"`
	Rows [][2]string `json:"rows,omitempty"`
	Lines string    `json:"lines,omitempty"`
	IDs  string     `json:"ids,omitempty"`
	Nav  []string   `json:"nav,omitempty"`
	TS   int64      `json:"ts"`
}

func appMarshal(st appState) (string, error) {
	st.TS = time.Now().Unix()
	b, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		// drop optional fields
		st.H = ""
		st.B = widgetTruncate(st.B, 200)
		st.Lines = widgetTruncate(st.Lines, 400)
		b, err = json.Marshal(st)
		if err != nil {
			return "", err
		}
		if len(b) > widgetChatDataMax {
			return "", fmt.Errorf("app chat_data %d", len(b))
		}
	}
	return string(b), nil
}

func buildAppHome() (string, error) {
	ru := getLang() != "en"
	t, h := "Netductor", "choose section"
	if ru {
		t, h = "Netductor", "раздел"
	}
	return appMarshal(appState{
		S: "home", T: t, H: h,
		Nav: []string{"status", "users", "nodes", "channel", "fleet", "secondary", "probes", "versions", "addons", "backup"},
		B: " ",
	})
}

func buildAppStatus() (string, error) {
	d := buildStatusWidgetData()
	rows := [][2]string{
		{"version", d.Ver},
		{"host", d.Host},
		{"nodes", d.Nodes},
		{"services", d.Svc},
		{"cpu/mem", d.Res},
		{"secondary", d.Sec},
		{"migrate", d.Mig},
	}
	return appMarshal(appState{S: "status", T: d.Title, Rows: rows, H: d.Hint})
}

func buildAppChannel() (string, error) {
	cd, err := channelWidgetChatDataJSON()
	if err != nil {
		return appMarshal(appState{S: "channel", T: "Channel", B: err.Error()})
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(cd), &m)
	rows := [][2]string{}
	for _, k := range []string{"pri", "sec", "up", "face", "path"} {
		if v, ok := m[k]; ok {
			rows = append(rows, [2]string{k, fmt.Sprint(v)})
		}
	}
	title := "Channel"
	if t, ok := m["title"].(string); ok && t != "" {
		title = t
	}
	return appMarshal(appState{S: "channel", T: title, Rows: rows})
}

func buildAppVersions() (string, error) {
	cd, err := versionsWidgetChatDataJSON()
	if err != nil {
		return appMarshal(appState{S: "versions", T: "Versions", B: err.Error()})
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(cd), &m)
	rows := [][2]string{}
	for _, k := range []string{"cur", "tgt", "floor", "mig"} {
		if v, ok := m[k]; ok {
			rows = append(rows, [2]string{k, fmt.Sprint(v)})
		}
	}
	body, _ := m["body"].(string)
	return appMarshal(appState{S: "versions", T: "Versions", Rows: rows, Lines: body})
}

func buildAppSecondary() (string, error) {
	cd, err := secondaryWidgetChatDataJSON()
	if err != nil {
		return appMarshal(appState{S: "secondary", T: "Secondary", B: err.Error()})
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(cd), &m)
	body, _ := m["body"].(string)
	title := "Secondary"
	if t, ok := m["title"].(string); ok {
		title = t
	}
	return appMarshal(appState{S: "secondary", T: title, Lines: widgetTruncate(body, 500)})
}

func buildAppProbes() (string, error) {
	cd, err := probesWidgetChatDataJSON()
	if err != nil {
		return appMarshal(appState{S: "probes", T: "Probes", B: err.Error()})
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(cd), &m)
	body, _ := m["body"].(string)
	return appMarshal(appState{S: "probes", T: "Probes", Lines: widgetTruncate(body, 500)})
}

func buildAppFleet() (string, error) {
	d := fleet.Build()
	rows := [][2]string{{"primary", d.PrimaryVersion}}
	if d.UpdateAvailable {
		rows = append(rows, [2]string{"update", d.LatestRelease})
	} else {
		rows = append(rows, [2]string{"update", "up to date"})
	}
	on := 0
	for _, s := range d.Secondaries {
		if s.Online {
			on++
		}
	}
	rows = append(rows, [2]string{"secondaries", fmt.Sprintf("%d/%d on", on, len(d.Secondaries))})
	return appMarshal(appState{S: "fleet", T: "Fleet", Rows: rows})
}

func buildAppUsers() (string, error) {
	items := collectUsersListItems()
	var lines []string
	var ids []string
	for _, it := range items {
		if it.I > 12 {
			break
		}
		lines = append(lines, fmt.Sprintf("%d %s %s", it.I, it.N, it.S))
		ids = append(ids, it.N)
	}
	appListIDs = ids
	return appMarshal(appState{
		S: "users", T: "Users", Lines: strings.Join(lines, "\n"),
		IDs: strings.Join(ids, ","), H: "# open card",
	})
}

func buildAppNodes() (string, error) {
	items := collectNodesListItems()
	var lines []string
	var ids []string
	for _, it := range items {
		if it.I > 12 {
			break
		}
		lines = append(lines, fmt.Sprintf("%d %s %s", it.I, it.N, it.S))
		// store node id via parseNodesList
		rows := parseNodesList()
		id := it.N
		if it.I-1 < len(rows) {
			id = rows[it.I-1].ID
			if id == "" {
				id = it.N
			}
		}
		ids = append(ids, id)
	}
	appListIDs = ids
	return appMarshal(appState{
		S: "nodes", T: "Nodes", Lines: strings.Join(lines, "\n"),
		IDs: strings.Join(ids, ","), H: "# open node",
	})
}

func buildAppAddons() (string, error) {
	out := runND("addons", "status")
	return appMarshal(appState{S: "addons", T: "Addons", Lines: widgetTruncate(strings.TrimSpace(out), 500)})
}

func buildAppBackup() (string, error) {
	out := runND("backup", "list")
	return appMarshal(appState{S: "backup", T: "Backups", Lines: widgetTruncate(strings.TrimSpace(out), 500), H: "manage via classic Full later"})
}

func buildAppScreen(screen string) (string, error) {
	appScreen = screen
	switch screen {
	case "home", "menu", "":
		appScreen = "home"
		return buildAppHome()
	case "status":
		return buildAppStatus()
	case "channel":
		return buildAppChannel()
	case "versions":
		return buildAppVersions()
	case "secondary":
		return buildAppSecondary()
	case "probes":
		return buildAppProbes()
	case "fleet":
		return buildAppFleet()
	case "users":
		return buildAppUsers()
	case "nodes":
		return buildAppNodes()
	case "addons":
		return buildAppAddons()
	case "backup":
		return buildAppBackup()
	default:
		return buildAppHome()
	}
}

func appBuildData() (string, error) {
	return buildAppScreen(appScreen)
}

func appPushState(token string, q *widgetQuery, cd string) {
	// answer carries full state for instant paint; edit persists for chatDataChanged
	answerWidgetQuery(token, q.ID, cd)
	if q.MessageID != 0 && q.Chat != nil {
		_, _ = apiPostWidget(token, "editMessageWidgetData", map[string]any{
			"chat_id": q.Chat.ID, "message_id": q.MessageID, "chat_data": cd,
		})
	}
}

func handleAppAct(token string, q *widgetQuery, admin int64, act string) bool {
	act = strings.TrimSpace(act)
	if act == "" {
		act = "r"
	}
	// acts are short: r, b, g:status, o:1
	switch {
	case act == "r" || act == "refresh":
		cd, err := buildAppScreen(appScreen)
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	case act == "b" || act == "back":
		cd, err := buildAppScreen("home")
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	case strings.HasPrefix(act, "g:"):
		screen := strings.TrimPrefix(act, "g:")
		cd, err := buildAppScreen(screen)
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	case strings.HasPrefix(act, "o:"):
		var n int
		fmt.Sscanf(strings.TrimPrefix(act, "o:"), "%d", &n)
		if n < 1 || n > len(appListIDs) {
			answerWidgetQuery(token, q.ID, `{"err":"idx"}`)
			return true
		}
		id := appListIDs[n-1]
		chat := admin
		if q.Chat != nil {
			chat = q.Chat.ID
		}
		// open classic detail outside SPA (prompts/destructive); keep app state
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		if appScreen == "users" {
			reply(token, chat, q.MessageID, formatUserHubHTML(id), userHubKeyboard(id))
		} else if appScreen == "nodes" {
			reply(token, chat, q.MessageID, formatNodeDetailHTML(id), nodeCardKeyboard(id))
		}
		return true
	case act == "m" || act == "menu":
		cd, err := buildAppScreen("home")
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	default:
		return false
	}
}

func tryReplyAppWidget(token string, chat int64, msgID int) bool {
	if !widgetsEnabled() {
		return false
	}
	appScreen = "home"
	// Prefer edit if msg is already app widget — we can't know; always send new app shell
	// so menu is one clean message. Callers for in-widget nav use handleAppAct.
	return sendWidgetScreen(token, chat, "app")
}

func init() {
	registerWidget(&widgetScreen{
		ID:        "app",
		HTML:      appWidgetHTML(),
		BuildData: appBuildData,
		HandleAct: handleAppAct,
	})
}
