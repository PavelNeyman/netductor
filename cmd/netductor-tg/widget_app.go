package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
)

// Single in-message SPA: one HTML, navigation via chat_data.s.
// Acts (≤64 bytes): r | b | m | g:<screen> | o:<n> | L:ru | L:en

var (
	appMu      sync.Mutex
	appScreen  = "home"
	appListIDs []string
)

// Labels live in HTML (not chat_data) so bilingual UI does not burn the 1 KiB budget.
const appWidgetBoot = `(function(){
var W=window.TelegramWidget;
var LAB={
en:{
  home:"Menu",status:"Status",channel:"Channel",versions:"Versions",secondary:"Secondary",
  probes:"Probes",fleet:"Fleet",users:"Users",nodes:"Nodes",devices:"Devices",sites:"Sites",
  pending:"Pending",ssh:"SSH",backup:"Backups",seclist:"Sec list",netmap:"Netmap",
  digest:"Digest",operator:"Operator",addons:"Addons",svcpaths:"Svc paths",
  lampac:"Lampac",canary:"Canary",sessions:"Sessions",audit:"Audit",actions:"Actions",
  routers:"Routers",menu:"Menu",back:"←",refresh:"↻",hint_home:"tap a section",
  hint_list:"# opens detail",nodata:"no data",err:"error",on:"on",off:"off",
  update:"update",primary:"primary",secondaries:"secondaries",uptodate:"up to date"
},
ru:{
  home:"Меню",status:"Статус",channel:"Канал",versions:"Версии",secondary:"Secondary",
  probes:"Probes",fleet:"Флот",users:"Пользователи",nodes:"Ноды",devices:"Устройства",sites:"Сайты",
  pending:"Pending",ssh:"SSH",backup:"Бэкапы",seclist:"Secondary",netmap:"Карта",
  digest:"Дайджест",operator:"Оператор",addons:"Аддоны",svcpaths:"Svc paths",
  lampac:"Lampac",canary:"Canary",sessions:"Сессии",audit:"Аудит",actions:"Действия",
  routers:"Роутеры",menu:"Меню",back:"←",refresh:"↻",hint_home:"выберите раздел",
  hint_list:"# — карточка",nodata:"нет данных",err:"ошибка",on:"вкл",off:"выкл",
  update:"обновление",primary:"primary",secondaries:"secondary",uptodate:"актуально"
}
};
function L(d){return (d&&d.L==="en")?"en":"ru";}
function tr(d,k){var lang=L(d);return (LAB[lang]&&LAB[lang][k])||(LAB.en[k])||k;}
function read(){
  try{if(W&&W.chatData){var x=W.chatData;return typeof x==="string"?JSON.parse(x):x;}}catch(e){}
  return null;
}
function paint(d){
  if(!d){d={s:"home",t:"Menu",b:"—",L:"ru"};}
  var title=document.getElementById("t");
  var body=document.getElementById("b");
  var hint=document.getElementById("h");
  var nav=document.getElementById("n");
  if(title)title.textContent=d.t||tr(d,d.s)||"Netductor";
  if(hint)hint.textContent=d.h||"";
  if(body){
    if(d.rows&&d.rows.length){
      var html="";
      for(var i=0;i<d.rows.length;i++){
        var r=d.rows[i];
        html+='<div class="row"><span class="k">'+(r[0]||"")+'</span><span class="v">'+(r[1]||"—")+'</span></div>';
      }
      body.innerHTML=html;
    }else if(d.lines){
      body.innerHTML='<div class="pre"></div>';
      body.firstChild.textContent=d.lines;
    }else{
      body.innerHTML='<div class="pre"></div>';
      body.firstChild.textContent=d.b||tr(d,"nodata");
    }
  }
  if(nav){
    nav.innerHTML="";
    function add(act,label,cls){
      var btn=document.createElement("button");
      btn.textContent=label;
      if(cls)btn.className=cls;
      btn.addEventListener("click",function(){send(act);});
      nav.appendChild(btn);
    }
    add("r",tr(d,"refresh"),"primary");
    if(d.s&&d.s!=="home")add("b",tr(d,"back"));
    if(d.s==="home"&&d.nav){
      for(var j=0;j<d.nav.length;j++){
        var id=d.nav[j];
        add("g:"+id,tr(d,id));
      }
    }else if(d.ids){
      var ids=String(d.ids).split(",");
      for(var k=0;k<ids.length&&k<12;k++){
        if(ids[k])add("o:"+(k+1),String(k+1));
      }
    }
    if(d.s!=="home")add("g:home",tr(d,"menu"));
    if(d.s==="home"){
      add("L:ru","RU");
      add("L:en","EN");
    }
  }
}
function send(act){
  act=String(act||"r");
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
if(W&&W.onEvent)W.onEvent("chatDataChanged",run);
})();`

