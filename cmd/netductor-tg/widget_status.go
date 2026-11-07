package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/metrics"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

// CF-18.4 HTMLBubbles status (Bot API 10.4). UX only — no CF-15 floor (AGENT_HANDOFF).

const statusWidgetHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"/>
<style>
body{margin:0;padding:10px;font:13px/1.35 system-ui,sans-serif;background:#0f1419;color:#e7ecf3}
h1{margin:0 0 8px;font-size:15px}.row{display:flex;justify-content:space-between;margin:3px 0}
.k{color:#8b98a5}.v{font-weight:600}.ok{color:#3dd68c}.bad{color:#ff6b6b}
.nav{display:flex;flex-wrap:wrap;gap:6px;margin-top:10px}
button{border:0;border-radius:8px;padding:7px 10px;font:12px system-ui;background:#1c2733;color:#e7ecf3}
button.primary{background:#2a6df4}.hint{margin-top:8px;color:#8b98a5;font-size:11px}
</style></head><body>
<h1 id="title">Netductor</h1>
<div class="row"><span class="k">version</span><span class="v" id="ver">—</span></div>
<div class="row"><span class="k">host</span><span class="v" id="host">—</span></div>
<div class="row"><span class="k">nodes</span><span class="v" id="nodes">—</span></div>
<div class="row"><span class="k">services</span><span class="v" id="svc">—</span></div>
<div class="row"><span class="k">cpu / mem</span><span class="v" id="res">—</span></div>
<div class="row"><span class="k">secondary</span><span class="v" id="sec">—</span></div>
<div class="row"><span class="k">migrate</span><span class="v" id="mig">—</span></div>
<div class="nav">
<button class="primary" data-act="refresh">↻ Refresh</button>
<button data-act="versions">Versions</button>
<button data-act="channel">Channel</button>
<button data-act="menu">Menu</button>
</div>
<div class="hint" id="hint">Refresh · Versions shows migrate tools when needed</div>
<script>
(function(){
function set(id,t,cls){var el=document.getElementById(id);if(!el)return;el.textContent=t||'—';el.className='v'+(cls?' '+cls:'');}
function paint(d){if(!d)return;set('title',d.title||'Netductor');set('ver',d.ver);set('host',d.host);set('nodes',d.nodes);
set('svc',d.svc,d.svc_ok===false?'bad':(d.svc_ok?'ok':''));set('res',d.res);
set('sec',d.sec,d.sec_ok===false?'bad':(d.sec_ok?'ok':''));set('mig',d.mig,d.mig_ok===false?'bad':(d.mig_ok?'ok':''));
if(d.hint){var h=document.getElementById('hint');if(h)h.textContent=d.hint;}}
function data(){try{if(window.TelegramWidget&&TelegramWidget.chatData)return TelegramWidget.chatData;}catch(e){}return null;}
function onData(){paint(data());}
if(window.TelegramWidget){if(TelegramWidget.onChatDataChanged)TelegramWidget.onChatDataChanged(onData);if(TelegramWidget.ready)try{TelegramWidget.ready();}catch(e){}}
onData();
document.querySelectorAll('button[data-act]').forEach(function(btn){btn.addEventListener('click',function(){
var act=btn.getAttribute('data-act')||'';if(!window.TelegramWidget||!TelegramWidget.request)return;
TelegramWidget.request(act).then(function(r){try{if(r&&r.result){var j=JSON.parse(r.result);if(j&&j.d)paint(j.d);}}catch(e){}}).catch(function(){});
});});
})();
</script>
</body></html>`

type statusWidgetData struct {
	Title string `json:"title"`
	Ver   string `json:"ver"`
	Host  string `json:"host"`
	Nodes string `json:"nodes"`
	Svc   string `json:"svc"`
	SvcOK *bool  `json:"svc_ok,omitempty"`
	Res   string `json:"res"`
	Sec   string `json:"sec"`
	SecOK *bool  `json:"sec_ok,omitempty"`
	Mig   string `json:"mig"`
	MigOK *bool  `json:"mig_ok,omitempty"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

func widgetsEnabled() bool { return os.Getenv("NETDUCTOR_TG_WIDGETS") == "1" }

func widgetAPIBase(token string) string {
	if os.Getenv("NETDUCTOR_TG_WIDGET_BETA") == "1" {
		return "https://api.telegram.org/beta/bot" + token + "/"
	}
	return "https://api.telegram.org/bot" + token + "/"
}

func buildStatusWidgetData() statusWidgetData {
	ru := getLang() != "en"
	host, _ := os.Hostname()
	d := fleet.Build()
	ver := d.PrimaryVersion
	if ver == "" {
		ver = ndver.Running()
	}
	m := metrics.Collect()
	cpu, mem := "—", "—"
	if c, ok := m["cpu_pct"].(float64); ok {
		cpu = fmt.Sprintf("%.0f%%", c)
	}
	if mm, ok := m["mem"].(map[string]any); ok {
		used, total := asFloat(mm["used"]), asFloat(mm["total"])
		if total > 0 {
			mem = fmt.Sprintf("%.0f%%", used/total*100)
		}
	}
	online, totalN := 0, 0
	for _, r := range parseNodesList() {
		totalN++
		if strings.EqualFold(r.Status, "online") {
			online++
		}
	}
	svcOK := true
	parts := []string{}
	for _, unit := range []string{"sing-box", "netductor-api", "netductor-telegram-bot"} {
		out, _ := exec.Command("systemctl", "is-active", unit).CombinedOutput()
		st := strings.TrimSpace(string(out))
		short := map[string]string{"netductor-telegram-bot": "tg", "netductor-api": "api"}[unit]
		if short == "" {
			short = unit
		}
		if st == "active" {
			parts = append(parts, short+"✓")
		} else {
			parts = append(parts, short+"✗")
			svcOK = false
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
			secLine = name + " online"
		} else {
			secLine = name + " offline"
		}
	}
	migOK := true
	migLine := "ok"
	if migrateNeedsAttention() {
		migOK = false
		migLine = "needs attention"
		if ru {
			migLine = "нужно внимание"
		}
	}
	title, hint := "Status", "Refresh updates · Versions opens migrate tools when needed"
	if ru {
		title, hint = "Статус", "Refresh · Versions — миграции при необходимости"
	}
	return statusWidgetData{
		Title: title, Ver: ver, Host: host, Nodes: fmt.Sprintf("%d/%d", online, totalN),
		Svc: strings.Join(parts, " · "), SvcOK: &svcOK, Res: cpu + " / " + mem,
		Sec: secLine, SecOK: secOK, Mig: migLine, MigOK: &migOK, Hint: hint, TS: time.Now().Unix(),
	}
}

func statusWidgetChatDataJSON() (string, error) {
	b, err := json.Marshal(buildStatusWidgetData())
	if err != nil {
		return "", err
	}
	if len(b) > 1024 {
		return "", fmt.Errorf("widget chat_data %d > 1024", len(b))
	}
	return string(b), nil
}

func apiPostWidget(token, method string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(widgetAPIBase(token)+method, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if json.Unmarshal(data, &wr) == nil && !wr.OK {
		desc := wr.Description
		if desc == "" {
			desc = string(data)
		}
		return data, fmt.Errorf("telegram %s: %s", method, desc)
	}
	if resp.StatusCode >= 300 {
		return data, fmt.Errorf("telegram %s HTTP %d", method, resp.StatusCode)
	}
	return data, nil
}

func sendStatusWidget(token string, chat int64) bool {
	if !widgetsEnabled() {
		return false
	}
	cd, err := statusWidgetChatDataJSON()
	if err != nil {
		fmt.Fprintln(os.Stderr, "widget chat_data:", err)
		return false
	}
	_, err = apiPostWidget(token, "sendWidget", map[string]any{
		"chat_id": chat, "html": statusWidgetHTML, "chat_data": cd, "allow_queries": true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sendWidget:", err)
		return false
	}
	return true
}

func editStatusWidgetData(token string, chat int64, msgID int) bool {
	if !widgetsEnabled() {
		return false
	}
	cd, err := statusWidgetChatDataJSON()
	if err != nil {
		return false
	}
	_, err = apiPostWidget(token, "editMessageWidgetData", map[string]any{
		"chat_id": chat, "message_id": msgID, "chat_data": cd,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "editMessageWidgetData:", err)
		return false
	}
	return true
}

func answerWidgetQuery(token, queryID, result string) {
	payload := map[string]any{"widget_query_id": queryID}
	if result != "" {
		payload["data"] = result
	}
	if _, err := apiPostWidget(token, "answerWidgetQuery", payload); err != nil {
		fmt.Fprintln(os.Stderr, "answerWidgetQuery:", err)
	}
}

func handleWidgetQuery(token string, q *widgetQuery, admin int64) bool {
	if q == nil {
		return false
	}
	if admin != 0 && q.From.ID != 0 && q.From.ID != admin {
		if q.Chat == nil || q.Chat.ID != admin {
			answerWidgetQuery(token, q.ID, `{"err":"forbidden"}`)
			return true
		}
	}
	act := strings.TrimSpace(q.Data)
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		cd, err := statusWidgetChatDataJSON()
		if err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		if q.MessageID != 0 && q.Chat != nil {
			_ = editStatusWidgetData(token, q.Chat.ID, q.MessageID)
		}
		answerWidgetQuery(token, q.ID, `{"ok":true,"d":`+cd+`}`)
		return true
	case "versions":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"versions"}`)
		reply(token, chat, q.MessageID, formatVersionsHTML(false), versionsKeyboard())
		return true
	case "channel":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"channel"}`)
		reply(token, chat, q.MessageID, formatChannelDetailHTML(), statusKeyboard())
		return true
	case "menu":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"menu"}`)
		reply(token, chat, q.MessageID, T("main_menu"), mainKeyboard())
		return true
	default:
		answerWidgetQuery(token, q.ID, `{"err":"unknown"}`)
		return true
	}
}

func tryReplyStatusWidget(token string, chat int64, msgID int) bool {
	if !widgetsEnabled() {
		return false
	}
	if msgID > 0 && editStatusWidgetData(token, chat, msgID) {
		return true
	}
	return sendStatusWidget(token, chat)
}
