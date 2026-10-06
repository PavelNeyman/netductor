package main

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/notify"
)

func registerTGAlertsAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/telegram/alerts-chat", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, notify.GetAlertsRoutingStatus())
		case http.MethodPost, http.MethodPut:
			body := readJSON(r)
			if clear, _ := body["clear"].(bool); clear {
				_ = notify.ClearAlertsChatID()
				_ = exec.Command("systemctl", "try-restart", "netductor-telegram-bot").Run()
				writeJSON(w, 200, notify.GetAlertsRoutingStatus())
				return
			}
			id, _ := body["chat_id"].(string)
			if id == "" {
				if n, ok := body["chat_id"].(float64); ok {
					id = strconv.FormatInt(int64(n), 10)
				}
			}
			id = strings.TrimSpace(id)
			if id == "" {
				writeJSON(w, 400, map[string]string{"error": "chat_id required (or clear:true)"})
				return
			}
			if err := notify.SetAlertsChatID(id); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			_ = exec.Command("systemctl", "try-restart", "netductor-telegram-bot").Run()
			writeJSON(w, 200, notify.GetAlertsRoutingStatus())
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})
}
