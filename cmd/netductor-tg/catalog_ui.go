package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/format"
	"github.com/PavelNeyman/netductor/internal/opcatalog"
	"github.com/PavelNeyman/netductor/internal/session"
)

func catalogLang() string {
	if getLang() == "en" {
		return "en"
	}
	return "ru"
}

// toolsKeyboard — navigation only; section links in toolsHubHTML body.
func toolsKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func toolsHubHTML() string {
	// Stack apply lives under Operator; Tools is day-2 catalog only (PLAN-TG-MENU).
	return formatToolsFromGroups()
}

func catalogSectionKeyboard(sec string) map[string]any {
	// Navigation only — actions live in HTML body (TG-UI pattern).
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}

func catalogSectionTitle(sec string) string {
	lang := catalogLang()
	var b strings.Builder
	if lang == "ru" {
		b.WriteString("📂 <b>" + sec + "</b>\nДействия из общего каталога (как Web Control).\n")
	} else {
		b.WriteString("📂 <b>" + sec + "</b>\nActions from shared catalog (same as Web Control).\n")
	}
	acts := opcatalog.ForSurface("tg")
	var ids []string
	var labels []string
	for _, a := range acts {
		if a.Section != sec {
			continue
		}
		ids = append(ids, a.ID)
		labels = append(labels, a.Label(lang))
	}
	if len(ids) == 0 {
		if lang == "ru" {
			b.WriteString("<i>нет действий</i>")
		} else {
			b.WriteString("<i>no actions</i>")
		}
		return b.String()
	}
	b.WriteString("<table>\n<tr><th>#</th><th>action</th></tr>\n")
	for i, lab := range labels {
		b.WriteString("<tr><td>" + fmt.Sprint(i+1) + "</td><td>" + esc(lab) + "</td></tr>\n")
	}
	b.WriteString("</table>\n")
	// body buttons in rows of up to 4
	for i := 0; i < len(ids); i += 4 {
		b.WriteString("<tg-button-row>")
		end := i + 4
		if end > len(ids) {
			end = len(ids)
		}
		for j := i; j < end; j++ {
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:op:%s">%d</tg-button>`, ids[j], j+1))
		}
		b.WriteString("</tg-button-row>")
	}
	return b.String()
}

