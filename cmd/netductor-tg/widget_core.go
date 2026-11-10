package main

import (
	"bytes"
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
	_, err = apiPostWidget(token, "sendWidget", map[string]any{
		"chat_id": chat, "html": s.HTML, "chat_data": cd, "allow_queries": true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sendWidget:", screenID, err)
		return false
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

// tryReplyWidget: edit existing widget message data, else send new widget.
func tryReplyWidget(token string, chat int64, msgID int, screenID string) bool {
	if !widgetsEnabled() {
		return false
	}
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
		strings.HasPrefix(data, "g:") || strings.HasPrefix(data, "o:") {
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
