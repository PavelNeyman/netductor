package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
)

// Channel HTML widget — Phase 1 read-only family.

func channelWidgetHTML() string {
	return `<!DOCTYPE html>
<html><head><meta charset="utf-8"/>
<style>
body{margin:0;padding:10px;font:13px/1.35 system-ui,sans-serif;background:#0f1419;color:#e7ecf3}
h1{margin:0 0 8px;font-size:15px}.row{display:flex;justify-content:space-between;margin:3px 0}
.k{color:#8b98a5}.v{font-weight:600}.ok{color:#3dd68c}.bad{color:#ff6b6b}
.nav{display:flex;flex-wrap:wrap;gap:6px;margin-top:10px}
button{border:0;border-radius:8px;padding:7px 10px;font:12px system-ui;background:#1c2733;color:#e7ecf3}
button.primary{background:#2a6df4}.hint{margin-top:8px;color:#8b98a5;font-size:11px}
</style></head><body>
<h1 id="title">Channel</h1>
<div class="row"><span class="k">primary</span><span class="v" id="pri">—</span></div>
<div class="row"><span class="k">secondary</span><span class="v" id="sec">—</span></div>
<div class="row"><span class="k">uplink</span><span class="v" id="up">—</span></div>
<div class="row"><span class="k">face</span><span class="v" id="face">—</span></div>
<div class="row"><span class="k">path</span><span class="v" id="path">—</span></div>
<div class="nav">
<button class="primary" data-act="channel:refresh">↻ Refresh</button>
<button data-act="channel:status">Status</button>
<button data-act="channel:menu">Menu</button>
</div>
<div class="hint" id="hint">Channel path summary</div>
<script>` + widgetBootJS + `
(function(){
function set(id,t,cls){var el=document.getElementById(id);if(!el)return;el.textContent=t||'—';el.className='v'+(cls?' '+cls:'');}
function paint(d){if(!d)return;set('title',d.title||'Channel');set('pri',d.pri);set('sec',d.sec,d.sec_ok===false?'bad':(d.sec_ok?'ok':''));
set('up',d.up,d.up_ok===false?'bad':(d.up_ok?'ok':''));set('face',d.face);set('path',d.path,d.path_ok===false?'bad':(d.path_ok?'ok':''));
if(d.hint){var h=document.getElementById('hint');if(h)h.textContent=d.hint;}}
__ndBind(paint);
document.querySelectorAll('[data-act]').forEach(function(btn){
btn.addEventListener('click',function(){__ndSend(btn.getAttribute('data-act')||'channel:refresh');});
});
})();
</script></body></html>`
}

type channelWidgetData struct {
	Title  string `json:"title"`
	Pri    string `json:"pri"`
	Sec    string `json:"sec"`
	SecOK  *bool  `json:"sec_ok,omitempty"`
	Up     string `json:"up"`
	UpOK   *bool  `json:"up_ok,omitempty"`
	Face   string `json:"face"`
	Path   string `json:"path"`
	PathOK *bool  `json:"path_ok,omitempty"`
	Hint   string `json:"hint,omitempty"`
	TS     int64  `json:"ts"`
}

func buildChannelWidgetData() channelWidgetData {
	ru := getLang() != "en"
	d := fleet.Build()
	pri := d.PrimaryVersion
	if pri == "" {
		pri = "primary"
	}
	secLine := "—"
	var secOK *bool
	upLine := "—"
	var upOK *bool
	face := "—"
	pathLine := "—"
	var pathOK *bool
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
		// Best-effort fields if present on fleet secondary struct via fmt
		upLine = "see status"
		face = "—"
		pathLine = "e2e n/a"
	}
	_ = strings.Builder{}
	title, hint := "Channel", "Refresh · path detail also in rich Status"
	if ru {
		title, hint = "Канал", "Refresh · детали также в Status"
	}
	return channelWidgetData{
		Title: title, Pri: pri, Sec: secLine, SecOK: secOK,
		Up: upLine, UpOK: upOK, Face: face, Path: pathLine, PathOK: pathOK,
		Hint: hint, TS: time.Now().Unix(),
	}
}

func channelWidgetChatDataJSON() (string, error) {
	b, err := json.Marshal(buildChannelWidgetData())
	if err != nil {
		return "", err
	}
	if len(b) > 1024 {
		return "", fmt.Errorf("widget chat_data %d > 1024", len(b))
	}
	return string(b), nil
}

func handleChannelWidgetAct(token string, q *widgetQuery, admin int64, act string) bool {
	chat := admin
	if q.Chat != nil {
		chat = q.Chat.ID
	}
	switch act {
	case "refresh", "":
		return false
	case "status":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"status"}`)
		if tryReplyWidget(token, chat, q.MessageID, "status") {
			return true
		}
		reply(token, chat, q.MessageID, formatStatusPretty(), statusKeyboard())
		return true
	case "menu":
		answerWidgetQuery(token, q.ID, `{"ok":true,"nav":"menu"}`)
		reply(token, chat, q.MessageID, T("main_menu"), mainKeyboard())
		return true
	default:
		return false
	}
}

func init() {
	registerWidget(&widgetScreen{
		ID:        "channel",
		HTML:      channelWidgetHTML(),
		BuildData: channelWidgetChatDataJSON,
		HandleAct: handleChannelWidgetAct,
	})
}
