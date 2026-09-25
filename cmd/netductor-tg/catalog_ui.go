package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/PavelNeyman/netductor/internal/opcatalog"
	"github.com/PavelNeyman/netductor/internal/session"
)

func catalogLang() string {
	if getLang() == "en" {
		return "en"
	}
	return "ru"
}

// toolsKeyboard builds Tools hub from opcatalog sections + a few product hubs not in catalog.
func toolsKeyboard() map[string]any {
	lang := catalogLang()
	var rows [][]map[string]any
	// catalog sections
	var row []map[string]any
	for _, sec := range opcatalog.Sections() {
		label := sec
		switch sec {
		case "overview":
			if lang == "ru" {
				label = "Обзор"
			} else {
				label = "Overview"
			}
		case "vpn":
			label = "VPN"
		case "nodes":
			if lang == "ru" {
				label = "Ноды"
			} else {
				label = "Nodes"
			}
		case "edge":
			label = "Edge"
		case "nvr":
			label = "NVR"
		case "git":
			label = "Git"
		case "backup":
			if lang == "ru" {
				label = "Бэкап"
			} else {
				label = "Backup"
			}
		case "probes":
			label = "Probes"
		}
		row = append(row, btn(label, "m:ops:"+sec, ""))
		if len(row) == 2 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	// product-specific hubs (not pure session GET/POST)
	rows = append(rows,
		[]map[string]any{btn("⏱ Guest VPN", "m:guest", ""), btn("📡 Guest Wi‑Fi", "m:edgeguest", "")},
		[]map[string]any{btn("🛡 DNS", "m:dns", ""), btn("📍 Locations", "m:loc", "")},
		[]map[string]any{btn("🔄 Updates", "m:updates", ""), btn(T("mtls"), "m:mtls", "")},
		[]map[string]any{btn(T("main_menu"), "m:menu", "primary")},
	)
	return map[string]any{"inline_keyboard": rows}
}

func catalogSectionKeyboard(sec string) map[string]any {
	lang := catalogLang()
	var rows [][]map[string]any
	var row []map[string]any
	for _, a := range opcatalog.ForSurface("tg") {
		if a.Section != sec {
			continue
		}
		row = append(row, btn(a.Label(lang), "m:op:"+a.ID, ""))
		if len(row) == 2 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, []map[string]any{btn("🧰 Tools", "m:tools", ""), btn(T("main_menu"), "m:menu", "")})
	return map[string]any{"inline_keyboard": rows}
}

func catalogSectionTitle(sec string) string {
	if catalogLang() == "ru" {
		return "📂 <b>" + sec + "</b>\nДействия из общего каталога (как Web Control)."
	}
	return "📂 <b>" + sec + "</b>\nActions from shared catalog (same as Web Control)."
}

// execCatalogAction calls node session API on localhost with a short-lived session.
func execCatalogAction(id string) string {
	a, ok := opcatalog.Get(id)
	if !ok {
		return "unknown action: " + id
	}
	tok, _, err := session.Create(1, "telegram-bot", "127.0.0.1")
	if err != nil {
		return "session: " + err.Error()
	}
	defer session.Revoke(tok)

	url := "http://127.0.0.1:8787" + a.Path
	var body io.Reader
	method := a.Method
	if method == "" {
		method = "GET"
	}
	if method == "POST" {
		b := a.Body
		if b == "" {
			b = "{}"
		}
		body = bytes.NewReader([]byte(b))
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 400 {
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	// pretty-print JSON if possible
	var v any
	if json.Unmarshal(raw, &v) == nil {
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	}
	return string(raw)
}


func opcatalogGetSection(id string) (string, bool) {
	a, ok := opcatalog.Get(id)
	if !ok {
		return "", false
	}
	return a.Section, true
}
