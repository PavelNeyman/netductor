package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// Channel HTML widget — Phase 1 read-only family + T5 rates/RTT.

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
.spk{font-family:ui-monospace,monospace;letter-spacing:1px}
</style></head><body>
<h1 id="title">Channel</h1>
<div class="row"><span class="k">secondary</span><span class="v" id="sec">—</span></div>
<div class="row"><span class="k">uplink</span><span class="v" id="up">—</span></div>
<div class="row"><span class="k">face</span><span class="v" id="face">—</span></div>
<div class="row"><span class="k">rtt</span><span class="v" id="rtt">—</span></div>
<div class="row"><span class="k">egress</span><span class="v" id="eg">—</span></div>
<div class="row"><span class="k">spark</span><span class="v spk" id="spk">—</span></div>
<div class="nav">
<button class="primary" data-act="channel:refresh">↻ Refresh</button>
<button data-act="channel:status">Status</button>
<button data-act="channel:menu">Menu</button>
</div>
<div class="hint" id="hint">Channel path summary</div>
<script>` + widgetBootJS + `
(function(){
function set(id,t,cls){var el=document.getElementById(id);if(!el)return;el.textContent=t||'—';el.className='v'+(cls?' '+cls:'');}
function paint(d){if(!d)return;set('title',d.title||'Channel');set('sec',d.sec,d.sec_ok===false?'bad':(d.sec_ok?'ok':''));
set('up',d.up,d.up_ok===false?'bad':(d.up_ok?'ok':''));set('face',d.face);set('rtt',d.rtt);set('eg',d.eg);
set('spk',d.spk);if(d.hint){var h=document.getElementById('hint');if(h)h.textContent=d.hint;}}
__ndBind(paint);
document.querySelectorAll('[data-act]').forEach(function(btn){
btn.addEventListener('click',function(){__ndSend(btn.getAttribute('data-act')||'channel:refresh');});
});
})();
</script></body></html>`
}

type channelWidgetData struct {
	Title string `json:"title"`
	Sec   string `json:"sec"`
	SecOK *bool  `json:"sec_ok,omitempty"`
	Up    string `json:"up"`
	UpOK  *bool  `json:"up_ok,omitempty"`
	Face  string `json:"face"`
	RTT   string `json:"rtt"`
	Eg    string `json:"eg"`
	Spk   string `json:"spk"`
	Hint  string `json:"hint,omitempty"`
	TS    int64  `json:"ts"`
}

func buildChannelWidgetData() channelWidgetData {
	ru := getLang() != "en"
	title, hint := "Channel", "RTT from agent · face = P→S:443"
	if ru {
		title, hint = "Канал", "RTT с агента · face = P→S:443"
	}
	d := channelWidgetData{Title: title, Hint: hint, TS: time.Now().Unix(), Sec: "—", Up: "—", Face: "—", RTT: "—", Eg: "—", Spk: "—"}
	out, err := exec.Command(netductorBin(), "channel", "status", "--json").CombinedOutput()
	if err != nil && len(out) == 0 {
		return d
	}
	var rep struct {
		Secondaries []struct {
			Name        string  `json:"name"`
			ID          string  `json:"id"`
			Online      bool    `json:"online"`
			UplinkOK    bool    `json:"uplink_ok"`
			UplinkRTTMs float64 `json:"uplink_rtt_ms"`
			TCP443OK    bool    `json:"tcp443_ok"`
			TCP443ms    float64 `json:"tcp443_ms"`
			EgressRxBps float64 `json:"egress_rx_bps"`
			EgressTxBps float64 `json:"egress_tx_bps"`
		} `json:"secondaries"`
	}
	if json.Unmarshal(out, &rep) != nil || len(rep.Secondaries) == 0 {
		return d
	}
	s := rep.Secondaries[0]
	name := s.Name
	if name == "" {
		name = s.ID
	}
	ok := s.Online
	d.SecOK = &ok
	if s.Online {
		d.Sec = name + " online"
	} else {
		d.Sec = name + " offline"
	}
	uok := s.UplinkOK
	d.UpOK = &uok
	if s.UplinkOK {
		d.Up = "ok"
	} else {
		d.Up = "down"
	}
	if s.TCP443OK {
		d.Face = fmt.Sprintf("ok %.0fms", s.TCP443ms)
	} else {
		d.Face = "fail"
	}
	if s.UplinkRTTMs > 0 {
		d.RTT = fmt.Sprintf("%.0fms", s.UplinkRTTMs)
	}
	if s.EgressRxBps > 0 || s.EgressTxBps > 0 {
		d.Eg = fmt.Sprintf("%.1f↓/%.1f↑ Mbps", s.EgressRxBps/1e6, s.EgressTxBps/1e6)
	}
	d.Spk = channelHistorySpark()
	if d.Spk == "" {
		d.Spk = "—"
	}
	return d
}

func channelWidgetChatDataJSON() (string, error) {
	b, err := json.Marshal(buildChannelWidgetData())
	if err != nil {
		return "", err
	}
	if len(b) > 1024 {
		// drop spark first
		var d channelWidgetData
		_ = json.Unmarshal(b, &d)
		d.Spk = ""
		d.Hint = ""
		b, err = json.Marshal(d)
		if err != nil {
			return "", err
		}
		if len(b) > 1024 {
			return "", fmt.Errorf("widget chat_data %d > 1024", len(b))
		}
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
