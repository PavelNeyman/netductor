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
	base := formatToolsFromGroups()
	if getLang() != "en" {
		return base + "\n\n🧱 <b>Стек</b>\n" +
			`<tg-button-row><tg-button type="callback_data" style="primary" data="m:stack">Stack status</tg-button></tg-button-row>`
	}
	return base + "\n\n🧱 <b>Stack</b>\n" +
		`<tg-button-row><tg-button type="callback_data" style="primary" data="m:stack">Stack status</tg-button></tg-button-row>`
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

func formatToolsFromGroups() string {
	lang := catalogLang()
	var b strings.Builder
	if lang == "ru" {
		b.WriteString("🛠 <b>Tools</b>\n<i>Группы Day-2 = Web / TUI (opcatalog.Groups).</i>\n")
	} else {
		b.WriteString("🛠 <b>Tools</b>\n<i>Day-2 groups = Web / TUI (opcatalog.Groups).</i>\n")
	}
	b.WriteString(`<tg-button-row align="left">`)
	groups := opcatalog.Groups()
	for i, g := range groups {
		lab := g.LabelEN
		if lang == "ru" {
			lab = g.LabelRU
		}
		b.WriteString(`<tg-button type="callback_data" data="m:cat:` + g.ID + `">` + lab + `</tg-button>`)
		if (i+1)%4 == 0 && i+1 < len(groups) {
			b.WriteString(`</tg-button-row><tg-button-row align="left">`)
		}
	}
	b.WriteString(`</tg-button-row>`)
	upd := "🔄 Updates"
	if lang == "ru" {
		upd = "🔄 Обновления"
	}
	top := "📁 Topics"
	if lang == "ru" {
		top = "📁 Топики"
	}
	b.WriteString(`<tg-button-row align="left"><tg-button type="callback_data" data="m:versions">` + upd + `</tg-button>` +
		`<tg-button type="callback_data" data="m:topics">` + top + `</tg-button></tg-button-row>`)
	return b.String()
}



// handleCatGroup maps opcatalog.Groups() IDs to real TG screens (not unknown → main menu).
func handleCatGroup(token string, chat int64, msgID int, groupID string) {
	ru := getLang() != "en"
	back := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
	switch groupID {
	case "home", "overview":
		body := formatStatusPretty()
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("🧱 Stack", "m:stack", "primary"), btn("🔄 Updates", "m:versions", "")},
			{btn("📡 Digest", "m:digest", ""), btn("🛟 DR", "m:dr", "")},
			{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
		}}
		if ru {
			kb = map[string]any{"inline_keyboard": [][]map[string]any{
				{btn("🧱 Стек", "m:stack", "primary"), btn("🔄 Обновления", "m:versions", "")},
				{btn("📡 Digest", "m:digest", ""), btn("🛟 DR", "m:dr", "")},
				{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
			}}
		}
		reply(token, chat, msgID, body, kb)
	case "users", "vpn":
		reply(token, chat, msgID, formatUsersListHTML(), usersListKeyboard())
	case "fleet", "nodes":
		// fleet hub or nodes list — both valid
		if groupID == "nodes" {
			reply(token, chat, msgID, formatNodesListHTML(), nodesKeyboard())
		} else {
			reply(token, chat, msgID, fleetHubHTML(), fleetKeyboard())
		}
	case "edge", "routers":
		reply(token, chat, msgID, routersHubHTML(), routersKeyboard())
	case "sites":
		reply(token, chat, msgID, formatSitesHTML(), sitesListKeyboard())
	case "media", "nvr":
		reply(token, chat, msgID, nvrHubHTML(), nvrKeyboard())
	case "data":
		title := "📦 <b>Data</b>"
		if ru {
			title = "📦 <b>Данные</b>"
		}
		body := title + "\n" +
			`<tg-button-row align="left">` +
			`<tg-button type="callback_data" data="m:ops:dns">DNS</tg-button>` +
			`<tg-button type="callback_data" data="m:ops:backup">Backup</tg-button>` +
			`<tg-button type="callback_data" data="m:ops:git">Git</tg-button>` +
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

