package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Dual info+nav pair (NETDUCTOR_TG_DUAL=1). Rules: docs/AI-RULES-TG-WIDGETS.md
//
// Dual ON: info = metrics; nav = interactive hub. SPA multi-HTML was removed;
// revive only under zlib pressure (docs/PLAN-TG-WIDGET-UI-V2.md).
//

type dualPair struct {
	InfoID    int
	NavID     int
	InfoShell string
	NavShell  string
}

var (
	lastDual   = map[int64]dualPair{}
	lastDualMu sync.Mutex
)

func dualEnabled() bool {
	loadWidgetUIStore()
	return widgetsEnabled() && os.Getenv("NETDUCTOR_TG_DUAL") == "1"
}

func getDual(chat int64) (dualPair, bool) {
	loadWidgetUIStore()
	lastDualMu.Lock()
	defer lastDualMu.Unlock()
	p, ok := lastDual[chat]
	return p, ok && (p.InfoID != 0 || p.NavID != 0)
}

func setDual(chat int64, p dualPair) {
	lastDualMu.Lock()
	lastDual[chat] = p
	lastDualMu.Unlock()
	saveWidgetUIStore()
}

func forgetDual(chat int64) {
	lastDualMu.Lock()
	delete(lastDual, chat)
	lastDualMu.Unlock()
	saveWidgetUIStore()
}

// closeDualPair deletes both messages and clears state.
func closeDualPair(token string, chat int64) {
	loadWidgetUIStore()
	p, ok := getDual(chat)
	if !ok {
		return
	}
	forgetDual(chat)
	setLastAppMsg(chat, 0)
	if p.InfoID > 0 {
		_ = deleteMessage(token, chat, p.InfoID)
	}
	if p.NavID > 0 {
		_ = deleteMessage(token, chat, p.NavID)
	}
}

// openDualPair sends info then nav. On failure closes orphans.
func openDualPair(token string, chat int64, infoHTML, infoCD, navHTML, navCD string, infoShell, navShell string) error {
	closeDualPair(token, chat) // no orphan previous pair

	infoID, err := sendWidgetMessage(token, chat, infoHTML, infoCD)
	if err != nil {
		return fmt.Errorf("dual info: %w", err)
	}
	navID, err := sendWidgetMessage(token, chat, navHTML, navCD)
	if err != nil {
		_ = deleteMessage(token, chat, infoID)
		return fmt.Errorf("dual nav: %w", err)
	}
	setDual(chat, dualPair{
		InfoID: infoID, NavID: navID,
		InfoShell: infoShell, NavShell: navShell,
	})
	syncDualNavAsApp(chat)
	return nil
}

func sendWidgetMessage(token string, chat int64, html, cd string) (int, error) {
	raw, err := apiPostWidget(token, "sendWidget", map[string]any{
		"chat_id": chat, "html": html, "chat_data": cd, "allow_queries": true,
	})
	if err != nil {
		return 0, err
	}
	var wr struct {
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &wr) != nil || wr.Result.MessageID == 0 {
		return 0, fmt.Errorf("sendWidget: no message_id")
	}
	return wr.Result.MessageID, nil
}

func editWidgetData(token string, chat int64, msgID int, cd string) error {
	if msgID <= 0 {
		return fmt.Errorf("editWidgetData: bad msg id")
	}
	_, err := apiPostWidget(token, "editMessageWidgetData", map[string]any{
		"chat_id": chat, "message_id": msgID, "chat_data": cd,
	})
	return err
}

// editDualSlot edits one slot; on failure retries once then recreates slot.
func editDualSlot(token string, chat int64, slot string, html, cd, shell string) {
	p, ok := getDual(chat)
	if !ok {
		return
	}
	var id int
	if slot == "info" {
		id = p.InfoID
	} else {
		id = p.NavID
	}
	err := editWidgetData(token, chat, id, cd)
	if err != nil {
		err = editWidgetData(token, chat, id, cd)
	}
	if err == nil {
		if shell != "" {
			if slot == "info" {
				p.InfoShell = shell
			} else {
				p.NavShell = shell
			}
			setDual(chat, p)
		}
		return
	}
	fmt.Fprintln(os.Stderr, "dual edit", slot, err, "→ recreate")
	newID, err2 := sendWidgetMessage(token, chat, html, cd)
	if err2 != nil {
		fmt.Fprintln(os.Stderr, "dual recreate", slot, err2)
		return
	}
	if id > 0 && id != newID {
		_ = deleteMessage(token, chat, id)
	}
	// keep the other slot from the snapshot — do not re-getDual (may race empty)
	if slot == "info" {
		p.InfoID = newID
		p.InfoShell = shell
	} else {
		p.NavID = newID
		p.NavShell = shell
	}
	setDual(chat, p)
}

