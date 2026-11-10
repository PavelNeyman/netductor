package main

import (
	"bytes"
	"sync"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Phase 0 foundation: HTML Widgets registry (Bot API 10.4).
// Gate: NETDUCTOR_TG_WIDGETS=1 (default off). See docs/PLAN-TG-HTML-WIDGETS-MENU.md.

type widgetScreen struct {
	ID        string
	HTML      string
	BuildData func() (string, error)
	// HandleAct processes q.Data after "id:" prefix stripped (e.g. "refresh").
	// Return true if handled. May call reply() for classic nav.
	HandleAct func(token string, q *widgetQuery, admin int64, act string) bool
}

var widgetRegistry = map[string]*widgetScreen{}

// lastAppMsgID remembers the SPA widget message per chat to edit in-place.
var lastAppMsgID = map[int64]int{}
var lastAppMu sync.Mutex

func forgetAppMsg(chat int64) {
	lastAppMu.Lock()
	delete(lastAppMsgID, chat)
	lastAppMu.Unlock()
}

func registerWidget(s *widgetScreen) {
	if s == nil || s.ID == "" {
		return
	}
	widgetRegistry[s.ID] = s
}

func widgetsEnabled() bool { return os.Getenv("NETDUCTOR_TG_WIDGETS") == "1" }

func widgetAPIBase(token string) string {
	if os.Getenv("NETDUCTOR_TG_WIDGET_BETA") == "1" {
		return "https://api.telegram.org/beta/bot" + token + "/"
	}
	return "https://api.telegram.org/bot" + token + "/"
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

func answerWidgetQuery(token, queryID, result string) {
	// Bot API 10.4: parameter is "data" (not "result"); widget promise gets { result: data }.
	payload := map[string]any{"widget_query_id": queryID}
	if result != "" {
		payload["data"] = result
	}
	if _, err := apiPostWidget(token, "answerWidgetQuery", payload); err != nil {
		fmt.Fprintln(os.Stderr, "answerWidgetQuery:", err)
	}
}

func sendWidgetScreen(token string, chat int64, screenID string) bool {
	if !widgetsEnabled() {
		return false
	}
	s := widgetRegistry[screenID]
	if s == nil || s.HTML == "" || s.BuildData == nil {
		return false
	}
	cd, err := s.BuildData()
	if err != nil {
		fmt.Fprintln(os.Stderr, "widget chat_data:", screenID, err)
		return false
	}
	if cd, err = clipWidgetChatData(cd); err != nil {
		fmt.Fprintln(os.Stderr, "widget chat_data:", screenID, err)
		return false
	}
	raw, err := apiPostWidget(token, "sendWidget", map[string]any{
		"chat_id": chat, "html": s.HTML, "chat_data": cd, "allow_queries": true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sendWidget:", screenID, err)
		return false
	}
	if screenID == "app" {
		var wr struct {
			Result struct {
				MessageID int `json:"message_id"`
			} `json:"result"`
		}
		if json.Unmarshal(raw, &wr) == nil && wr.Result.MessageID > 0 {
			lastAppMu.Lock()
			lastAppMsgID[chat] = wr.Result.MessageID
			lastAppMu.Unlock()
		}
	}
	return true
}

func editWidgetScreenData(token string, chat int64, msgID int, screenID string) bool {
	if !widgetsEnabled() || msgID <= 0 {
		return false
	}
	s := widgetRegistry[screenID]
	if s == nil || s.BuildData == nil {
		return false
	}
	cd, err := s.BuildData()
	if err != nil {
		fmt.Fprintln(os.Stderr, "widget chat_data:", screenID, err)
		return false
	}
	if cd, err = clipWidgetChatData(cd); err != nil {
		fmt.Fprintln(os.Stderr, "widget chat_data:", screenID, err)
		return false
	}
	_, err = apiPostWidget(token, "editMessageWidgetData", map[string]any{
		"chat_id": chat, "message_id": msgID, "chat_data": cd,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "editMessageWidgetData:", screenID, err)
		return false
	}
	return true
}

// spaScreenAlias maps legacy per-screen widget ids → app SPA screen ids.
var spaScreenAlias = map[string]string{
	"sshhosts": "ssh",
	"menu":     "home",
}

// spaContentScreens are shown inside the single app shell (same HTML).
// Routing classic m:* callbacks here avoids empty UI from editing chat_data
// of a different HTML document.
var spaContentScreens = map[string]bool{
	"home": true, "status": true, "channel": true, "versions": true,
	"secondary": true, "probes": true, "netmap": true, "digest": true,
	"fleet": true, "users": true, "nodes": true, "devices": true,
	"sites": true, "pending": true, "ssh": true, "backup": true,
	"seclist": true, "operator": true, "addons": true, "svcpaths": true,
	"lampac": true, "canary": true, "sessions": true, "audit": true,
	"actions": true, "routers": true,
	"tools": true, "edge": true, "media": true, "data": true, "adv": true,
	"logs": true, "stack": true, "help": true, "ucard": true, "ncard": true,
	"ulist": true, "incident": true, "reboot": true, "dns": true, "git": true,
	"certs": true, "fw": true, "doctor": true, "cleanup": true, "admin": true,
	"session": true, "alerts": true, "nvr": true,
	"edgeguest": true, "edgeluci": true, "sshkeys": true, "loc": true, "prdraft": true, "edgelife": true, "secops": true, "mtls": true, "presets": true, "edgetpl": true, "edgepol": true, "edgestor": true, "aptguard": true,
}

func openAppScreen(token string, chat int64, screenID string) bool {
	if !widgetsEnabled() {
		return false
	}
	if alt, ok := spaScreenAlias[screenID]; ok {
		screenID = alt
	}
	if screenID == "" {
		screenID = "home"
	}
	appMu.Lock()
	appScreen = screenID
	appMu.Unlock()
	// Prefer in-place edit of the last SPA message in this chat.
	lastAppMu.Lock()
	mid := lastAppMsgID[chat]
	lastAppMu.Unlock()
	if mid > 0 {
		cd, err := buildAppScreen(screenID)
		if err == nil {
			if cd2, err2 := clipWidgetChatData(cd); err2 == nil {
				cd = cd2
				if _, err := apiPostWidget(token, "editMessageWidgetData", map[string]any{
					"chat_id": chat, "message_id": mid, "chat_data": cd,
				}); err == nil {
					return true
				}
			}
		}
		// Stale id after chat clear / delete — force a new sendWidget.
		forgetAppMsg(chat)
	}
	return sendWidgetScreen(token, chat, "app")
}

// tryReplyWidget: prefer SPA app shell for content screens; else legacy per-screen widget.
func tryReplyWidget(token string, chat int64, msgID int, screenID string) bool {
	if !widgetsEnabled() {
		return false
	}
	if alt, ok := spaScreenAlias[screenID]; ok {
		screenID = alt
	}
	if spaContentScreens[screenID] || screenID == "app" {
		return openAppScreen(token, chat, screenID)
	}
	// Cards / one-off screens: edit if possible, else send that HTML.
	if msgID > 0 && editWidgetScreenData(token, chat, msgID, screenID) {
		return true
	}
	return sendWidgetScreen(token, chat, screenID)
}

// parseWidgetAct splits "status:refresh" or legacy "refresh" → screen, act.
func parseWidgetAct(data string) (screen, act string) {
	data = strings.TrimSpace(data)
	if data == "" {
		return "app", "r"
	}
	// SPA short acts (single app widget)
	if data == "r" || data == "b" || data == "m" || data == "refresh" || data == "back" || data == "menu" ||
		data == "en" || data == "di" || data == "lk" || data == "sb" || data == "rv" || data == "y_rv" ||
		data == "up" || data == "rb" || data == "jn" || data == "y_rb" ||
		data == "api_tg" || data == "usp_tg" || data == "vpnref" ||
		data == "q50" || data == "q200" || data == "q0" || data == "rn" || data == "add" ||
		data == "heal" || data == "rbk" || data == "y_rbk" || data == "wd" || data == "y_rst" ||
		data == "al_test" || data == "al_clr" || data == "qn" || data == "qr" || data == "sess1" || data == "sess4" || data == "sess24" || data == "sess72" || data == "sessx" || data == "srev" || data == "cy" || data == "sk_on" || data == "sk_off" || data == "sk_scan" || data == "cert_ren" || data == "fw_apply" || data == "ag_st" || data == "ag_prev" || data == "ag_scan" || data == "nrn" || data == "mx" || data == "gbuild" || data == "gmirror" || data == "gjobs" || data == "gadd" || data == "gmig" || data == "gpause" || data == "gresume" || data == "grm" || data == "el_1" || data == "el_4" || data == "el_24" || data == "el_72" || data == "el_ext" || data == "es_st" || data == "ep_st" || data == "et_st" || data == "eg_facts" || data == "eg_code" || data == "pd_net" || data == "pd_mode" || data == "pd_save" || data == "pr_draft" || data == "up_inet" || data == "up_mode" || data == "upol" || data == "spa_apply" || data == "room_add" || data == "et_dns_vpn" || data == "et_dns_wan" || data == "et_dns_off" || data == "et_mode_tun" || data == "et_mode_off" || data == "et_fb_wan" || data == "et_fb_block" || data == "et_soft_on" || data == "et_soft_off" || data == "pr_new" || data == "mt_list" || data == "mt_rot" || data == "mt_roll" || data == "sec_exp" || data == "sec_ol" || data == "sec_enr" || data == "sec_sync" || data == "er_reg" || data == "er_bind" || data == "er_apply" || data == "er_site" || data == "er_rec" || data == "loc_home" || data == "loc_off" || data == "ep_inet" || data == "ep_mode" || data == "es_fmt" || data == "y_fmt" || data == "et_fb" || data == "et_soft" || data == "glog" || data == "et_mode" || data == "et_dns" || data == "et_off" || data == "et_on" || data == "es_mute" || data == "es_list" || data == "es_rst" || data == "es_bak" || data == "kd" || data == "ku" || data == "sp" || data == "rs" || data == "gsall" || data == "inc1h" ||
		data == "gsync" || data == "gship" || data == "eg_gr" || data == "eg_rv" ||
		data == "el_on" || data == "el_off" ||
		strings.HasPrefix(data, "rc:") || strings.HasPrefix(data, "p:") || strings.HasPrefix(data, "ps:") || strings.HasPrefix(data, "us:") ||
		strings.HasPrefix(data, "g:") || strings.HasPrefix(data, "o:") || strings.HasPrefix(data, "L:") {
		return "app", data
	}
	if i := strings.IndexByte(data, ':'); i > 0 {
		return data[:i], data[i+1:]
	}
	return "app", data
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
	screenID, act := parseWidgetAct(q.Data)
	if screenID == "" {
		screenID = "app"
	}
	s := widgetRegistry[screenID]
	if s == nil {
		answerWidgetQuery(token, q.ID, `{"err":"unknown_screen"}`)
		return true
	}
	if s.HandleAct != nil && s.HandleAct(token, q, admin, act) {
		return true
	}
	// default refresh
	if act == "refresh" || act == "" {
		if _, err := s.BuildData(); err != nil {
			answerWidgetQuery(token, q.ID, `{"err":"build"}`)
			return true
		}
		if q.MessageID != 0 && q.Chat != nil {
			_ = editWidgetScreenData(token, q.Chat.ID, q.MessageID, screenID)
		}
		answerWidgetQuery(token, q.ID, `{"ok":true}`)
		return true
	}
	answerWidgetQuery(token, q.ID, `{"err":"unknown"}`)
	return true
}
