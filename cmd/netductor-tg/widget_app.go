package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/incident"
	gitstore "github.com/PavelNeyman/netductor/internal/git"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

// SPA menu with parity to classic hubs (Status / Users / Fleet / Tools / Operator).
// Acts ≤64 bytes: r | b | m | g:<screen> | o:<n> | a:<code> | y:<code> | L:ru | L:en

var (
	appMu      sync.Mutex
	appScreen  = "home"
	appListIDs []string
	appFocus   string // current user name or node id for card actions
	appRebootTarget string
	appBackupName   string
	appCompose      string
	appComposeMode  string // quota|rename|add
)

const appWidgetBoot = `(function(){
var W=window.TelegramWidget;
var LAB={
en:{
  home:"Menu",status:"Status",users:"Users",fleet:"Fleet",tools:"Tools",operator:"Operator",
  channel:"Channel",versions:"Versions",secondary:"Secondary",probes:"Probes",netmap:"Netmap",
  digest:"Digest",addons:"Addons",svcpaths:"Svc paths",lampac:"Lampac",canary:"Canary",
  sessions:"Sessions",audit:"Audit",actions:"Actions",routers:"Routers",nodes:"Nodes",
  devices:"Devices",sites:"Sites",pending:"Pending",ssh:"SSH",backup:"Backups",seclist:"Secondaries",
  edge:"Edge",media:"Media",data:"Data",adv:"Advanced",logs:"Logs",stack:"Stack",help:"Help",
  ulist:"List",incident:"Incident",reboot:"Reboot",dns:"DNS",git:"Git",certs:"Certs",fw:"Firewall",
  doctor:"Doctor",admin:"Admin",session:"Session",alerts:"Alerts",nvr:"NVR",cleanup:"Cleanup",
  api_tg:"API WAN",usp_tg:"Users→SP",vpnref:"Refresh links",
  ucard:"User",ncard:"Node",menu:"Menu",back:"←",refresh:"↻",hint_home:"choose a section",
  nodata:"no data",empty:"empty",
  en:"Enable",di:"Disable",lk:"Link",sb:"Sub",rv:"Revoke",y_rv:"Confirm revoke",
  up:"Upgrade",rb:"Reboot",jn:"Journal",y_rb:"Confirm reboot",
  q50:"50 GiB",q200:"200 GiB",q0:"∞ quota",rn:"Rename",add:"Add user",
  heal:"Heal",rbk:"Rollback",y_rbk:"Confirm rollback",wd:"Watchdog",
  y_rst:"Confirm restore",al_test:"Test alert",al_clr:"Clear alerts chat",
  qn:"Custom GiB",qr:"QR",kd:"→device",ku:"→user",sp:"Sub profile",rs:"Restart SB",gsall:"Sync all",sshkeys:"SSH keys",sess1:"1h",sess4:"4h",sess24:"24h",sess72:"72h",sessx:"Hours…",srev:"Revoke all",cy:"Canary",sk_on:"Protect ON",sk_off:"Protect OFF",sk_scan:"Scan keys",cert_ren:"Renew",fw_apply:"Apply FW",ag_st:"Status",ag_prev:"Preview",ag_scan:"Scan",nrn:"Rename node",mx:"Metrics",gbuild:"Build all",gmirror:"Mirror all",gjobs:"Ship jobs",gadd:"Add project",gmig:"Migrate",gpause:"Pause",gresume:"Resume",grm:"Remove job",el_1:"1h",el_4:"4h",el_24:"24h",el_72:"72h",el_ext:"+24h",es_st:"Storage",ep_st:"Policy",et_st:"Templates",eg_facts:"Facts",es_bak:"Backup",es_rst:"Restore",es_list:"List USB",es_mute:"Mute",et_on:"Tpl on",et_off:"Tpl off",et_dns:"DNS blocky",et_mode:"Mode full",glog:"Job log",eg_code:"Grant code",es_bak:"Backup",es_rst:"Restore",es_list:"List USB",es_mute:"Mute",et_on:"Tpl on",et_off:"Tpl off",et_dns:"DNS blocky",et_mode:"Mode full",glog:"Job log",eg_code:"Grant code",pad_ok:"OK",pad_bs:"⌫",pad_cl:"Clear",
  gsync:"Sync",gship:"Ship",eg_gr:"Grant",eg_rv:"Revoke guest",el_on:"LuCI on",el_off:"LuCI off",
  edgeguest:"Guest",edgeluci:"LuCI",
  st_hub:"Status hub",fl_hub:"Fleet hub",tl_hub:"Tools hub"
},
ru:{
  home:"Меню",status:"Статус",users:"Пользователи",fleet:"Флот",tools:"Инструменты",operator:"Оператор",
  channel:"Канал",versions:"Версии",secondary:"Secondary",probes:"Probes",netmap:"Карта",
  digest:"Дайджест",addons:"Аддоны",svcpaths:"Svc paths",lampac:"Lampac",canary:"Canary",
  sessions:"Сессии",audit:"Аудит",actions:"Действия",routers:"Роутеры",nodes:"Ноды",
  devices:"Устройства",sites:"Сайты",pending:"Pending",ssh:"SSH",backup:"Бэкапы",seclist:"Secondary",
  edge:"Edge",media:"Медиа",data:"Данные",adv:"Ещё",logs:"Логи",stack:"Стек",help:"Справка",
  ulist:"Список",incident:"Инцидент",reboot:"Reboot",dns:"DNS",git:"Git",certs:"Серты",fw:"Firewall",
  doctor:"Doctor",admin:"Админ",session:"Сессия",alerts:"Алерты",nvr:"NVR",cleanup:"Очистка",
  api_tg:"API WAN",usp_tg:"Users→SP",vpnref:"Обновить ссылки",
  ucard:"Пользователь",ncard:"Нода",menu:"Меню",back:"←",refresh:"↻",hint_home:"выберите раздел",
  nodata:"нет данных",empty:"пусто",
  en:"Вкл",di:"Выкл",lk:"Ссылка",sb:"Sub",rv:"Удалить",y_rv:"Подтвердить удаление",
  up:"Upgrade",rb:"Reboot",jn:"Journal",y_rb:"Подтвердить reboot",
  q50:"50 GiB",q200:"200 GiB",q0:"∞ квота",rn:"Имя",add:"Добавить",
  heal:"Heal",rbk:"Откат",y_rbk:"Подтвердить откат",wd:"Watchdog",
  y_rst:"Подтвердить restore",al_test:"Тест алерта",al_clr:"Сбросить alerts chat",
  qn:"Своя GiB",qr:"QR",kd:"→device",ku:"→user",sp:"Sub profile",rs:"Restart SB",gsall:"Sync all",sshkeys:"SSH keys",sess1:"1h",sess4:"4h",sess24:"24h",sess72:"72h",sessx:"Часы…",srev:"Отозвать все",cy:"Canary",sk_on:"Защита ON",sk_off:"Защита OFF",sk_scan:"Скан ключей",cert_ren:"Продлить",fw_apply:"Применить FW",ag_st:"Status",ag_prev:"Preview",ag_scan:"Scan",nrn:"Имя ноды",mx:"Metrics",gbuild:"Build all",gmirror:"Mirror all",gjobs:"Ship jobs",gadd:"Add project",gmig:"Migrate",gpause:"Pause",gresume:"Resume",grm:"Удалить job",el_1:"1h",el_4:"4h",el_24:"24h",el_72:"72h",el_ext:"+24h",es_st:"Storage",ep_st:"Policy",et_st:"Templates",eg_facts:"Facts",es_bak:"Backup",es_rst:"Restore",es_list:"List USB",es_mute:"Mute",et_on:"Tpl on",et_off:"Tpl off",et_dns:"DNS blocky",et_mode:"Mode full",glog:"Job log",eg_code:"Grant code",pad_ok:"OK",pad_bs:"⌫",pad_cl:"Сброс",
  gsync:"Sync",gship:"Ship",eg_gr:"Выдать",eg_rv:"Отозвать",el_on:"LuCI on",el_off:"LuCI off",
  edgeguest:"Guest",edgeluci:"LuCI",
  st_hub:"Статус",fl_hub:"Флот",tl_hub:"Инструменты"
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
      if(d.lines){html+='<div class="pre"></div>';}
      body.innerHTML=html;
      if(d.lines){body.querySelector(".pre").textContent=d.lines;}
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
      for(var j=0;j<d.nav.length;j++)add("g:"+d.nav[j],tr(d,d.nav[j]));
    }
    if(d.nav&&d.s!=="home"){
      for(var j2=0;j2<d.nav.length;j2++)add("g:"+d.nav[j2],tr(d,d.nav[j2]));
    }
    if(d.ids){
      var ids=String(d.ids).split(",");
      for(var k=0;k<ids.length&&k<12;k++){if(ids[k])add("o:"+(k+1),String(k+1));}
    }
    if(d.acts){
      var acts=String(d.acts).split(",");
      for(var a=0;a<acts.length;a++){
        if(!acts[a])continue;
        var lab=tr(d,acts[a]);
        if(acts[a].indexOf("y_")==0)lab=tr(d,acts[a]);
        var cls="";
        if(acts[a]==="rv"||acts[a]==="y_rv"||acts[a]==="rb"||acts[a]==="y_rb"||acts[a]==="y_rbk"||acts[a]==="y_rst")cls="danger";
        add(acts[a],lab,cls);
      }
    }
    if(d.btns&&d.btns.length){
      for(var bi=0;bi<d.btns.length;bi++){
        var pair=d.btns[bi];
        if(!pair||!pair[0])continue;
        add(String(pair[0]),String(pair[1]||pair[0]),"");
      }
    }
    // pad removed: free text via chat message
    if(d.s!=="home")add("g:home",tr(d,"menu"));
    if(d.s==="home"){add("L:ru","RU");add("L:en","EN");}
  }
}
function send(act){
  act=String(act||"r");
  if(act.length>64)act=act.slice(0,64);
  if(!(W&&W.request))return;
  W.request(act).then(function(ans){
    if(ans&&ans.result){try{paint(JSON.parse(ans.result));}catch(e){}}
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
.pre{margin:6px 0;padding:8px;background:var(--tg-theme-secondary-bg-color,#1c2733);border-radius:8px;font:11px/1.4 ui-monospace,monospace;white-space:pre-wrap;max-height:240px;overflow:hidden}
#n{display:flex;flex-wrap:wrap;gap:6px;margin-top:10px}
button{border:0;border-radius:8px;padding:7px 10px;font:12px system-ui;background:var(--tg-theme-secondary-bg-color,#1c2733);color:inherit}
button.primary{background:var(--tg-theme-button-color,#2a6df4);color:var(--tg-theme-button-text-color,#fff)}
button.danger{background:var(--tg-theme-destructive-text-color,#e53935);color:#fff}
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
	Acts  string      `json:"acts,omitempty"` // comma-separated act codes for buttons
	Btns  [][2]string `json:"btns,omitempty"` // dynamic [{act,label},...] for challenge codes etc.
	Draft string      `json:"draft,omitempty"`
	Pad   bool        `json:"pad,omitempty"` // show on-screen key pad (no free-text in widget profile)
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
	st.B = widgetTruncate(st.B, 160)
	st.Lines = widgetTruncate(st.Lines, 280)
	if len(st.Rows) > 6 {
		st.Rows = st.Rows[:6]
	}
	b, err = json.Marshal(st)
	if err != nil {
		return "", err
	}
	if len(b) > widgetChatDataMax {
		st.Lines = widgetTruncate(st.Lines, 120)
		st.Rows = st.Rows[:appMin(4, len(st.Rows))]
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

func appMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func appFromLegacyWidget(screen string, titleEN, titleRU string, build func() (string, error)) (string, error) {
	cd, err := build()
	if err != nil {
		msg := err.Error()
		if len(msg) > 180 {
			msg = msg[:180]
		}
		return appMarshal(appState{S: screen, T: appTitle(titleEN, titleRU), B: msg})
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(cd), &m); err != nil {
		return appMarshal(appState{S: screen, T: appTitle(titleEN, titleRU), B: widgetTruncate(cd, 400)})
	}
	title := appTitle(titleEN, titleRU)
	if t, ok := m["title"].(string); ok && t != "" {
		title = t
	}
	body, _ := m["body"].(string)
	if body == "" {
		body, _ = m["Body"].(string)
	}
	rows := [][2]string{}
	for _, k := range []string{"pri", "sec", "up", "face", "path", "cur", "tgt", "floor", "mig", "name", "online", "ver", "sb", "ip"} {
		if v, ok := m[k]; ok && v != nil && fmt.Sprint(v) != "" {
			rows = append(rows, [2]string{k, fmt.Sprint(v)})
		}
	}
	st := appState{S: screen, T: title, Rows: rows, Lines: widgetTruncate(body, 450)}
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
		lines = append(lines, fmt.Sprintf("%d %s %s", it.I, it.N, it.S))
		ids = append(ids, it.N)
	}
	appMu.Lock()
	appListIDs = ids
	appMu.Unlock()
	st := appState{
		S: screen, T: appTitle(titleEN, titleRU),
		IDs: strings.Join(ids, ","),
		H:   appTitle("# opens card", "# — карточка"),
	}
	if len(lines) == 0 {
		st.B = appTitle("empty", "пусто")
		st.H = ""
	} else {
		st.Lines = strings.Join(lines, "\n")
	}
	return appMarshal(st)
}

func buildAppHome() (string, error) {
	// Parity with mainKeyboard: Status, Users, Fleet, Tools, Operator
	return appMarshal(appState{
		S: "home",
		T: "Netductor",
		H: appTitle("same hubs as classic main menu", "те же хабы, что classic-меню"),
		B: " ",
		Nav: []string{"status", "users", "fleet", "tools", "operator", "help"},
	})
}

func buildAppStatusHub() (string, error) {
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
	// classic also: incident, node reboot — multi-step stay out of SPA for now
	return appMarshal(appState{
		S: "status", T: d.Title, Rows: rows, H: d.Hint,
		Nav: []string{"channel", "netmap", "actions", "sessions", "digest", "incident", "reboot"},
	})
}

func buildAppFleetHub() (string, error) {
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
	return appMarshal(appState{
		S: "fleet", T: appTitle("Fleet", "Флот"), Rows: rows,
		Nav: []string{"nodes", "routers", "sites", "secondary", "addons"},
	})
}

func buildAppToolsHub() (string, error) {
	// Classic Tools = Day2Groups (edge/media/data/adv) + Git/Updates/Logs/SSH keys rows.
	return appMarshal(appState{
		S: "tools", T: appTitle("Tools", "Инструменты"),
		B: appTitle("Edge · Media · Data · Advanced", "Edge · Медиа · Данные · Ещё"),
		H: appTitle("no Status/Users/Fleet duplicates", "без дублей главных хабов"),
		Nav: []string{"edge", "media", "data", "adv", "versions", "logs", "ssh", "sshkeys", "git"},
	})
}

func buildAppEdgeHub() (string, error) {
	return appMarshal(appState{
		S: "edge", T: appTitle("Edge", "Edge"),
		B: appTitle("routers · sites · pending", "роутеры · сайты · pending"),
		Nav: []string{"routers", "sites", "pending", "edgeguest", "edgeluci", "edgestor", "edgepol", "edgetpl"},
	})
}

func buildAppMediaHub() (string, error) {
	return appMarshal(appState{
		S: "media", T: appTitle("Media", "Медиа"),
		B: appTitle("lampac · canary", "lampac · canary"),
		Nav: []string{"lampac", "canary", "nvr", "aptguard"},
	})
}

func buildAppDataHub() (string, error) {
	return appMarshal(appState{
		S: "data", T: appTitle("Data", "Данные"),
		B: appTitle("backup · logs", "бэкапы · логи"),
		Nav: []string{"backup", "logs", "dns"},
	})
}

func buildAppAdvHub() (string, error) {
	return appMarshal(appState{
		S: "adv", T: appTitle("Advanced", "Ещё"),
		B: appTitle("probes · stack", "probes · стек"),
		Nav: []string{"probes", "stack", "doctor", "certs", "fw", "cleanup"},
	})
}

func buildAppLogs() (string, error) {
	out := runND("logs", "tail")
	if strings.TrimSpace(out) == "" {
		out = runND("status")
	}
	return appMarshal(appState{
		S: "logs", T: appTitle("Logs", "Логи"),
		Lines: widgetTruncate(strings.TrimSpace(out), 500),
		H: appTitle("refresh for latest", "обновите для свежих"),
	})
}

func buildAppStack() (string, error) {
	out := runND("stack", "status")
	return appMarshal(appState{
		S: "stack", T: appTitle("Stack", "Стек"),
		Lines: widgetTruncate(strings.TrimSpace(out), 420),
		Acts:  "heal,rbk,wd",
		H:     appTitle("heal · rollback · watchdog", "heal · откат · watchdog"),
	})
}


func buildAppRebootPick() (string, error) {
	var lines []string
	var ids []string
	lines = append(lines, "1 primary")
	ids = append(ids, "primary")
	for _, d := range secondary.List() {
		if len(ids) >= 10 {
			break
		}
		name := d.Name
		if name == "" {
			name = d.ID
		}
		lines = append(lines, fmt.Sprintf("%d %s", len(ids)+1, name))
		ids = append(ids, d.ID)
	}
	appMu.Lock()
	appListIDs = ids
	appMu.Unlock()
	return appMarshal(appState{
		S: "reboot", T: appTitle("Reboot", "Перезагрузка"),
		Lines: strings.Join(lines, "\n"),
		IDs:   strings.Join(ids, ","),
		H:     appTitle("# pick node → challenge codes", "# выбор → коды подтверждения"),
	})
}

func buildAppRebootChallenge(target, display string) (string, error) {
	code, err := notify.BeginRebootChallenge(target)
	if err != nil {
		return appMarshal(appState{S: "reboot", T: "Reboot", B: err.Error()})
	}
	codes := notify.DecoyCodes(code, 3)
	appMu.Lock()
	appRebootTarget = target
	appFocus = target
	appMu.Unlock()
	btns := make([][2]string, 0, 3)
	for _, c := range codes {
		// act rc:<code> ≤64
		btns = append(btns, [2]string{"rc:" + c, c})
	}
	return appMarshal(appState{
		S: "reboot", T: appTitle("Confirm reboot", "Подтверждение reboot"),
		Rows: [][2]string{{"node", display}, {"code", code}},
		H:    appTitle("tap the matching code", "нажмите кнопку с верным кодом"),
		Btns: btns,
	})
}


func buildAppGit() (string, error) {
	list, err := gitstore.LoadProjects()
	if err != nil {
		return appMarshal(appState{S: "git", T: "Git", B: err.Error()})
	}
	var lines []string
	var ids []string
	for i, pr := range list {
		if i >= 10 {
			break
		}
		lines = append(lines, fmt.Sprintf("%d %s", i+1, pr.Name))
		ids = append(ids, pr.Name)
	}
	appMu.Lock()
	appListIDs = ids
	appMu.Unlock()
	st := appState{
		S: "git", T: "Git",
		H: appTitle("# card · sync-all / ship via acts on card", "# карточка · sync/ship"),
		Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig",
	}
	if len(lines) == 0 {
		st.B = appTitle("no projects", "нет проектов")
	} else {
		st.Lines = strings.Join(lines, "\n")
		st.IDs = strings.Join(ids, ",")
	}
	return appMarshal(st)
}


func buildAppAlerts() (string, error) {
	st := notify.GetAlertsRoutingStatus()
	known := notify.ListKnownChats()
	var lines []string
	var ids []string
	cur := st.AlertsChatID
	if cur == "" {
		cur = appTitle("(none)", "(нет)")
	}
	for i, k := range known {
		if i >= 8 {
			break
		}
		lab := k.Title
		if lab == "" {
			lab = fmt.Sprintf("%d", k.ID)
		}
		lines = append(lines, fmt.Sprintf("%d %s", i+1, lab))
		ids = append(ids, fmt.Sprintf("%d", k.ID))
	}
	appMu.Lock()
	appListIDs = ids
	appMu.Unlock()
	return appMarshal(appState{
		S: "alerts", T: appTitle("Alerts", "Алерты"),
		Rows:  [][2]string{{"chat", cur}},
		Lines: strings.Join(lines, "\n"),
		IDs:   strings.Join(ids, ","),
		Acts:  "al_test,al_clr",
		H:     appTitle("# pick known chat", "# выбрать известный чат"),
	})
}

func buildAppEdgeGuest() (string, error) {
	out := strings.TrimSpace(runND("edge", "guest", "list"))
	// parse simple names if tab/lines
	var lines, ids []string
	for i, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || i >= 10 {
			continue
		}
		name := strings.Fields(line)
		if len(name) == 0 {
			continue
		}
		id := name[0]
		lines = append(lines, fmt.Sprintf("%d %s", len(ids)+1, line))
		ids = append(ids, id)
	}
	appMu.Lock()
	appListIDs = ids
	appMu.Unlock()
	st := appState{S: "edgeguest", T: "Edge guest", H: appTitle("# device actions", "# устройство")}
	if len(lines) == 0 {
		st.Lines = widgetTruncate(out, 400)
		if st.Lines == "" {
			st.B = appTitle("empty", "пусто")
		}
	} else {
		st.Lines = strings.Join(lines, "\n")
		st.IDs = strings.Join(ids, ",")
	}
	return appMarshal(st)
}

func buildAppEdgeLuci() (string, error) {
	out := strings.TrimSpace(runND("edge", "luci", "status"))
	return appMarshal(appState{
		S: "edgeluci", T: "Edge LuCI",
		Lines: widgetTruncate(out, 450),
		Acts:  "el_on,el_off",
		H:     appTitle("temp LuCI access", "временный доступ LuCI"),
	})
}


func buildAppIncident() (string, error) {
	ev := incident.ListEvents(20)
	var lines []string
	for _, e := range ev {
		lines = append(lines, widgetTruncate(fmt.Sprintf("%s %s", e.Kind, e.Text), 90))
		if len(lines) >= 12 {
			break
		}
	}
	st := appState{
		S: "incident", T: appTitle("Incident", "Инцидент"),
		Acts: "inc1h",
		H:    appTitle("1h pack → document in chat (auto-clean)", "пакет 1ч → документ в чат"),
	}
	if len(lines) == 0 {
		st.B = appTitle("no recent events", "нет событий")
	} else {
		st.Lines = strings.Join(lines, "\n")
	}
	return appMarshal(st)
}

func buildAppHelp() (string, error) {
	h := helpText()
	r := strings.NewReplacer("<b>", "", "</b>", "", "<i>", "", "</i>", "",
		"<br>", " ", "<br/>", " ", "<br />", " ")
	h = r.Replace(h)
	return appMarshal(appState{
		S: "help", T: appTitle("Help", "Справка"),
		Lines: widgetTruncate(h, 500),
	})
}


func buildAppOperatorHub() (string, error) {
	// Classic operatorHub: session, alerts, sessions, audit, stack, svc paths, admin, toggles, vpn refresh.
	apiArmed := strings.Contains(runND("api-public", "status"), "armed=true")
	pol := runND("svc-paths", "failover", "status")
	usersSP := strings.Contains(pol, `"users_to_sp_on_vless_down": true`) || strings.Contains(pol, `"users_to_sp_on_vless_down":true`)
	apiCell := "closed"
	if apiArmed {
		apiCell = "OPEN"
	}
	uspCell := "off"
	if usersSP {
		uspCell = "on"
	}
	st := appState{
		S: "operator", T: appTitle("Operator", "Оператор"),
		Rows: [][2]string{
			{"API WAN", apiCell},
			{"Users→SP", uspCell},
		},
		Nav:  []string{"session", "alerts", "sessions", "audit", "svcpaths", "stack", "admin"},
		Acts: "api_tg,usp_tg,vpnref",
		H:    appTitle("toggles + admin sections", "переключатели и разделы"),
	}
	return appMarshal(st)
}

func buildAppND(screen, en, ru string, args ...string) (string, error) {
	out := strings.TrimSpace(runND(args...))
	if out == "" {
		out = appTitle("(empty)", "(пусто)")
	}
	return appMarshal(appState{
		S: screen, T: appTitle(en, ru),
		Lines: widgetTruncate(out, 500),
	})
}

func buildAppUserCard(name string) (string, error) {
	if name == "" {
		return appMarshal(appState{S: "ucard", T: "?", B: appTitle("no user", "нет пользователя")})
	}
	appMu.Lock()
	appFocus = name
	appMu.Unlock()
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
			}
			break
		}
	}
	lim := vpn.SoftLimitGB(name)
	prof := vpn.SubProfileOf(name)
	kind := "user"
	if vpn.IsDeviceProfile(name) {
		kind = "device"
	}
	rows := [][2]string{
		{"status", en},
		{"limit", fmt.Sprintf("%.0f GiB", lim)},
		{"sub", prof},
		{"kind", kind},
	}
	return appMarshal(appState{
		S: "ucard", T: name, Rows: rows,
		H:    appTitle("actions below", "действия ниже"),
		Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,rv",
	})
}

func buildAppNodeCard(id string) (string, error) {
	if id == "" {
		return appMarshal(appState{S: "ncard", T: "?", B: appTitle("no node", "нет ноды")})
	}
	appMu.Lock()
	appFocus = id
	appMu.Unlock()
	rows := parseNodesList()
	host, role, st, ver := id, "", "", ""
	for _, r := range rows {
		if r.ID == id || r.Host == id {
			if r.Host != "" {
				host = r.Host
			}
			role = r.Role
			st = r.Status
			ver = r.Version
			if id == r.Host && r.ID != "" {
				id = r.ID
				appMu.Lock()
				appFocus = id
				appMu.Unlock()
			}
			break
		}
	}
	rs := [][2]string{
		{"id", id},
		{"host", host},
		{"role", displayRole(role)},
		{"status", st},
		{"version", ver},
	}
	return appMarshal(appState{
		S: "ncard", T: host, Rows: rs,
		H:    appTitle("node actions", "действия ноды"),
		Acts: "up,rb,jn,rs,nrn,mx",
	})
}

func buildAppUsersHub() (string, error) {
	return appMarshal(appState{
		S: "users", T: appTitle("Users", "Пользователи"),
		B: appTitle("VPN users · device profiles", "VPN пользователи · устройства"),
		Nav:  []string{"ulist", "devices"},
		Acts: "add",
		H:    appTitle("list · devices · add (reply with name)", "список · устройства · добавить (имя сообщением)"),
	})
}

func buildAppUsers() (string, error) {
	return appListFromItems("ulist", "Users", "Пользователи", collectUsersListItems())
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
		if it.I-1 < len(rows) && rows[it.I-1].ID != "" {
			id = rows[it.I-1].ID
		}
		lines = append(lines, fmt.Sprintf("%d %s %s", it.I, it.N, it.S))
		ids = append(ids, id)
	}
	appMu.Lock()
	appListIDs = ids
	appMu.Unlock()
	st := appState{S: "nodes", T: appTitle("Nodes", "Ноды"), IDs: strings.Join(ids, ","), H: appTitle("# opens node", "# — нода")}
	if len(lines) == 0 {
		st.B = appTitle("empty", "пусто")
		st.H = ""
	} else {
		st.Lines = strings.Join(lines, "\n")
	}
	return appMarshal(st)
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
		return buildAppStatusHub()
	case "fleet":
		return buildAppFleetHub()
	case "tools":
		return buildAppToolsHub()
	case "edge":
		return buildAppEdgeHub()
	case "media":
		return buildAppMediaHub()
	case "data":
		return buildAppDataHub()
	case "adv":
		return buildAppAdvHub()
	case "logs":
		return buildAppLogs()
	case "stack":
		return buildAppStack()
	case "help":
		return buildAppHelp()
	case "operator":
		return buildAppOperatorHub()
	case "users":
		return buildAppUsersHub()
	case "ulist":
		return buildAppUsers()
	case "nodes":
		return buildAppNodes()
	case "incident":
		return buildAppIncident()
	case "sshkeys":
		return buildAppSSHKeys()
	case "reboot":
		return buildAppRebootPick()
	case "dns":
		return buildAppND("dns", "DNS", "DNS", "dns", "status")
	case "git":
		return buildAppGit()
	case "certs":
		return buildAppCerts()
	case "fw":
		return buildAppFW()
	case "doctor":
		return buildAppND("doctor", "Doctor", "Doctor", "doctor")
	case "cleanup":
		return buildAppND("cleanup", "Cleanup", "Очистка", "cleanup", "status")
	case "admin":
		return buildAppAdmin()
	case "session":
		return buildAppSession()
	case "alerts":
		return buildAppAlerts()
	case "edgestor":
		return buildAppEdgeStorage()
	case "edgepol":
		return appMarshal(appState{S: "edgepol", T: "Edge policy", Acts: "ep_st", H: appTitle("status", "статус")})
	case "edgetpl":
		return buildAppEdgeTemplates()
	case "edgeguest":
		return buildAppEdgeGuest()
	case "edgeluci":
		return buildAppEdgeLuciFull()
	case "nvr":
		return buildAppND("nvr", "NVR", "NVR", "nvr", "status")
	case "devices":
		return appListFromItems("devices", "Devices", "Устройства", collectDevicesListItems())
	case "sites":
		return appListFromItems("sites", "Sites", "Сайты", collectSitesListItems())
	case "pending":
		return appListFromItems("pending", "Pending", "Pending", collectPendingListItems())
	case "ssh":
		return appListFromItems("ssh", "SSH hosts", "SSH хосты", collectSSHHostsItems())
	case "backup":
		return appListFromItems("backup", "Backups", "Бэкапы", collectBackupListItems())
	case "seclist":
		return appListFromItems("seclist", "Secondaries", "Secondary", collectSecondaryListItems())
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
	case "addons":
		return appFromLegacyWidget("addons", "Addons", "Аддоны", addonsWidgetChatDataJSON)
	case "svcpaths":
		return appFromLegacyWidget("svcpaths", "Svc paths", "Svc paths", svcpathsWidgetChatDataJSON)
	case "lampac":
		return appFromLegacyWidget("lampac", "Lampac", "Lampac", lampacWidgetChatDataJSON)
	case "aptguard":
		return buildAppAptguard()
	case "canary":
		return buildAppCanary()
	case "sessions":
		out := strings.TrimSpace(runND("session", "list"))
		if out == "" {
			out = runVPN("session", "list")
		}
		return appMarshal(appState{S: "sessions", T: appTitle("Sessions", "Сессии"), Lines: widgetTruncate(out, 400), Acts: "srev"})
	case "audit":
		return appFromLegacyWidget("audit", "Audit", "Аудит", auditWidgetChatDataJSON)
	case "actions":
		return appFromLegacyWidget("actions", "Actions", "Действия", actionsWidgetChatDataJSON)
	case "routers":
		return appFromLegacyWidget("routers", "Routers", "Роутеры", routersWidgetChatDataJSON)
	case "ucard":
		appMu.Lock()
		n := appFocus
		appMu.Unlock()
		return buildAppUserCard(n)
	case "ncard":
		appMu.Lock()
		n := appFocus
		appMu.Unlock()
		return buildAppNodeCard(n)
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

	// language
	if strings.HasPrefix(act, "L:") {
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
	}

	// navigation
	if act == "r" || act == "refresh" {
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
	}
	if act == "b" || act == "back" {
		appMu.Lock()
		s := appScreen
		appMu.Unlock()
		parent := "home"
		switch s {
		case "ucard", "ulist", "devices":
			parent = "users"
		case "ncard":
			parent = "nodes"
		case "channel", "netmap", "digest", "actions", "incident", "reboot":
			parent = "status"
		case "nodes", "secondary", "addons", "seclist":
			parent = "fleet"
		case "routers", "sites", "pending", "edgeguest", "edgeluci", "edgestor", "edgepol", "edgetpl":
			parent = "edge"
		case "lampac", "canary", "nvr", "aptguard":
			parent = "media"
		case "backup", "logs", "dns":
			parent = "data"
		case "probes", "stack", "doctor", "certs", "fw", "cleanup":
			parent = "adv"
		case "edge", "media", "data", "adv", "versions", "ssh", "sshkeys", "git":
			parent = "tools"
		case "svcpaths", "session", "alerts", "admin", "audit", "sessions":
			parent = "operator"
		case "users", "status", "fleet", "tools", "operator", "help":
			parent = "home"
		}
		cd, err := buildAppScreen(parent)
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	}
	if act == "m" || act == "menu" || strings.HasPrefix(act, "g:") {
		screen := "home"
		if strings.HasPrefix(act, "g:") {
			screen = strings.TrimPrefix(act, "g:")
		}
		cd, err := buildAppScreen(screen)
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	}

	// open list item
	if strings.HasPrefix(act, "o:") {
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
		var cd string
		var err error
		switch screen {
		case "users", "ulist", "devices":
			cd, err = buildAppUserCard(id)
		case "nodes":
			cd, err = buildAppNodeCard(id)
		case "reboot":
			disp := id
			if id == "primary" {
				disp = "primary"
			}
			cd, err = buildAppRebootChallenge(id, disp)
		case "backup":
			appMu.Lock()
			appBackupName = id
			appMu.Unlock()
			cd, err = appMarshal(appState{
				S: "backup", T: appTitle("Restore?", "Restore?"),
				B: id, Acts: "y_rst",
				H: appTitle("confirm restore from backup", "подтвердите восстановление"),
			})
		case "git":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			cd, err = appMarshal(appState{
				S: "git", T: id,
				B: appTitle("project actions", "действия проекта"),
				Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig",
				H: id,
			})
		case "alerts":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			// pick known chat by index id stored as chat id string
			_ = notify.SetAlertsChatID(id)
			cd, err = appMarshal(appState{
				S: "alerts", T: appTitle("Alerts", "Алерты"),
				B: appTitle("bound to", "привязано") + " " + id,
				Acts: "al_test,al_clr,al_pick",
			})
		case "edgeguest":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			cd, err = appMarshal(appState{
				S: "edgeguest", T: id,
				B: appTitle("guest device", "guest устройство"),
				Acts: "eg_gr,eg_rv,eg_facts,eg_code",
			})
		case "edgestor":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			cd, err = appMarshal(appState{
				S: "edgestor", T: id,
				Acts: "es_bak,es_rst,es_list,es_mute",
				H: id,
			})
		case "edgetpl":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			cd, err = appMarshal(appState{
				S: "edgetpl", T: id,
				Acts: "et_on,et_off,et_dns,et_mode",
				H: id,
			})
		case "canary":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			_, _ = vpn.ToggleCanary(id)
			cd, err = buildAppCanary()
		case "gjobs":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			cd, err = appMarshal(appState{
				S: "gjobs", T: id,
				Acts: "gpause,gresume,grm,glog",
				H: id,
			})
		default:
			cd, err = buildAppScreen(screen)
		}
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	}

	// user/node actions
	appMu.Lock()
	focus := appFocus
	appMu.Unlock()

	switch act {
	case "api_tg":
		st := runND("api-public", "status")
		if strings.Contains(st, "armed=true") {
			_ = runND("api-public", "disarm")
		} else {
			_ = runND("api-public", "arm", "--ttl", "15m")
		}
		cd, _ := buildAppOperatorHub()
		appPushState(token, q, cd)
		return true
	case "usp_tg":
		pol := runND("svc-paths", "failover", "status")
		on := strings.Contains(pol, `"users_to_sp_on_vless_down": true`) || strings.Contains(pol, `"users_to_sp_on_vless_down":true`)
		if on {
			_ = runND("svc-paths", "failover", "set-users-sp", "off")
		} else {
			_ = runND("svc-paths", "failover", "set-users-sp", "on")
		}
		cd, _ := buildAppOperatorHub()
		appPushState(token, q, cd)
		return true
	case "vpnref":
		out := runND("vpn", "refresh-links")
		cd, _ := appMarshal(appState{
			S: "operator", T: appTitle("Operator", "Оператор"),
			Lines: widgetTruncate(out, 400),
			Nav:  []string{"session", "alerts", "sessions", "audit", "svcpaths", "stack", "admin"},
			Acts: "api_tg,usp_tg,vpnref",
		})
		appPushState(token, q, cd)
		return true
	case "qn":
		if focus == "" || q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_quota:"+focus,
			appTitle("Send soft limit in GiB (number, 0 = unlimited) for ", "Пришлите лимит GiB (число, 0 = ∞) для ")+"<code>"+esc(focus)+"</code>:")
		cd, _ := appMarshal(appState{
			S: "ucard", T: focus,
			B: appTitle("waiting for GiB number in chat…", "ждём число GiB в чате…"),
			H: appTitle("your message will be deleted after apply", "сообщение удалится после применения"),
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,rv",
		})
		appPushState(token, q, cd)
		return true

	case "q50", "q200", "q0":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		gb := 50.0
		if act == "q200" {
			gb = 200
		}
		if act == "q0" {
			gb = 0
		}
		_ = vpn.SetSoftLimitGB(focus, gb)
		cd, _ := buildAppUserCard(focus)
		appPushState(token, q, cd)
		return true
	case "rn":
		if focus == "" || q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_rename_new:"+focus, Tf("rename_new_hint", focus))
		cd, _ := appMarshal(appState{
			S: "ucard", T: focus,
			B: appTitle("waiting for new name in chat…", "ждём новое имя в чате…"),
			H: appTitle("message will be deleted after rename", "сообщение удалится после переименования"),
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,rv",
		})
		appPushState(token, q, cd)
		return true
	case "add":
		if q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nochat"}`)
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_vpn_add_name", T("add_prompt"))
		cd, _ := appMarshal(appState{
			S: "users", T: appTitle("Users", "Пользователи"),
			B: appTitle("waiting for name in chat…", "ждём имя в чате…"),
			H: appTitle("message will be deleted after add", "сообщение удалится после добавления"),
			Nav: []string{"ulist", "devices"}, Acts: "add",
		})
		appPushState(token, q, cd)
		return true
	case "gsync":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out, err := gitstore.SyncProject(focus)
		msg := out
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "git", T: focus, Lines: widgetTruncate(msg, 400), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig"})
		appPushState(token, q, cd)
		return true
	case "gship":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_, logOut, err := gitstore.ShipProject(gitstore.ShipOpts{Project: focus})
		msg := logOut
		if err != nil {
			msg = err.Error() + "\n" + logOut
		}
		cd, _ := appMarshal(appState{S: "git", T: focus, Lines: widgetTruncate(msg, 400), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig"})
		appPushState(token, q, cd)
		return true
	case "eg_gr":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := runND("edge", "guest", "grant", focus)
		cd, _ := appMarshal(appState{S: "edgeguest", T: focus, Lines: widgetTruncate(out, 400), Acts: "eg_gr,eg_rv"})
		appPushState(token, q, cd)
		return true
	case "eg_rv":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := runND("edge", "guest", "revoke", focus)
		cd, _ := appMarshal(appState{S: "edgeguest", T: focus, Lines: widgetTruncate(out, 400), Acts: "eg_gr,eg_rv"})
		appPushState(token, q, cd)
		return true
	case "el_on":
		out := runND("edge", "luci", "on", "4h")
		cd, _ := appMarshal(appState{S: "edgeluci", T: "Edge LuCI", Lines: widgetTruncate(out, 400), Acts: "el_on,el_off"})
		appPushState(token, q, cd)
		return true
	case "el_off":
		out := runND("edge", "luci", "off")
		cd, _ := appMarshal(appState{S: "edgeluci", T: "Edge LuCI", Lines: widgetTruncate(out, 400), Acts: "el_on,el_off"})
		appPushState(token, q, cd)
		return true

	case "heal":
		out := runND("stack", "heal")
		cd, _ := appMarshal(appState{S: "stack", T: "Stack", Lines: widgetTruncate(out, 450), Acts: "heal,rbk,wd"})
		appPushState(token, q, cd)
		return true
	case "wd":
		out := runND("stack", "watchdog")
		cd, _ := appMarshal(appState{S: "stack", T: "Stack", Lines: widgetTruncate(out, 450), Acts: "heal,rbk,wd"})
		appPushState(token, q, cd)
		return true
	case "rbk":
		cd, _ := appMarshal(appState{
			S: "stack", T: appTitle("Rollback?", "Откат?"),
			B: appTitle("restore last-good stack", "восстановить last-good"),
			Acts: "y_rbk",
		})
		appPushState(token, q, cd)
		return true
	case "y_rbk":
		out := runND("stack", "rollback")
		cd, _ := appMarshal(appState{S: "stack", T: "Stack", Lines: widgetTruncate(out, 450), Acts: "heal,rbk,wd"})
		appPushState(token, q, cd)
		return true
	case "y_rst":
		appMu.Lock()
		name := appBackupName
		appMu.Unlock()
		if name == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nobak"}`)
			return true
		}
		out := runND("backup", "restore", name)
		cd, _ := appMarshal(appState{
			S: "backup", T: appTitle("Restore", "Restore"),
			Lines: widgetTruncate(out, 450),
		})
		appPushState(token, q, cd)
		return true
	case "al_test":
		out := runND("alerts", "test")
		if strings.TrimSpace(out) == "" {
			out = runND("notify", "test")
		}
		cd, _ := appMarshal(appState{S: "alerts", T: appTitle("Alerts", "Алерты"), Lines: widgetTruncate(out, 400), Acts: "al_test,al_clr"})
		appPushState(token, q, cd)
		return true
	case "al_clr":
		_ = notify.ClearAlertsChatID()
		cd, _ := buildAppAlerts()
		appPushState(token, q, cd)
		return true
	case "kd":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_ = vpn.SetKind(focus, "device")
		cd, _ := buildAppUserCard(focus)
		appPushState(token, q, cd)
		return true
	case "ku":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_ = vpn.SetKind(focus, "user")
		cd, _ := buildAppUserCard(focus)
		appPushState(token, q, cd)
		return true
	case "sp":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		next, err := vpn.CycleSubProfile(focus)
		msg := next
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{
			S: "ucard", T: focus, B: "sub → " + msg,
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,rv",
		})
		appPushState(token, q, cd)
		return true
	case "rs":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := restartSingBox(focus)
		cd, _ := appMarshal(appState{S: "ncard", T: focus, Lines: widgetTruncate(out, 400), Acts: "up,rb,jn,rs,nrn,mx"})
		appPushState(token, q, cd)
		return true
	case "gsall":
		list, err := gitstore.LoadProjects()
		var b strings.Builder
		if err != nil {
			b.WriteString(err.Error())
		} else {
			for _, pr := range list {
				out, e := gitstore.SyncProject(pr.Name)
				if e != nil {
					b.WriteString(pr.Name + ": " + e.Error() + "\n")
				} else {
					b.WriteString(pr.Name + ": ok\n")
				}
				_ = out
			}
		}
		cd, _ := appMarshal(appState{S: "git", T: "Git", Lines: widgetTruncate(b.String(), 450), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig"})
		appPushState(token, q, cd)
		return true
	case "inc1h":
		if q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nochat"}`)
			return true
		}
		go func(token string, chat int64) {
			_, arch, err := incident.Collect(1)
			if err != nil {
				mid := sendPlainHTML(token, chat, "❌ incident: "+esc(err.Error()))
				if mid > 0 {
					go scheduleDelete(token, chat, mid, 60)
				}
				return
			}
			mid, err := sendPhotoFileID(token, chat, arch, "Incident 1h (doc)", nil)
			// sendDocument better
			_ = mid
			if err := sendDocumentFile(token, chat, arch, "Incident pack 1h", nil); err != nil {
				mid := sendPlainHTML(token, chat, "✅ "+esc(arch))
				if mid > 0 {
					go scheduleDelete(token, chat, mid, 120)
				}
				return
			}
			// cannot easily get doc msg id from sendDocumentFile — leave 2 min note
			mid = sendPlainHTML(token, chat, appTitle("incident pack sent (delete manually if needed)", "пакет инцидента отправлен"))
			if mid > 0 {
				go scheduleDelete(token, chat, mid, 60)
			}
		}(token, q.Chat.ID)
		cd, _ := appMarshal(appState{
			S: "incident", T: appTitle("Incident", "Инцидент"),
			B: appTitle("collecting 1h pack…", "собираю пакет 1ч…"),
			Acts: "inc1h",
		})
		appPushState(token, q, cd)
		return true

	case "en":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_ = runVPN("enable", focus)
		cd, _ := buildAppUserCard(focus)
		appPushState(token, q, cd)
		return true
	case "di":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_ = runVPN("disable", focus)
		cd, _ := buildAppUserCard(focus)
		appPushState(token, q, cd)
		return true
	case "qr":
		if focus == "" || q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		uri := strings.TrimSpace(accessPayload(focus, "vless"))
		if uri == "" {
			cd, _ := appMarshal(appState{S: "ucard", T: focus, B: appTitle("empty link", "пустая ссылка"), Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,rv"})
			appPushState(token, q, cd)
			return true
		}
		dir := filepath.Join("/etc/netductor/clients", focus)
		_ = os.MkdirAll(dir, 0o700)
		qrPath := filepath.Join(dir, "qr-vless.png")
		p := ensureQRFile(qrPath, uri)
		cap := "📱 QR <code>" + esc(focus) + "</code> · " + appTitle("auto-delete 60s", "удалится через 60с")
		if p != "" {
			sendEphemeralPhoto(token, q.Chat.ID, p, cap, 60)
		} else {
			mid := sendPlainHTML(token, q.Chat.ID, cap+"\n<pre>"+esc(widgetTruncate(uri, 500))+"</pre>")
			if mid > 0 {
				go scheduleDelete(token, q.Chat.ID, mid, 60)
			}
		}
		cd, _ := buildAppUserCard(focus)
		appPushState(token, q, cd)
		return true
	case "lk", "sb":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		mode := "vless"
		if act == "sb" {
			mode = "sub"
		}
		link := strings.TrimSpace(accessPayload(focus, mode))
		if link == "" {
			link = appTitle("(empty link)", "(пустая ссылка)")
		}
		if len(link) > 400 {
			link = link[:400] + "…"
		}
		cd, _ := appMarshal(appState{
			S: "ucard", T: focus,
			Rows:  [][2]string{{"mode", mode}},
			Lines: link,
			H:    appTitle("link / sub URI", "ссылка / sub"),
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,rv",
		})
		appPushState(token, q, cd)
		return true
	case "rv":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		cd, _ := appMarshal(appState{
			S: "ucard", T: appTitle("Revoke?", "Удалить?"),
			B:    focus,
			H:    appTitle("confirm destructive action", "подтвердите удаление"),
			Acts: "y_rv",
		})
		appPushState(token, q, cd)
		return true
	case "y_rv":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := runVPN("revoke", focus)
		appMu.Lock()
		appFocus = ""
		appMu.Unlock()
		cd, _ := appMarshal(appState{
			S: "users", T: appTitle("Users", "Пользователи"),
			B: widgetTruncate(out, 300),
			H: appTitle("revoked — refresh list", "удалён — обновите список"),
		})
		// rebuild list
		if listCD, err := buildAppUsers(); err == nil {
			cd = listCD
		}
		appPushState(token, q, cd)
		return true
	case "up":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := enqueueNodeCmd(focus, "upgrade")
		cd, _ := appMarshal(appState{
			S: "ncard", T: focus, Lines: widgetTruncate(out, 400),
			Acts: "up,rb,jn,rs,nrn,mx", H: appTitle("upgrade queued/result", "upgrade"),
		})
		appPushState(token, q, cd)
		return true
	case "rb":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		disp := focus
		cd, err := buildAppRebootChallenge(focus, disp)
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
	case "y_rb":
		// legacy simple path unused; kept for safety
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := enqueueNodeCmd(focus, "reboot")
		cd, _ := appMarshal(appState{S: "ncard", T: focus, Lines: widgetTruncate(out, 400), Acts: "up,rb,jn"})
		appPushState(token, q, cd)
		return true
	case "jn":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		var out string
		role := nodeRole(focus)
		if role == "secondary" || strings.HasPrefix(focus, "secondary-") {
			out = runND("secondary", "cmd", focus, "metrics")
		} else {
			out = enqueueNodeCmd(focus, "metrics")
			if b, err := execJournalSingBox(); err == nil {
				out = b
			}
		}
		cd, _ := appMarshal(appState{
			S: "ncard", T: focus, Lines: widgetTruncate(out, 450),
			Acts: "up,rb,jn",
		})
		appPushState(token, q, cd)
		return true
	}


	// on-screen pad (widget profile has no text input)
	if strings.HasPrefix(act, "p:") {
		key := strings.TrimPrefix(act, "p:")
		appMu.Lock()
		mode := appComposeMode
		draft := appCompose
		focus := appFocus
		appMu.Unlock()
		if mode == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nopad"}`)
			return true
		}
		switch key {
		case "bs":
			if len(draft) > 0 {
				draft = draft[:len(draft)-1]
			}
		case "cl":
			draft = ""
		case "ok":
			appMu.Lock()
			appCompose = draft
			appComposeMode = ""
			appMu.Unlock()
			switch mode {
			case "quota":
				gb := 0.0
				fmt.Sscanf(draft, "%f", &gb)
				if focus != "" {
					_ = vpn.SetSoftLimitGB(focus, gb)
				}
				cd, _ := buildAppUserCard(focus)
				appPushState(token, q, cd)
				return true
			case "rename":
				if focus != "" && draft != "" {
					_ = runVPN("rename", focus, draft)
					focus = draft
				}
				cd, _ := buildAppUserCard(focus)
				appPushState(token, q, cd)
				return true
			case "add":
				if draft != "" {
					_ = runVPN("add", draft)
					cd, _ := buildAppUserCard(draft)
					appPushState(token, q, cd)
					return true
				}
				cd, _ := buildAppUsersHub()
				appPushState(token, q, cd)
				return true
			}
			cd, _ := buildAppScreen("home")
			appPushState(token, q, cd)
			return true
		default:
			if len(key) == 1 {
				if len(draft) < 24 {
					draft += key
				}
			}
		}
		appMu.Lock()
		appCompose = draft
		appMu.Unlock()
		show := draft
		if show == "" {
			show = "_"
		}
		title := focus
		if title == "" {
			title = appTitle("compose", "ввод")
		}
		screen := "ucard"
		if mode == "add" {
			screen = "users"
		}
		cd, _ := appMarshal(appState{
			S: screen, T: title, Pad: true, Draft: show,
			H: appTitle("pad · OK to apply", "панель · OK"),
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,rv",
		})
		appPushState(token, q, cd)
		return true
	}

	// reboot challenge code button
	if strings.HasPrefix(act, "rc:") {
		code := strings.TrimPrefix(act, "rc:")
		appMu.Lock()
		target := appRebootTarget
		if target == "" {
			target = appFocus
		}
		appMu.Unlock()
		if target == "" || code == "" {
			answerWidgetQuery(token, q.ID, `{"err":"reboot"}`)
			return true
		}
		if err := notify.ConsumeRebootChallenge(target, code); err != nil {
			cd, _ := appMarshal(appState{
				S: "reboot", T: "Reboot",
				B: err.Error(),
				H: appTitle("request reboot again", "запросите reboot снова"),
			})
			appPushState(token, q, cd)
			return true
		}
		var out string
		if target == "primary" {
			out = appTitle("reboot queued (+1 min)", "reboot поставлен (+1 мин)")
			go func() { _ = exec.Command("shutdown", "-r", "+1", "netductor TG reboot").Run() }()
		} else {
			out = enqueueNodeCmd(target, "reboot")
		}
		cd, _ := appMarshal(appState{
			S: "reboot", T: appTitle("Reboot", "Reboot"),
			Lines: widgetTruncate(out, 400),
			H:     target,
		})
		appPushState(token, q, cd)
		return true
	}

	if handleAppOpsAct(token, q, act, focus) {
		return true
	}
	if handleAppResidualAct(token, q, act, focus) {
		return true
	}
	return false
}

func execJournalSingBox() (string, error) {
	b, err := exec.Command("journalctl", "-u", "sing-box", "-n", "30", "--no-pager", "-o", "short-iso").CombinedOutput()
	return string(b), err
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