// swapDualSlot replaces HTML shell for one slot only.
func swapDualSlot(token string, chat int64, slot, html, cd, shell string) {
	p, ok := getDual(chat)
	if !ok {
		return
	}
	var old int
	if slot == "info" {
		old = p.InfoID
	} else {
		old = p.NavID
	}
	newID, err := sendWidgetMessage(token, chat, html, cd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dual swap", slot, err)
		return
	}
	if old > 0 && old != newID {
		_ = deleteMessage(token, chat, old)
	}
	if slot == "info" {
		p.InfoID = newID
		p.InfoShell = shell
	} else {
		p.NavID = newID
		p.NavShell = shell
	}
	setDual(chat, p)
}

// pushDualPair updates nav (+ answers query) and info after a nav action.
func pushDualPair(token string, q *widgetQuery, infoCD, navCD, infoShell, navShell string) {
	if q == nil || q.Chat == nil {
		return
	}
	chat := q.Chat.ID
	// Answer is only the request ack (≤1024). State is applied via editMessageWidgetData.
	// Do not put full chat_data in answerWidgetQuery — SPA paints from chatDataChanged.
	if nav2, err := clipWidgetChatData(navCD); err == nil {
		navCD = nav2
	}
	if info2, err := clipWidgetChatData(infoCD); err == nil {
		infoCD = info2
	}
	// AI-RULES §1.3: answer payload = nav chat_data (≤1024). State still via edit*.
	answerWidgetQuery(token, q.ID, navCD)

	p, ok := getDual(chat)
	if !ok {
		// recover: open pair from scratch
		_ = openDualPair(token, chat, appInfoHTML(), infoCD, appWidgetHTML(), navCD, infoShell, shellHub)
		return
	}

	navShell = shellHub
	editDualSlot(token, chat, "nav", appWidgetHTML(), navCD, navShell)

	// Info: one metrics shell; "spark" kind only marks that series may be present.
	infoHTML := appInfoHTML()
	p, _ = getDual(chat)
	if p.InfoShell != infoShell && infoShell != "" && p.InfoID > 0 {
		// Prefer edit; swap only if we ever introduce a second info HTML.
		editDualSlot(token, chat, "info", infoHTML, infoCD, infoShell)
		p, _ = getDual(chat)
		p.InfoShell = infoShell
		setDual(chat, p)
	} else {
		editDualSlot(token, chat, "info", infoHTML, infoCD, infoShell)
	}
	syncDualNavAsApp(chat)
}

// infoShellForScreen picks info HTML kind (spark when metrics series useful).
func infoShellForScreen(screen string) string {
	switch screen {
	case "home", "status", "":
		return shellSpark
	default:
		return shellHub
	}
}

// dualSplitChatData splits a full SPA appState JSON into nav (controls) + info (content).
func dualSplitChatData(full string) (navCD, infoCD string) {
	var st appState
	if json.Unmarshal([]byte(full), &st) != nil {
		return full, `{"s":"info","t":"Status"}`
	}
	info := appState{
		S: st.S, T: st.T, B: st.B, H: st.H,
		Rows: st.Rows, Lines: st.Lines, Spk: st.Spk, Pct: st.Pct,
	}
	if st.Pad && st.Draft != "" && info.B == "" {
		info.B = st.Draft
	}
	if info.T == "" {
		info.T = st.S
	}
	if info.H == "" {
		info.H = appTitle("content · controls below", "контент · управление ниже")
	}
	nav := appState{
		S: st.S, T: st.T, H: st.H, L: st.L,
		Nav: st.Nav, Acts: st.Acts, Segs: st.Segs, Seg: st.Seg, IDs: st.IDs,
		Btns: st.Btns, Draft: st.Draft, Pad: st.Pad,
	}
	// Keep a one-line summary on nav if no acts/nav at all
	if nav.Nav == nil && nav.Acts == "" && nav.Segs == nil {
		nav.B = appTitle("refresh · back", "обновить · назад")
	}
	nb, err1 := appMarshal(nav)
	if err1 != nil {
		nb = full
	}
	ib, err2 := appMarshal(info)
	if err2 != nil {
		// shrink lines until fit
		for lim := 400; lim >= 80; lim -= 80 {
			info.Lines = widgetTruncate(info.Lines, lim)
			if len(info.Rows) > 4 {
				info.Rows = info.Rows[:4]
			}
			ib, err2 = appMarshal(info)
			if err2 == nil {
				break
			}
		}
		if err2 != nil {
			ib = `{"s":"info","t":"Status"}`
		}
	}
	if _, err := clipWidgetChatData(ib); err != nil {
		info.Lines = widgetTruncate(info.Lines, 200)
		info.Spk = ""
		if ib2, e := appMarshal(info); e == nil {
			ib = ib2
		}
	}
	return nb, ib
}

