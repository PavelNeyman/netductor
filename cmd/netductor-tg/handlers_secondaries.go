package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/session"
)

func localAPI(method, path string, body []byte) ([]byte, int, error) {
	tok, _, err := session.Create(1, "telegram-bot", "127.0.0.1")
	if err != nil {
		return nil, 0, err
	}
	defer session.Revoke(tok)
	url := "http://127.0.0.1:8787" + path
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return b, resp.StatusCode, nil
}

func secondariesHubHTML() string {
	ru := getLang() != "en"
	b, code, err := localAPI("GET", "/api/secondaries", nil)
	if err != nil || code >= 400 {
		msg := "inventory unavailable"
		if ru {
			msg = "инвентарь недоступен"
		}
		if err != nil {
			return "📋 <b>Secondaries</b>\n" + esc(msg) + ": " + esc(err.Error())
		}
		return "📋 <b>Secondaries</b>\n" + esc(msg) + " HTTP " + fmt.Sprint(code)
	}
	var items []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		PublicIP  string `json:"public_ip"`
		Lifecycle string `json:"lifecycle_state"`
	}
	_ = json.Unmarshal(b, &items)
	var sb strings.Builder
	if ru {
		sb.WriteString("📋 <b>Secondary (кандидаты)</b>\n<i>Регистрация ≠ deploy.</i>\n\n")
	} else {
		sb.WriteString("📋 <b>Secondary candidates</b>\n<i>Register ≠ deploy.</i>\n\n")
	}
	if len(items) == 0 {
		sb.WriteString("<i>(empty)</i>\n")
	}
	for _, s := range items {
		sb.WriteString(fmt.Sprintf("• <code>%s</code> %s — <code>%s</code> [%s]\n",
			esc(s.ID), esc(s.Name), esc(s.PublicIP), esc(s.Lifecycle)))
	}
	return sb.String()
}

func secondariesKeyboard() map[string]any {
	ru := getLang() != "en"
	list, add := "🔄 Refresh", "➕ Register"
	if ru {
		list, add = "🔄 Обновить", "➕ Зарегистрировать"
	}
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(list, "m:sec:list", ""), btn(add, "m:sec:add", "")},
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func handleSecondariesCB(token string, chat int64, msgID int, data string) bool {
	switch data {
	case "m:sec", "m:sec:list":
		reply(token, chat, msgID, secondariesHubHTML(), secondariesKeyboard())
		return true
	case "m:sec:add":
		setState(chat, "sec_add", "await_ip")
		ru := getLang() != "en"
		msg := "Send: <code>name IP</code> or <code>IP</code>\nExample: <code>ru2 203.0.113.10</code>"
		if ru {
			msg = "Отправьте: <code>имя IP</code> или <code>IP</code>\nПример: <code>ru2 203.0.113.10</code>"
		}
		reply(token, chat, msgID, msg, secondariesKeyboard())
		return true
	}
	return false
}

func trySecAddMessage(token string, chat int64, text string) bool {
	st, _ := getState(chat)
	if st != "sec_add" {
		return false
	}
	setState(chat, "", "")
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) == 0 {
		return true
	}
	name, ip := "", ""
	if len(parts) == 1 {
		ip = parts[0]
	} else {
		name, ip = parts[0], parts[1]
	}
	body, _ := json.Marshal(map[string]string{"name": name, "public_ip": ip})
	b, code, err := localAPI("POST", "/api/secondaries", body)
	ru := getLang() != "en"
	if err != nil || code >= 400 {
		msg := "register failed"
		if ru {
			msg = "ошибка регистрации"
		}
		detail := ""
		if err != nil {
			detail = err.Error()
		} else {
			detail = fmt.Sprintf("HTTP %d %s", code, string(b))
		}
		reply(token, chat, 0, esc(msg)+": "+esc(detail), secondariesKeyboard())
		return true
	}
	ok := "✅ registered (candidate — deploy from Mac)"
	if ru {
		ok = "✅ зарегистрирован (кандидат — deploy с Mac)"
	}
	reply(token, chat, 0, ok+"\n\n"+secondariesHubHTML(), secondariesKeyboard())
	return true
}
