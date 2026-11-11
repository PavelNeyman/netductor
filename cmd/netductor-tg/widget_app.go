package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/metrics"
	"github.com/PavelNeyman/netductor/internal/incident"
	gitstore "github.com/PavelNeyman/netductor/internal/git"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/sites"
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
updates:"Updates",primary:"Primary",secondary:"Sec",edge_a:"Edge",
channel:"Channel",versions:"Ver",nodes:"Nodes",devices:"Dev",sites:"Sites",
edge:"Edge",media:"Media",data:"Data",adv:"More",logs:"Logs",stack:"Stack",help:"Help",
git:"Git",backup:"Backup",alerts:"Alerts",incident:"Incident",reboot:"Reboot",
session:"Session",sessions:"Sessions",sshkeys:"SSH",presets:"Presets",mtls:"mTLS",
secops:"Sec ops",routers:"Routers",pending:"Pending",addons:"Addons",seclist:"Sec",
doctor:"Doctor",admin:"Admin",certs:"Certs",fw:"FW",dns:"DNS",nvr:"NVR",cleanup:"Clean",
ulist:"List",ucard:"User",ncard:"Node",menu:"Menu",back:"←",refresh:"↻",nodata:"—",empty:"∅",
en:"On",di:"Off",lk:"Link",sb:"Sub",rv:"Revoke",y_rv:"OK?",up:"Upgrade",rb:"Reboot",jn:"Log",
y_rb:"OK?",q50:"50G",q200:"200G",q0:"∞",rn:"Rename",add:"Add",qr:"QR",heal:"Heal",
al_test:"Test",al_clr:"Clear",gsync:"Sync",gship:"Ship",gjobs:"Jobs",spa_apply:"Apply",
api_tg:"API",usp_tg:"SP",vpnref:"Refresh",upol:"Policy",pr_new:"New",pr_draft:"Draft",
es_bak:"Backup",es_rst:"Restore",es_list:"USB",es_mute:"Mute",es_fmt:"Format",y_fmt:"OK?",
et_on:"Tpl on",et_off:"Tpl off",eg_code:"Grant",glog:"Log",gpause:"Pause",gresume:"Resume",grm:"Del",
edgestor:"Storage",edgetpl:"Tpl",edgeguest:"Guest",edgeluci:"LuCI",edgepol:"Policy",edgelife:"Life",
loc:"Loc",room_add:"+room",mt_list:"List",mt_rot:"Rotate",sec_exp:"Export",sec_enr:"Enroll",
er_reg:"Reg",er_bind:"Bind",er_apply:"Apply",cert_ren:"Renew",fw_apply:"FW",ag_scan:"Scan",
sess1:"1h",sess4:"4h",sess24:"24h",sess72:"72h",sessx:"Hours",srev:"Revoke all",cy:"Canary"
},
ru:{
home:"Меню",status:"Статус",users:"Юзеры",fleet:"Флот",tools:"Tools",operator:"Оператор",
updates:"Обновления",primary:"Primary",secondary:"Sec",edge_a:"Edge",
channel:"Канал",versions:"Версии",nodes:"Ноды",devices:"Устр.",sites:"Сайты",
edge:"Edge",media:"Медиа",data:"Данные",adv:"Ещё",logs:"Логи",stack:"Стек",help:"Справка",
git:"Git",backup:"Бэкап",alerts:"Алерты",incident:"Инцидент",reboot:"Ребут",
session:"Сессия",sessions:"Сессии",sshkeys:"SSH",presets:"Пресеты",mtls:"mTLS",
secops:"Sec ops",routers:"Роутеры",pending:"Pending",addons:"Аддоны",seclist:"Sec",
doctor:"Doctor",admin:"Админ",certs:"Серты",fw:"FW",dns:"DNS",nvr:"NVR",cleanup:"Очистка",
ulist:"Список",ucard:"User",ncard:"Node",menu:"Меню",back:"←",refresh:"↻",nodata:"—",empty:"∅",
en:"Вкл",di:"Выкл",lk:"Ссылка",sb:"Sub",rv:"Отозвать",y_rv:"OK?",up:"Upgrade",rb:"Ребут",jn:"Журнал",
y_rb:"OK?",q50:"50G",q200:"200G",q0:"∞",rn:"Имя",add:"Добавить",qr:"QR",heal:"Heal",
al_test:"Тест",al_clr:"Сброс",gsync:"Sync",gship:"Ship",gjobs:"Jobs",spa_apply:"Apply",
api_tg:"API",usp_tg:"SP",vpnref:"Обновить",upol:"Policy",pr_new:"Новый",pr_draft:"Черновик",
es_bak:"Backup",es_rst:"Restore",es_list:"USB",es_mute:"Mute",es_fmt:"Format",y_fmt:"OK?",
et_on:"Tpl on",et_off:"Tpl off",eg_code:"Grant",glog:"Log",gpause:"Pause",gresume:"Resume",grm:"Del",
edgestor:"Storage",edgetpl:"Tpl",edgeguest:"Guest",edgeluci:"LuCI",edgepol:"Policy",edgelife:"Life",
loc:"Loc",room_add:"+комн",mt_list:"List",mt_rot:"Rotate",sec_exp:"Export",sec_enr:"Enroll",
er_reg:"Reg",er_bind:"Bind",er_apply:"Apply",cert_ren:"Renew",fw_apply:"FW",ag_scan:"Scan",
sess1:"1ч",sess4:"4ч",sess24:"24ч",sess72:"72ч",sessx:"Часы",srev:"Сбросить",cy:"Canary"
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
  var tabs=document.getElementById("tabs");
  var seg=document.getElementById("seg");
  if(title)title.textContent=d.t||tr(d,d.s)||"Netductor";
  if(hint)hint.textContent=d.h||"";
  function mk(act,label,cls,parent){
    var btn=document.createElement("button");
    btn.textContent=label;
    if(cls)btn.className=cls;
    btn.addEventListener("click",function(){send(act);});
    parent.appendChild(btn);
  }
  if(tabs){
    tabs.innerHTML="";
    var tops=["status","users","fleet","tools","operator"];
    for(var ti=0;ti<tops.length;ti++){
      var id=tops[ti], cls=(d.s===id)?"primary":"";
      mk("g:"+id,tr(d,id),cls,tabs);
    }
  }
  if(seg){
    seg.innerHTML="";
    if(d.segs&&d.segs.length){
      for(var si=0;si<d.segs.length;si++){
        var sid=d.segs[si], sc=(d.seg===sid)?"primary":"";
        mk("sg:"+sid,tr(d,sid)||sid,sc,seg);
      }
    }
  }
  if(body){
    if(d.rows&&d.rows.length){
      var html="";
      for(var i=0;i<d.rows.length;i++){
        var r=d.rows[i];
        html+='<div class="row"><span class="k">'+(r[0]||"")+'</span><span class="v">'+(r[1]||"—")+'</span></div>';
        if(d.pct&&d.pct[i]!=null&&d.pct[i]>=0){
          var p=Math.max(0,Math.min(100,+d.pct[i]||0));
          html+='<div class="bar"><i style="width:'+p+'%"></i></div>';
        }
      }
      if(d.spk)html+='<canvas id="spk" width="300" height="36" style="width:100%;height:36px;margin:4px 0"></canvas>';
      if(d.lines){html+='<div class="pre"></div>';}
      body.innerHTML=html;
      if(d.lines){body.querySelector(".pre").textContent=d.lines;}
      if(d.spk)drawSpk(d.spk);
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
    function add(act,label,cls){mk(act,label,cls,nav);}
    add("r",tr(d,"refresh"),"primary");
    if(d.s&&d.s!=="home")add("b",tr(d,"back"));
    if(d.nav){
      for(var j=0;j<d.nav.length;j++)add("g:"+d.nav[j],tr(d,d.nav[j]));
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
        var cls="";
        if(/^(rv|y_rv|rb|y_rb|y_rbk|y_rst)$/.test(acts[a]))cls="danger";
        if(acts[a]==="en"||acts[a]==="et_on"||acts[a]==="et_soft_on")cls="on";
        add(acts[a],lab,cls);
      }
    }
    if(d.btns&&d.btns.length){
      for(var bi=0;bi<d.btns.length;bi++){
        var pair=d.btns[bi];
        if(!pair||!pair[0])continue;
        var bl=String(pair[1]||pair[0]); var bc="";
        if(/·\s*(on|OPEN|вкл)/i.test(bl))bc="on";
        add(String(pair[0]),bl,bc);
      }
    }
    if(d.s!=="home")add("g:home",tr(d,"menu"));
    if(d.s==="home"){add("L:ru","RU");add("L:en","EN");}
  }
}
function drawSpk(s){var c=document.getElementById("spk");if(!c||!c.getContext)return;var a=String(s).split(",");var ctx=c.getContext("2d"),w=c.width,h=c.height,n=a.length;if(n<2)return;ctx.clearRect(0,0,w,h);ctx.strokeStyle="#2a6df4";ctx.lineWidth=2;ctx.beginPath();for(var i=0;i<n;i++){var v=Math.max(0,Math.min(100,+a[i]||0));var x=i/(n-1)*(w-2)+1,y=h-1-(v/100)*(h-4);if(i===0)ctx.moveTo(x,y);else ctx.lineTo(x,y);}ctx.stroke();}
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
	// Single SPA HTML (canvas-capable). Multi-shell split deferred — see UI v2 plan.
	css := `body{margin:0;padding:8px;font:13px/1.3 system-ui,sans-serif;background:var(--tg-theme-bg-color,#0f1419);color:var(--tg-theme-text-color,#e7ecf3)}
#tabs,#seg,#n{display:flex;flex-wrap:wrap;gap:5px}
#tabs{margin:0 0 6px}#tabs button{flex:1;min-width:52px;padding:9px 4px;font:600 11px system-ui;border-radius:9px}
h1{margin:0 0 4px;font-size:15px}
.row{display:flex;justify-content:space-between;gap:6px;margin:3px 0}
.k{color:var(--tg-theme-hint-color,#8b98a5)}.v{font-weight:600;text-align:right;word-break:break-word}
.bar{height:5px;border-radius:3px;background:var(--tg-theme-secondary-bg-color,#1c2733);margin:1px 0 5px;overflow:hidden}
.bar>i{display:block;height:100%;background:var(--tg-theme-button-color,#2a6df4)}
.pre{margin:5px 0;padding:8px;background:var(--tg-theme-secondary-bg-color,#1c2733);border-radius:9px;font:12px/1.35 system-ui;white-space:pre-wrap;max-height:200px;overflow:hidden}
#seg{margin:0 0 6px}#seg button{flex:1;padding:7px;border-radius:8px;font-weight:600}
#n{margin-top:6px}#n button{padding:10px 10px;font:600 12px system-ui;border-radius:9px;min-width:calc(50% - 5px)}
button{border:0;background:var(--tg-theme-secondary-bg-color,#1c2733);color:inherit}
button.primary{background:var(--tg-theme-button-color,#2a6df4);color:var(--tg-theme-button-text-color,#fff)}
button.danger{background:var(--tg-theme-destructive-text-color,#e53935);color:#fff}
button.on{outline:2px solid var(--tg-theme-button-color,#2a6df4)}
#h{margin-top:6px;color:var(--tg-theme-hint-color,#8b98a5);font-size:11px}`
	return `<!DOCTYPE html><html><head><meta charset="utf-8"/><style>` + css + `</style></head><body>
<div id="tabs"></div>
<h1 id="t">Netductor</h1>
<div id="seg"></div>
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
	Acts  string      `json:"acts,omitempty"`
	Btns  [][2]string `json:"btns,omitempty"`
	Draft string      `json:"draft,omitempty"`
	Pad   bool        `json:"pad,omitempty"`
	Segs  []string    `json:"segs,omitempty"`
	Seg   string      `json:"seg,omitempty"`
	Pct   []int       `json:"pct,omitempty"`
	Spk   string      `json:"spk,omitempty"`
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


func parseCPUMemPct(res string) (cpu, mem int) {
	cpu, mem = -1, -1
	res = strings.ReplaceAll(res, "%", "")
	parts := strings.Split(res, "/")
	if len(parts) >= 1 {
		var c float64
		if _, err := fmt.Sscanf(strings.TrimSpace(parts[0]), "%f", &c); err == nil {
			cpu = int(c)
		}
	}
	if len(parts) >= 2 {
		var m float64
		if _, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &m); err == nil {
			mem = int(m)
		}
	}
	return cpu, mem
}

func sparkCPU(n int) string {
	if n <= 0 {
		n = 24
	}
	hist := metrics.History(n)
	vals := make([]string, 0, len(hist))
	for _, row := range hist {
		v, ok := row["cpu_pct"].(float64)
		if !ok {
			continue
		}
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		vals = append(vals, fmt.Sprintf("%.0f", v))
	}
	if len(vals) < 2 {
		m := metrics.Collect()
		if c, ok := m["cpu_pct"].(float64); ok {
			return fmt.Sprintf("%.0f", c)
		}
		return ""
	}
	return strings.Join(vals, ",")
}

func parseResPct(res string) []int {
	// res like "cpu 12% mem 40%" — optional bars aligned to version/host/nodes/svc/cpu row index 4
	out := []int{-1, -1, -1, -1, -1, -1, -1}
	res = strings.ToLower(res)
	var cpu, mem int
	fmt.Sscanf(res, "cpu %d", &cpu)
	if i := strings.Index(res, "mem"); i >= 0 {
		fmt.Sscanf(res[i:], "mem %d", &mem)
	}
	if cpu > 0 {
		out[4] = cpu
	}
	if mem > 0 && cpu == 0 {
		out[4] = mem
	}
	// keep only if any set
	for _, v := range out {
		if v >= 0 {
			return out
		}
	}
	return nil
}

func formatCmdOut(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if strings.HasPrefix(s, "{") {
		var m map[string]any
		if json.Unmarshal([]byte(s), &m) == nil {
			return formatMapLines(m, 480)
		}
	}
	if strings.HasPrefix(s, "[") {
		var arr []any
		if json.Unmarshal([]byte(s), &arr) == nil {
			var b strings.Builder
			for i, it := range arr {
				if i >= 12 || b.Len() > 450 {
					break
				}
				switch v := it.(type) {
				case map[string]any:
					id, _ := v["id"].(string)
					if id == "" {
						id, _ = v["name"].(string)
					}
					ver, _ := v["version"].(string)
					st, _ := v["status"].(string)
					line := strings.TrimSpace(fmt.Sprintf("%d %s %s %s", i+1, id, ver, st))
					b.WriteString(line)
					b.WriteByte('\n')
				default:
					fmt.Fprintf(&b, "%d %v\n", i+1, v)
				}
			}
			if b.Len() > 0 {
				return strings.TrimSpace(b.String())
			}
		}
	}
	// key: value or key=value lines already readable
	return s
}

func formatMapLines(m map[string]any, max int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		if b.Len() > max {
			break
		}
		v := m[k]
		switch x := v.(type) {
		case map[string]any, []any:
			continue
		case bool:
			fmt.Fprintf(&b, "%s: %v\n", k, x)
		case float64:
			if x == float64(int64(x)) {
				fmt.Fprintf(&b, "%s: %d\n", k, int64(x))
			} else {
				fmt.Fprintf(&b, "%s: %.2f\n", k, x)
			}
		default:
			sv := fmt.Sprint(x)
			if len(sv) > 80 {
				sv = sv[:80] + "…"
			}
			fmt.Fprintf(&b, "%s: %s\n", k, sv)
		}
	}
	return strings.TrimSpace(b.String())
}

func appLines(s string) string {
	return widgetTruncate(formatCmdOut(s), 480)
}


func buildAppUpdates(seg string) (string, error) {
	if seg == "" {
		seg = "primary"
	}
	d := fleet.Build()
	st := appState{
		S: "updates", T: appTitle("Updates", "Обновления"),
		Segs: []string{"primary", "secondary", "edge_a"},
		Seg:  seg,
		H:    appTitle("Primary · Secondary · Edge agents", "Primary · Secondary · Edge"),
	}
	switch seg {
	case "secondary":
		var lines []string
		var ids []string
		for i, s := range d.Secondaries {
			if i >= 10 {
				break
			}
			on := "off"
			if s.Online {
				on = "on"
			}
			name := s.ID
			if name == "" {
				name = s.Name
			}
			ver := s.Version
			lines = append(lines, fmt.Sprintf("%d %s %s %s", len(ids)+1, name, ver, on))
			ids = append(ids, name)
		}
		appMu.Lock()
		appListIDs = ids
		appMu.Unlock()
		st.Lines = strings.Join(lines, "\n")
		st.IDs = strings.Join(ids, ",")
		if st.Lines == "" {
			st.B = appTitle("no secondaries", "нет secondary")
		}
		st.Acts = "vpnref"
		st.H = appTitle("# node · upgrade via secondary cmd", "# нода · upgrade через secondary cmd")
	case "edge_a":
		out := formatCmdOut(runND("edge", "list"))
		st.Lines = widgetTruncate(out, 450)
		st.Acts = ""
		st.H = appTitle("edge agents / devices", "edge агенты / устройства")
	default:
		upd := appTitle("up to date", "актуально")
		if d.UpdateAvailable {
			upd = d.LatestRelease
		}
		st.Rows = [][2]string{
			{appTitle("primary", "primary"), d.PrimaryVersion},
			{appTitle("latest", "latest"), upd},
			{appTitle("secondaries", "secondary"), fmt.Sprintf("%d", len(d.Secondaries))},
		}
		st.Nav = []string{"stack", "versions"}
		st.H = appTitle("stack apply from Operator · versions list", "stack apply в Operator · список versions")
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
		Nav: []string{"status", "users", "fleet", "updates", "tools", "operator", "help"},
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
	cpuN, _ := parseCPUMemPct(d.Res)
	pct := []int{-1, -1, -1, -1, -1, -1, -1}
	if cpuN >= 0 {
		pct[4] = cpuN
	}
	spk := sparkCPU(24)
	st := appState{
		S: "status", T: d.Title, Rows: rows, H: d.Hint,
		Nav: []string{"channel", "netmap", "actions", "sessions", "digest", "incident", "reboot", "updates"},
	}
	if cpuN >= 0 {
		st.Pct = pct
	}
	if spk != "" {
		st.Spk = spk
	}
	return appMarshal(st)
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
		Nav: []string{"nodes", "updates", "routers", "sites", "secondary", "addons", "secops"},
	})
}

func buildAppToolsHub() (string, error) {
	// Classic Tools = Day2Groups (edge/media/data/adv) + Git/Updates/Logs/SSH keys rows.
	return appMarshal(appState{
		S: "tools", T: appTitle("Tools", "Инструменты"),
		B: appTitle("Edge · Media · Data · Advanced", "Edge · Медиа · Данные · Ещё"),
		H: appTitle("no Status/Users/Fleet duplicates", "без дублей главных хабов"),
		Nav: []string{"edge", "media", "data", "adv", "updates", "versions", "logs", "ssh", "sshkeys", "git", "presets", "mtls"},
	})
}

func buildAppEdgeHub() (string, error) {
	return appMarshal(appState{
		S: "edge", T: appTitle("Edge", "Edge"),
		B: appTitle("routers · sites · pending", "роутеры · сайты · pending"),
		Nav: []string{"routers", "sites", "pending", "edgeguest", "edgeluci", "edgestor", "edgepol", "edgetpl", "edgelife", "loc"},
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
	out := appLines(runND("logs", "tail"))
	if out == "" {
		out = appLines(runND("status"))
	}
	return appMarshal(appState{
		S: "logs", T: appTitle("Logs", "Логи"),
		Lines: widgetTruncate(strings.TrimSpace(out), 500),
		H: appTitle("refresh for latest", "обновите для свежих"),
	})
}

func buildAppStack() (string, error) {
	out := appLines(runND("stack", "status"))
	return appMarshal(appState{
		S: "stack", T: appTitle("Stack", "Стек"),
		Lines: out,
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
		Lines: appLines(out),
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
	apiLab := "API WAN · " + apiCell
	uspLab := "Users→SP · " + uspCell
	st := appState{
		S: "operator", T: appTitle("Operator", "Оператор"),
		Rows: [][2]string{
			{"API WAN", apiCell},
			{"Users→SP", uspCell},
		},
		Nav: []string{"session", "alerts", "sessions", "audit", "svcpaths", "stack", "admin", "updates"},
		Btns: [][2]string{
			{"api_tg", apiLab},
			{"usp_tg", uspLab},
			{"vpnref", appTitle("Refresh links", "Обновить ссылки")},
		},
		H: appTitle("tap toggle to flip", "нажмите чтобы переключить"),
	}
	return appMarshal(st)
}

func buildAppND(screen, en, ru string, args ...string) (string, error) {
	out := appLines(runND(args...))
	if out == "" {
		out = appTitle("(empty)", "(пусто)")
	}
	return appMarshal(appState{
		S: screen, T: appTitle(en, ru),
		Lines: out,
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
	stLab := en
	if stLab == "" || stLab == "?" {
		stLab = "—"
	}
	rows := [][2]string{
		{appTitle("status", "статус"), stLab},
		{appTitle("quota", "квота"), fmt.Sprintf("%.0f GiB", lim)},
		{appTitle("profile", "профиль"), prof},
		{appTitle("kind", "тип"), kind},
	}
	on := strings.Contains(strings.ToLower(en), "en") || strings.EqualFold(en, "on") || strings.EqualFold(en, "true") || en == "1"
	// heuristic: common list columns show enabled/disabled
	if strings.Contains(strings.ToLower(en), "dis") {
		on = false
	}
	vpnLab := appTitle("VPN · off (tap on)", "VPN · выкл (включить)")
	if on {
		vpnLab = appTitle("VPN · on (tap off)", "VPN · вкл (выключить)")
	}
	vpnAct := "en"
	if on {
		vpnAct = "di"
	}
	return appMarshal(appState{
		S: "ucard", T: name, Rows: rows,
		H: appTitle("toggle + actions", "переключатель и действия"),
		Btns: [][2]string{
			{vpnAct, vpnLab},
			{"lk", appTitle("Link", "Ссылка")},
			{"sb", "Sub"},
			{"qr", "QR"},
			{"upol", appTitle("Policy", "Политика")},
		},
		Acts: "q50,q200,q0,qn,rn,kd,ku,sp,cy,rv",
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
		{appTitle("host", "хост"), host},
		{appTitle("role", "роль"), displayRole(role)},
		{appTitle("status", "статус"), st},
		{appTitle("version", "версия"), ver},
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
		rows, lines := formatGitHubLines()
		st := appState{S: "git", T: "Git", Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig",
			H: appTitle("sync · ship · jobs", "sync · ship · jobs")}
		if len(rows) > 0 {
			st.Rows = rows
		} else {
			st.Lines = lines
		}
		return appMarshal(st)
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
		return buildAppEdgeLife() // pick device then policy via o:
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
	case "updates":
		return buildAppUpdates("")
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
		return buildAppSvcPaths()
	case "prdraft":
		return buildAppPresetDraft()
	case "upol":
		appMu.Lock()
		n := appFocus
		appMu.Unlock()
		if n == "" {
			return appMarshal(appState{S: "upol", T: "policy", B: "no user"})
		}
		return buildAppUserPolicy(n)
	case "lampac":
		return appFromLegacyWidget("lampac", "Lampac", "Lampac", lampacWidgetChatDataJSON)
	case "presets":
		return buildAppPresets()
	case "mtls":
		return buildAppMTLS()
	case "secops":
		return buildAppSecondaryOps()
	case "edgelife":
		return buildAppEdgeLife()
	case "loc":
		return buildAppLoc()
	case "aptguard":
		return buildAppAptguard()
	case "canary":
		return buildAppCanary()
	case "sessions":
		out := strings.TrimSpace(runND("session", "list"))
		if out == "" {
			out = runVPN("session", "list")
		}
		return appMarshal(appState{S: "sessions", T: appTitle("Sessions", "Сессии"), Lines: appLines(out), Acts: "srev"})
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
	screen := parseScreenFromChatData(cd)
	if dualEnabled() && q != nil && q.Chat != nil {
		infoShell := infoShellForScreen(screen)
		infoCD, err := buildInfoData(screen)
		if err != nil {
			infoCD, _ = appMarshal(appState{S: "info", T: "Status", B: err.Error()})
		}
		if c, e := clipWidgetChatData(infoCD); e == nil {
			infoCD = c
		}
		if c, e := clipWidgetChatData(cd); e == nil {
			cd = c
		}
		pushDualPair(token, q, infoCD, cd, infoShell, shellHub)
		return
	}
	appPushStateSingle(token, q, cd)
}

func handleAppAct(token string, q *widgetQuery, admin int64, act string) bool {
	act = strings.TrimSpace(act)
	if act == "" {
		act = "r"
	}

	// Dual: refresh from info slot — update info only (nav untouched).
	if dualEnabled() && q != nil && q.Chat != nil && (act == "r" || act == "refresh") {
		if p, ok := getDual(q.Chat.ID); ok && p.InfoID > 0 && q.MessageID == p.InfoID {
			appMu.Lock()
			s := appScreen
			appMu.Unlock()
			infoCD, err := buildInfoData(s)
			if err != nil {
				answerWidgetQuery(token, q.ID, `{"err":"build"}`)
				return true
			}
			if c, e := clipWidgetChatData(infoCD); e == nil {
				infoCD = c
			}
			answerWidgetQuery(token, q.ID, `{"ok":true}`)
			infoShell := infoShellForScreen(s)
			editDualSlot(token, q.Chat.ID, "info", appInfoHTML(), infoCD, infoShell)
			return true
		}
	}

	// segment switch (updates hub etc.)
	if strings.HasPrefix(act, "sg:") {
		seg := strings.TrimPrefix(act, "sg:")
		appMu.Lock()
		s := appScreen
		appMu.Unlock()
		var cd string
		var err error
		if s == "updates" || seg == "primary" || seg == "secondary" || seg == "edge_a" {
			cd, err = buildAppUpdates(seg)
		} else {
			cd, err = buildAppScreen(s)
		}
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		appPushState(token, q, cd)
		return true
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
		case "nodes", "secondary", "addons", "seclist", "secops":
			parent = "fleet"
		case "routers", "sites", "pending", "edgeguest", "edgeluci", "edgestor", "edgepol", "edgetpl", "edgelife", "loc":
			parent = "edge"
		case "lampac", "canary", "nvr", "aptguard":
			parent = "media"
		case "backup", "logs", "dns":
			parent = "data"
		case "probes", "stack", "doctor", "certs", "fw", "cleanup":
			parent = "adv"
		case "edge", "media", "data", "adv", "ssh", "sshkeys", "git", "presets", "mtls":
			parent = "tools"
		case "svcpaths", "session", "alerts", "admin", "audit", "sessions":
			parent = "operator"
		case "users", "status", "fleet", "tools", "operator", "help", "updates":
			parent = "home"
		case "versions":
			parent = "updates"
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
				Acts: "al_test,al_clr",
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
				Acts: "es_bak,es_rst,es_list,es_mute,es_fmt",
				H: id,
			})
		case "edgetpl":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			cd, err = buildAppTemplateCard(id)
		case "edgepol", "edgelife":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			cd, err = buildAppEdgePolicyCard(id)
		case "loc":
			appMu.Lock()
			appFocus = id
			appMu.Unlock()
			rooms, _ := sites.ListRooms(id)
			var lines []string
			for i, r := range rooms {
				if i >= 10 {
					break
				}
				lines = append(lines, fmt.Sprintf("%d %s %s", i+1, r.ID, r.Name))
			}
			cd, err = appMarshal(appState{
				S: "loc", T: id,
				Lines: strings.Join(lines, "\n"),
				Acts: "room_add",
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
			Lines: appLines(out),
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
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,upol,rv",
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
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,upol,rv",
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
		cd, _ := appMarshal(appState{S: "git", T: focus, Lines: appLines(msg), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig"})
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
		cd, _ := appMarshal(appState{S: "git", T: focus, Lines: appLines(msg), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig"})
		appPushState(token, q, cd)
		return true
	case "eg_gr":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := runND("edge", "guest", "grant", focus)
		cd, _ := appMarshal(appState{S: "edgeguest", T: focus, Lines: appLines(out), Acts: "eg_gr,eg_rv"})
		appPushState(token, q, cd)
		return true
	case "eg_rv":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := runND("edge", "guest", "revoke", focus)
		cd, _ := appMarshal(appState{S: "edgeguest", T: focus, Lines: appLines(out), Acts: "eg_gr,eg_rv"})
		appPushState(token, q, cd)
		return true
	case "el_on":
		out := runND("edge", "luci", "on", "4h")
		cd, _ := appMarshal(appState{S: "edgeluci", T: "Edge LuCI", Lines: appLines(out), Acts: "el_on,el_off"})
		appPushState(token, q, cd)
		return true
	case "el_off":
		out := runND("edge", "luci", "off")
		cd, _ := appMarshal(appState{S: "edgeluci", T: "Edge LuCI", Lines: appLines(out), Acts: "el_on,el_off"})
		appPushState(token, q, cd)
		return true

	case "heal":
		out := runND("stack", "heal")
		cd, _ := appMarshal(appState{S: "stack", T: "Stack", Lines: appLines(out), Acts: "heal,rbk,wd"})
		appPushState(token, q, cd)
		return true
	case "wd":
		out := runND("stack", "watchdog")
		cd, _ := appMarshal(appState{S: "stack", T: "Stack", Lines: appLines(out), Acts: "heal,rbk,wd"})
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
		cd, _ := appMarshal(appState{S: "stack", T: "Stack", Lines: appLines(out), Acts: "heal,rbk,wd"})
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
			Lines: appLines(out),
		})
		appPushState(token, q, cd)
		return true
	case "al_test":
		out := runND("alerts", "test")
		if strings.TrimSpace(out) == "" {
			out = runND("notify", "test")
		}
		cd, _ := appMarshal(appState{S: "alerts", T: appTitle("Alerts", "Алерты"), Lines: appLines(out), Acts: "al_test,al_clr"})
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
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,upol,rv",
		})
		appPushState(token, q, cd)
		return true
	case "rs":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := restartSingBox(focus)
		cd, _ := appMarshal(appState{S: "ncard", T: focus, Lines: appLines(out), Acts: "up,rb,jn,rs,nrn,mx"})
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
		cd, _ := appMarshal(appState{S: "git", T: "Git", Lines: appLines(b.String()), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig"})
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
			cd, _ := appMarshal(appState{S: "ucard", T: focus, B: appTitle("empty link", "пустая ссылка"), Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,upol,rv"})
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
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,upol,rv",
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
			S: "ncard", T: focus, Lines: appLines(out),
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
		cd, _ := appMarshal(appState{S: "ncard", T: focus, Lines: appLines(out), Acts: "up,rb,jn"})
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
			S: "ncard", T: focus, Lines: appLines(out),
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
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,upol,rv",
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
			Lines: appLines(out),
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
	if handleAppGapsAct(token, q, act, focus) {
		return true
	}
	if handleAppFinalAct(token, q, act, focus) {
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