// buildInfoData — dual top pane: rows/lines/spark from the same screen as nav.
func buildInfoData(screen string) (string, error) {
	if screen == "" {
		screen = "home"
	}
	full, err := buildAppScreen(screen)
	if err != nil {
		// fallback status snapshot
		d := buildStatusWidgetData()
		st := appState{
			S: "info", T: d.Title,
			Rows: [][2]string{
				{appTitle("version", "версия"), d.Ver},
				{appTitle("host", "хост"), d.Host},
				{appTitle("cpu/mem", "cpu/mem"), d.Res},
			},
		}
		if spk := sparkCPU(24); spk != "" {
			st.Spk = spk
		}
		return appMarshal(st)
	}
	_, infoCD := dualSplitChatData(full)
	return infoCD, nil
}

func appInfoHTML() string {
	return appInfoHTMLBase()
}

func appInfoHTMLSpark() string { // test alias
	return appInfoHTML()
}

func appInfoHTMLBase() string {
	css := `body{margin:0;padding:8px;font:13px/1.3 system-ui,sans-serif;background:var(--tg-theme-bg-color,#0f1419);color:var(--tg-theme-text-color,#e7ecf3)}
h1{margin:0 0 6px;font-size:15px}
.row{display:flex;justify-content:space-between;gap:6px;margin:3px 0}
.k{color:var(--tg-theme-hint-color,#8b98a5)}.v{font-weight:600;text-align:right;word-break:break-word}
.bar{height:5px;border-radius:3px;background:var(--tg-theme-secondary-bg-color,#1c2733);margin:1px 0 5px;overflow:hidden}
.bar>i{display:block;height:100%;background:var(--tg-theme-button-color,#2a6df4)}
#h{margin-top:6px;color:var(--tg-theme-hint-color,#8b98a5);font-size:11px}
button{border:0;border-radius:8px;padding:8px 12px;margin-top:6px;background:var(--tg-theme-button-color,#2a6df4);color:#fff;font-weight:600}`
	boot := `(function(){var W=window.TelegramWidget;
function esc(s){return String(s==null?"":s).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;");}
function paint(d){if(!d)return;var t=document.getElementById("t"),b=document.getElementById("b"),h=document.getElementById("h");
if(t)t.textContent=d.t||"Status";if(h)h.textContent=d.h||"";
if(b){var html="";if(d.rows){for(var i=0;i<d.rows.length;i++){var r=d.rows[i];
html+='<div class="row"><span class="k">'+esc(r[0]||"")+'</span><span class="v">'+esc(r[1]||"—")+'</span></div>';
if(d.pct&&d.pct[i]!=null&&d.pct[i]>=0){var p=Math.max(0,Math.min(100,+d.pct[i]||0));html+='<div class="bar"><i style="width:'+p+'%"></i></div>';}}
}if(d.spk)html+='<canvas id="spk" width="300" height="36" style="width:100%;height:36px"></canvas>';
if(d.lines)html+='<pre style="white-space:pre-wrap;font:12px system-ui"></pre>';
b.innerHTML=html;if(d.lines){var pre=b.querySelector("pre");if(pre)pre.textContent=d.lines;}
if(d.spk){var c=document.getElementById("spk");if(c&&c.getContext){var a=String(d.spk).split(","),ctx=c.getContext("2d"),w=c.width,hgt=c.height,n=a.length;
if(n>1){ctx.strokeStyle="#2a6df4";ctx.lineWidth=2;ctx.beginPath();for(var j=0;j<n;j++){var v=Math.max(0,Math.min(100,+a[j]||0));var x=j/(n-1)*(w-2)+1,y=hgt-1-(v/100)*(hgt-4);if(j===0)ctx.moveTo(x,y);else ctx.lineTo(x,y);}ctx.stroke();}}}}
var r=document.getElementById("r");if(r)r.onclick=function(){if(W&&W.request)W.request("r");};
}function run(){try{var x=(W&&W.chatData);if(typeof x==="string")x=JSON.parse(x||"{}");paint(x||{});}catch(e){}}
run();if(W&&W.onEvent)W.onEvent("chatDataChanged",run);
// auto-refresh top info pane (default on)
setInterval(function(){try{if(W&&W.request)W.request("r");}catch(e){}},15000);
})();`
	return `<!DOCTYPE html><html><head><meta charset="utf-8"/><style>` + css + `</style></head><body>
<h1 id="t">Status</h1><div id="b"></div><div id="h"></div>
<button id="r" type="button">↻</button>
<script>` + boot + `</script></body></html>`
}