func execCatalogActionRaw(id string) ([]byte, error) {
	a, ok := opcatalog.Get(id)
	if !ok {
		return nil, fmt.Errorf("unknown action: %s", id)
	}
	tok, _, err := session.Create(1, "telegram-bot", "127.0.0.1")
	if err != nil {
		return nil, err
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
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if resp.StatusCode >= 400 {
		return raw, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

func catalogResultKeyboard(id string) map[string]any {
	sec := "overview"
	if s, ok := opcatalogGetSection(id); ok {
		sec = s
	}
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("📂 "+sec, "m:ops:"+sec, ""), btn("🧰 Tools", "m:tools", "")},
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func formatCatalogHTML(id string, raw []byte, asRaw bool) string {
	lang := catalogLang()
	if asRaw {
		html := format.RawJSONHTML(raw, lang == "ru")
		// allow switch back to pretty view
		html += `<tg-button-row><tg-button type="callback_data" style="link" data="m:op:` + id + `">📋</tg-button></tg-button-row>`
		return html
	}
	r := format.API(id, raw, lang)
	html := r.HTML
	html += `<tg-button-row><tg-button type="callback_data" style="link" data="m:opraw:` + id + `">📄 JSON</tg-button></tg-button-row>`
	return html
}

func opcatalogGetSection(id string) (string, bool) {
	a, ok := opcatalog.Get(id)
	if !ok {
		return "", false
	}
	return a.Section, true
}

// replyCatalog prefers editRich; on failure sends a new rich message (keeps tables).
func replyCatalog(token string, chat int64, msgID int, html string, kb map[string]any) {
	if msgID > 0 {
		if err := editRich(token, chat, msgID, html, kb); err == nil {
			return
		}
	}
	sendHTML(token, chat, html, kb)
}

func toolsDay2Groups() []opcatalog.Group {
	// PLAN-TG-MENU: Tools = day-2 only. Status/Users/Fleet are top-level hubs.
	skip := map[string]bool{"home": true, "users": true, "fleet": true, "overview": true, "vpn": true, "nodes": true}
	var out []opcatalog.Group
	for _, g := range opcatalog.Groups() {
		if skip[g.ID] {
			continue
		}
		out = append(out, g)
	}
	return out
}

func formatToolsFromGroups() string {
	// PLAN-TG-MENU: Tools is day-2 only — no Status/Users/Fleet clones.
	lang := catalogLang()
	nl := "\n"
	var b strings.Builder
	if lang == "ru" {
		b.WriteString("🧰 <b>Инструменты</b>" + nl)
		b.WriteString("<i>DNS, бэкап, edge, медиа, probes — без дублей главных хабов</i>" + nl)
	} else {
		b.WriteString("🧰 <b>Tools</b>" + nl)
		b.WriteString("<i>DNS, backup, edge, media, probes — no top-hub duplicates</i>" + nl)
	}
	b.WriteString(`<tg-button-row align="left">`)
	skip := map[string]bool{"home": true, "users": true, "fleet": true}
	groups := opcatalog.Groups()
	shown := 0
	for _, g := range groups {
		if skip[g.ID] {
			continue
		}
		lab := g.LabelEN
		if lang == "ru" {
			lab = g.LabelRU
		}
		if shown > 0 && shown%4 == 0 {
			b.WriteString(`</tg-button-row><tg-button-row align="left">`)
		}
		b.WriteString(`<tg-button type="callback_data" data="m:cat:` + g.ID + `">` + lab + `</tg-button>`)
		shown++
	}
	b.WriteString(`</tg-button-row>`)
	gitLab, upd, logsLab := "📦 Git / Projects", "🔄 Updates", "📋 Logs"
	if lang == "ru" {
		gitLab, upd, logsLab = "📦 Git / Проекты", "🔄 Обновления", "📋 Логи"
	}
	b.WriteString(`<tg-button-row align="left"><tg-button type="callback_data" style="primary" data="m:git">` + gitLab + `</tg-button></tg-button-row>`)
	b.WriteString(`<tg-button-row align="left"><tg-button type="callback_data" data="m:versions">` + upd + `</tg-button>` +
		`<tg-button type="callback_data" style="primary" data="m:logs">` + logsLab + `</tg-button></tg-button-row>`)
	return b.String()
}


func handleCatGroup(token string, chat int64, msgID int, groupID string) {
	ru := getLang() != "en"
	back := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
	switch groupID {
	case "home", "overview":
		// Status is a top-level hub — do not clone under Tools (PLAN-TG-MENU).
		reply(token, chat, msgID, formatStatusPretty(), statusKeyboard())
	case "users", "vpn":
		reply(token, chat, msgID, formatUsersListHTML(), usersListKeyboard())
	case "fleet", "nodes":
		if groupID == "nodes" {
			reply(token, chat, msgID, nodesHubHTML()+"\n<i>"+T("nodes_hint")+"</i>", nodesKeyboard())
		} else {
			reply(token, chat, msgID, fleetHubHTML(), fleetKeyboard())
		}
	case "edge", "routers":
		reply(token, chat, msgID, routersHubHTML(), routersKeyboard())
	case "sites":
		reply(token, chat, msgID, formatSitesHTML(), sitesListKeyboard())
	case "media", "nvr":
		reply(token, chat, msgID, nvrHubHTML(), nvrKeyboard())
	case "git":
		handleGitCB(token, chat, msgID, "m:git")
		return
	case "data":
		title := "📦 <b>Data</b>"
		if ru {
			title = "📦 <b>Данные</b>"
		}
		body := title + "\n" +
			`<tg-button-row align="left">` +
			`<tg-button type="callback_data" data="m:ops:dns">DNS</tg-button>` +
			`<tg-button type="callback_data" data="m:ops:backup">Backup</tg-button>` +
			`<tg-button type="callback_data" data="m:ops:git">Git / Projects</tg-button>` +
			`</tg-button-row>`
		reply(token, chat, msgID, body, back)
	case "adv", "probes", "updates":
		if groupID == "updates" {
			handleUpdatesCB(token, chat, msgID, "m:versions")
			return
		}
		out := runND("probe")
		r := format.API("probes", []byte(out), catalogLang())
		reply(token, chat, msgID, r.HTML, back)
	default:
		reply(token, chat, msgID, catalogSectionTitle(groupID), catalogSectionKeyboard(groupID))
	}
}