func appWidgetHTML() string {
	return `<!DOCTYPE html><html><head><meta charset="utf-8"/>
<style>
body{margin:0;padding:10px;font:13px/1.35 system-ui,sans-serif;background:var(--tg-theme-bg-color,#0f1419);color:var(--tg-theme-text-color,#e7ecf3)}
h1{margin:0 0 8px;font-size:15px}
.row{display:flex;justify-content:space-between;gap:8px;margin:3px 0}
.k{color:var(--tg-theme-hint-color,#8b98a5);flex-shrink:0}
.v{font-weight:600;text-align:right;word-break:break-word}
.pre{margin:6px 0;padding:8px;background:var(--tg-theme-secondary-bg-color,#1c2733);border-radius:8px;font:11px/1.4 ui-monospace,monospace;white-space:pre-wrap;max-height:280px;overflow:hidden}
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

type appState struct {
	S     string      `json:"s"`
	T     string      `json:"t"`
	B     string      `json:"b,omitempty"`
	H     string      `json:"h,omitempty"`
	L     string      `json:"L,omitempty"`
	Rows  [][2]string `json:"rows,omitempty"`
	Lines string      `json:"lines,omitempty"`
	IDs   string      `json:"ids,omitempty"`
	Nav   []string    `json:"nav,omitempty"`
	TS    int64       `json:"ts"`
}

func appLang() string {
	if getLang() == "en" {
		return "en"
	}
	return "ru"
}

func appTitle(en, ru string) string {
	if appLang() == "en" {
		return en
	}
	return ru
}

func appMarshal(st appState) (string, error) {
	st.TS = time.Now().Unix()
	if st.L == "" {
		st.L = appLang()
	}
	b, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	if len(b) <= widgetChatDataMax {
		return string(b), nil
	}
	st.H = ""
	st.B = widgetTruncate(st.B, 180)
	st.Lines = widgetTruncate(st.Lines, 360)
	if len(st.Rows) > 8 {
		st.Rows = st.Rows[:8]
	}
	b, err = json.Marshal(st)
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		st.Lines = widgetTruncate(st.Lines, 200)
		st.Rows = nil
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

func appErr(screen, msg string) (string, error) {
	return appMarshal(appState{
		S: screen,
		T: appTitle("Error", "Ошибка"),
		B: widgetTruncate(msg, 400),
		H: screen,
	})
}

// appFromLegacyWidget reuses existing *WidgetChatDataJSON builders (title/body/rows-ish).
func appFromLegacyWidget(screen string, titleEN, titleRU string, build func() (string, error)) (string, error) {
	cd, err := build()
	if err != nil {
		// Prefer a readable short error over a hard fail (size overflows on busy hosts).
		msg := err.Error()
		if len(msg) > 180 {
			msg = msg[:180]
		}
		return appMarshal(appState{
			S: screen, T: appTitle(titleEN, titleRU),
			B: msg, H: appTitle("open classic menu if needed", "при необходимости — classic меню"),
		})
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(cd), &m); err != nil {
		return appMarshal(appState{S: screen, T: appTitle(titleEN, titleRU), B: widgetTruncate(cd, 500)})
	}
	title := appTitle(titleEN, titleRU)
	if t, ok := m["title"].(string); ok && t != "" {
		title = t
	}
	if t, ok := m["Title"].(string); ok && t != "" {
		title = t
	}
	body, _ := m["body"].(string)
	if body == "" {
		body, _ = m["Body"].(string)
	}
	rows := [][2]string{}
	// common short keys from various builders
	for _, k := range []string{"pri", "sec", "up", "face", "path", "cur", "tgt", "floor", "mig", "ver", "host", "nodes", "svc", "cpu", "mem"} {
		if v, ok := m[k]; ok && v != nil && fmt.Sprint(v) != "" {
			rows = append(rows, [2]string{k, fmt.Sprint(v)})
		}
	}
	st := appState{S: screen, T: title, Rows: rows, Lines: widgetTruncate(body, 500)}
	if hint, ok := m["hint"].(string); ok {
		st.H = hint
	}
	return appMarshal(st)
}

func appListFromItems(screen, titleEN, titleRU string, items []listWidgetItem) (string, error) {
	var lines []string
	var ids []string
	for _, it := range items {
		if len(ids) >= 12 {
			break
		}
		st := it.S
		lines = append(lines, fmt.Sprintf("%d %s %s", it.I, it.N, st))
		ids = append(ids, it.N)
	}
	appMu.Lock()
	appListIDs = ids
	appMu.Unlock()
	return appMarshal(appState{
		S: screen, T: appTitle(titleEN, titleRU),
		Lines: strings.Join(lines, "\n"),
		IDs:   strings.Join(ids, ","),
		H:     appTitle("# opens detail", "# — карточка"),
	})
}

func buildAppHome() (string, error) {
	return appMarshal(appState{
		S: "home",
		T: appTitle("Netductor", "Netductor"),
		H: appTitle("tap a section", "выберите раздел"),
		B: " ",
		Nav: []string{
			"status", "channel", "fleet", "digest", "versions",
			"users", "nodes", "devices", "sites",
			"secondary", "probes", "netmap",
			"operator", "addons", "backup", "ssh",
			"lampac", "sessions", "audit",
		},
	})
}

func buildAppStatus() (string, error) {
	d := buildStatusWidgetData()
	rows := [][2]string{
		{appTitle("version", "версия"), d.Ver},
		{appTitle("host", "хост"), d.Host},
		{appTitle("nodes", "ноды"), d.Nodes},
		{appTitle("services", "сервисы"), d.Svc},
		{appTitle("cpu/mem", "cpu/mem"), d.Res},
		{appTitle("secondary", "secondary"), d.Sec},
		{appTitle("migrate", "миграции"), d.Mig},
	}
	return appMarshal(appState{S: "status", T: d.Title, Rows: rows, H: d.Hint})
}

func buildAppFleet() (string, error) {
	d := fleet.Build()
	upd := appTitle("up to date", "актуально")
	if d.UpdateAvailable {
		upd = d.LatestRelease
	}
	on := 0
	for _, s := range d.Secondaries {
		if s.Online {
			on++
		}
	}
	rows := [][2]string{
		{appTitle("primary", "primary"), d.PrimaryVersion},
		{appTitle("update", "обновление"), upd},
		{appTitle("secondaries", "secondary"), fmt.Sprintf("%d/%d on", on, len(d.Secondaries))},
	}
	return appMarshal(appState{S: "fleet", T: appTitle("Fleet", "Флот"), Rows: rows})
}

func buildAppUsers() (string, error) {
	return appListFromItems("users", "Users", "Пользователи", collectUsersListItems())
}

func buildAppNodes() (string, error) {
	items := collectNodesListItems()
	rows := parseNodesList()
	var lines []string
	var ids []string
	for _, it := range items {
		if len(ids) >= 12 {
			break
		}
		id := it.N
		if it.I-1 < len(rows) {
			if rows[it.I-1].ID != "" {
				id = rows[it.I-1].ID
			}
		}
		lines = append(lines, fmt.Sprintf("%d %s %s", it.I, it.N, it.S))
		ids = append(ids, id)
	}
	appMu.Lock()
	appListIDs = ids
	appMu.Unlock()
	return appMarshal(appState{
		S: "nodes", T: appTitle("Nodes", "Ноды"),
		Lines: strings.Join(lines, "\n"), IDs: strings.Join(ids, ","),
		H: appTitle("# opens node", "# — нода"),
	})
}

func buildAppDevices() (string, error) {
	return appListFromItems("devices", "Devices", "Устройства", collectDevicesListItems())
}

func buildAppSites() (string, error) {
	return appListFromItems("sites", "Sites", "Сайты", collectSitesListItems())
}

func buildAppPending() (string, error) {
	return appListFromItems("pending", "Pending", "Pending", collectPendingListItems())
}

func buildAppSSH() (string, error) {
	return appListFromItems("ssh", "SSH hosts", "SSH хосты", collectSSHHostsItems())
}

func buildAppBackup() (string, error) {
	return appListFromItems("backup", "Backups", "Бэкапы", collectBackupListItems())
}

func buildAppSecondaryList() (string, error) {
	return appListFromItems("seclist", "Secondaries", "Secondary", collectSecondaryListItems())
}

func buildAppScreen(screen string) (string, error) {
	appMu.Lock()
	appScreen = screen
	appMu.Unlock()
	switch screen {
	case "home", "menu", "":
		appMu.Lock()
		appScreen = "home"
		appMu.Unlock()
		return buildAppHome()
	case "status":
		return buildAppStatus()
	case "channel":
		return appFromLegacyWidget("channel", "Channel", "Канал", channelWidgetChatDataJSON)
	case "versions":
		return appFromLegacyWidget("versions", "Versions", "Версии", versionsWidgetChatDataJSON)
	case "secondary":
		return appFromLegacyWidget("secondary", "Secondary", "Secondary", secondaryWidgetChatDataJSON)
	case "probes":
		return appFromLegacyWidget("probes", "Probes", "Probes", probesWidgetChatDataJSON)
	case "netmap":
		return appFromLegacyWidget("netmap", "Netmap", "Карта сети", netmapWidgetChatDataJSON)
	case "digest":
		return appFromLegacyWidget("digest", "Digest", "Дайджест", digestWidgetChatDataJSON)
	case "fleet":
		return buildAppFleet()
	case "users":
		return buildAppUsers()
	case "nodes":
		return buildAppNodes()
	case "devices":
		return buildAppDevices()
	case "sites":
		return buildAppSites()
	case "pending":
		return buildAppPending()
	case "ssh":
		return buildAppSSH()
	case "backup":
		return buildAppBackup()
	case "seclist":
		return buildAppSecondaryList()
	case "operator":
		return appFromLegacyWidget("operator", "Operator", "Оператор", operatorWidgetChatDataJSON)
	case "addons":
		return appFromLegacyWidget("addons", "Addons", "Аддоны", addonsWidgetChatDataJSON)
	case "svcpaths":
		return appFromLegacyWidget("svcpaths", "Svc paths", "Svc paths", svcpathsWidgetChatDataJSON)
	case "lampac":
		return appFromLegacyWidget("lampac", "Lampac", "Lampac", lampacWidgetChatDataJSON)
	case "canary":
		return appFromLegacyWidget("canary", "Canary", "Canary", canaryWidgetChatDataJSON)
	case "sessions":
		return appFromLegacyWidget("sessions", "Sessions", "Сессии", sessionsWidgetChatDataJSON)
	case "audit":
		return appFromLegacyWidget("audit", "Audit", "Аудит", auditWidgetChatDataJSON)
	case "actions":
		return appFromLegacyWidget("actions", "Actions", "Действия", actionsWidgetChatDataJSON)
	case "routers":
		return appFromLegacyWidget("routers", "Routers", "Роутеры", routersWidgetChatDataJSON)
	default:
		return buildAppHome()
	}
}

func appBuildData() (string, error) {
	appMu.Lock()
	s := appScreen
	appMu.Unlock()
	return buildAppScreen(s)
}

func appPushState(token string, q *widgetQuery, cd string) {
	answerWidgetQuery(token, q.ID, cd)
	if q.MessageID != 0 && q.Chat != nil {
		lastAppMu.Lock()
		lastAppMsgID[q.Chat.ID] = q.MessageID
		lastAppMu.Unlock()
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
	switch {
	case act == "r" || act == "refresh":
		appMu.Lock()
		s := appScreen
		appMu.Unlock()
		cd, err := buildAppScreen(s)
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	case act == "b" || act == "back" || act == "m" || act == "menu":
		cd, err := buildAppScreen("home")
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	case strings.HasPrefix(act, "L:"):
		lang := strings.TrimPrefix(act, "L:")
		if lang == "en" || lang == "ru" {
			setLang(lang)
		}
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
		appMu.Lock()
		ids := append([]string(nil), appListIDs...)
		screen := appScreen
		appMu.Unlock()
		if n < 1 || n > len(ids) {
			answerWidgetQuery(token, q.ID, `{"err":"idx"}`)
			return true
		}
		id := ids[n-1]
		chat := admin
		if q.Chat != nil {
			chat = q.Chat.ID
		}
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		switch screen {
		case "users":
			reply(token, chat, q.MessageID, formatUserHubHTML(id), userHubKeyboard(id))
		case "nodes":
			reply(token, chat, q.MessageID, formatNodeDetailHTML(id), nodeCardKeyboard(id))
		case "devices":
			reply(token, chat, q.MessageID, formatUserHubHTML(id), userHubKeyboard(id))
		case "sites":
			// best-effort classic sites list / detail
			reply(token, chat, q.MessageID, formatSitesHTML(), sitesListKeyboard())
		case "ssh":
			reply(token, chat, q.MessageID, formatSSHHostsHTML(), sshHostsKeyboard())
		default:
			// stay in widget; no classic card
		}
		return true
	default:
		return false
	}
}

func tryReplyAppWidget(token string, chat int64, msgID int) bool {
	return openAppScreen(token, chat, "home")
}

func init() {
	registerWidget(&widgetScreen{
		ID:        "app",
		HTML:      appWidgetHTML(),
		BuildData: appBuildData,
		HandleAct: handleAppAct,
	})
}
