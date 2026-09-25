// Package format turns session-API JSON into human-readable rich HTML (TG Bot API 10.x)
// and plain text for other UIs.
package format

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"
)

// Result of formatting.
type Result struct {
	HTML string // TG rich_message html (tables, details, emoji)
	Text string // plain / TUI
}

// API formats known action payloads; unknown → compact summary + raw details-friendly body.
func API(actionID string, raw []byte, lang string) Result {
	ru := lang == "ru"
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		s := string(raw)
		return Result{HTML: "<pre>" + html.EscapeString(truncate(s, 3500)) + "</pre>", Text: s}
	}
	switch actionID {
	case "doctor":
		return formatDoctor(v, ru)
	case "domain":
		return formatDomain(v, ru)
	case "vpn-users":
		return formatVPNUsers(v, ru)
	case "nvr-cameras":
		return formatNVRCameras(v, ru)
	case "status", "metrics", "bot", "health":
		return formatKeyValues(actionID, v, ru)
	case "nodes", "sites", "edge-devices", "edge-pending", "git-repos", "sessions", "ssh-hosts":
		return formatListy(actionID, v, ru)
	default:
		return formatGeneric(actionID, v, ru)
	}
}

func formatDoctor(v any, ru bool) Result {
	m, _ := v.(map[string]any)
	if m == nil {
		return formatGeneric("doctor", v, ru)
	}
	okN, failN, warnN := num(m, "summary", "ok"), num(m, "summary", "fail"), num(m, "summary", "warn")
	// also nested summary object
	if s, ok := m["summary"].(map[string]any); ok {
		okN, failN, warnN = asInt(s["ok"]), asInt(s["fail"]), asInt(s["warn"])
	}
	role, _ := m["role"].(string)
	host, _ := m["host"].(string)
	title := "🩺 Doctor"
	if ru {
		title = "🩺 Диагностика"
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b><br>")
	b.WriteString(fmt.Sprintf("🖥 <code>%s</code> · %s<br>", html.EscapeString(host), html.EscapeString(role)))
	b.WriteString(fmt.Sprintf("✅ %d · ⚠️ %d · ❌ %d<br><br>", okN, warnN, failN))
	b.WriteString("<table bordered striped compact><tr><th>check</th><th>status</th></tr>")
	if checks, ok := m["checks"].([]any); ok {
		for _, c := range checks {
			cm, _ := c.(map[string]any)
			if cm == nil {
				continue
			}
			id, _ := cm["id"].(string)
			st, _ := cm["status"].(string)
			emoji := "✅"
			if st == "fail" {
				emoji = "❌"
			} else if st == "warn" {
				emoji = "⚠️"
			}
			// only show non-ok to keep short, or show all if few fails
			if st == "ok" {
				continue
			}
			b.WriteString("<tr><td>" + html.EscapeString(id) + "</td><td>" + emoji + " " + html.EscapeString(st) + "</td></tr>")
		}
	}
	b.WriteString("</table>")
	if failN == 0 && warnN == 0 {
		if ru {
			b.WriteString("<br>✨ Всё в порядке.")
		} else {
			b.WriteString("<br>✨ All clear.")
		}
	}
	text := fmt.Sprintf("doctor %s/%s ok=%d warn=%d fail=%d", role, host, okN, warnN, failN)
	return Result{HTML: b.String(), Text: text}
}

func formatDomain(v any, ru bool) Result {
	m, _ := v.(map[string]any)
	dom, _ := m["domain"].(map[string]any)
	if dom == nil {
		if d, ok := m["domain"].(string); ok {
			return Result{HTML: "🌐 <b>Domain</b><br><code>" + html.EscapeString(d) + "</code>", Text: d}
		}
		return formatGeneric("domain", v, ru)
	}
	title := "🌐 Domain"
	if ru {
		title = "🌐 Домен"
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b><br><table bordered striped compact>")
	b.WriteString("<tr><th>key</th><th>value</th></tr>")
	keys := make([]string, 0, len(dom))
	for k := range dom {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString("<tr><td>" + html.EscapeString(k) + "</td><td><code>" + html.EscapeString(fmt.Sprint(dom[k])) + "</code></td></tr>")
	}
	b.WriteString("</table>")
	return Result{HTML: b.String(), Text: "domain settings"}
}

func formatVPNUsers(v any, ru bool) Result {
	// shape varies: {users:[...]} or array
	var list []any
	switch t := v.(type) {
	case map[string]any:
		if u, ok := t["users"].([]any); ok {
			list = u
		} else if u, ok := t["Users"].([]any); ok {
			list = u
		}
	case []any:
		list = t
	}
	title := "👥 VPN users"
	if ru {
		title = "👥 VPN пользователи"
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b> · " + fmt.Sprintf("%d", len(list)) + "<br>")
	b.WriteString("<table bordered striped compact><tr><th>name</th><th>status</th></tr>")
	for _, item := range list {
		im, _ := item.(map[string]any)
		if im == nil {
			continue
		}
		name := firstStr(im, "name", "Name", "id", "ID")
		st := "—"
		if en, ok := im["enabled"].(bool); ok {
			if en {
				st = "✅ on"
			} else {
				st = "🚫 off"
			}
		} else if s, ok := im["status"].(string); ok {
			st = s
		}
		b.WriteString("<tr><td>" + html.EscapeString(name) + "</td><td>" + html.EscapeString(st) + "</td></tr>")
	}
	b.WriteString("</table>")
	return Result{HTML: b.String(), Text: fmt.Sprintf("%d vpn users", len(list))}
}

func formatNVRCameras(v any, ru bool) Result {
	var list []any
	if m, ok := v.(map[string]any); ok {
		if c, ok := m["cameras"].([]any); ok {
			list = c
		}
	}
	title := "🎥 Cameras"
	if ru {
		title = "🎥 Камеры"
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b> · " + fmt.Sprintf("%d", len(list)) + "<br>")
	b.WriteString("<table bordered striped compact><tr><th>id</th><th>name</th><th>IP</th><th>rec</th></tr>")
	for _, item := range list {
		im, _ := item.(map[string]any)
		if im == nil {
			continue
		}
		id := firstStr(im, "id", "ID")
		name := firstStr(im, "name", "Name")
		ip := firstStr(im, "lan_ip", "LANIP", "ip")
		rec := "—"
		if r, ok := im["record"].(bool); ok {
			if r {
				rec = "⏺"
			} else {
				rec = "⏸"
			}
		}
		b.WriteString("<tr><td><code>" + html.EscapeString(id) + "</code></td><td>" + html.EscapeString(name) +
			"</td><td>" + html.EscapeString(ip) + "</td><td>" + rec + "</td></tr>")
	}
	b.WriteString("</table>")
	return Result{HTML: b.String(), Text: fmt.Sprintf("%d cameras", len(list))}
}

func formatKeyValues(action string, v any, ru bool) Result {
	m, ok := v.(map[string]any)
	if !ok {
		return formatGeneric(action, v, ru)
	}
	emoji := "📊"
	if action == "health" || action == "bot" {
		emoji = "💚"
	}
	var b strings.Builder
	b.WriteString("<b>" + emoji + " " + html.EscapeString(action) + "</b><br>")
	b.WriteString("<table bordered striped compact><tr><th>key</th><th>value</th></tr>")
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	n := 0
	for _, k := range keys {
		if k == "metrics" || k == "probes" || k == "mismatch" {
			continue // too deep
		}
		val := m[k]
		if _, isMap := val.(map[string]any); isMap {
			continue
		}
		if _, isArr := val.([]any); isArr {
			continue
		}
		b.WriteString("<tr><td>" + html.EscapeString(k) + "</td><td><code>" + html.EscapeString(fmt.Sprint(val)) + "</code></td></tr>")
		n++
		if n >= 24 {
			break
		}
	}
	b.WriteString("</table>")
	return Result{HTML: b.String(), Text: action}
}

func formatListy(action string, v any, ru bool) Result {
	var list []any
	switch t := v.(type) {
	case []any:
		list = t
	case map[string]any:
		for _, k := range []string{"nodes", "sites", "devices", "pending", "repos", "sessions", "hosts", "items"} {
			if a, ok := t[k].([]any); ok {
				list = a
				break
			}
		}
		if list == nil {
			return formatKeyValues(action, v, ru)
		}
	}
	title := "📋 " + action
	var b strings.Builder
	b.WriteString("<b>" + html.EscapeString(title) + "</b> · " + fmt.Sprintf("%d", len(list)) + "<br>")
	b.WriteString("<table bordered striped compact><tr><th>#</th><th>item</th></tr>")
	for i, item := range list {
		if i >= 30 {
			b.WriteString("<tr><td colspan=\"2\">…</td></tr>")
			break
		}
		label := fmt.Sprint(item)
		if im, ok := item.(map[string]any); ok {
			label = firstStr(im, "id", "name", "host", "path", "repo")
			if label == "" {
				label = fmt.Sprint(im)
			}
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td></tr>", i+1, html.EscapeString(truncate(label, 80))))
	}
	b.WriteString("</table>")
	return Result{HTML: b.String(), Text: fmt.Sprintf("%s (%d)", action, len(list))}
}

func formatGeneric(action string, v any, ru bool) Result {
	title := "📦 " + action
	if ru {
		title = "📦 " + action
	}
	// shallow summary
	var b strings.Builder
	b.WriteString("<b>" + html.EscapeString(title) + "</b><br>")
	switch t := v.(type) {
	case map[string]any:
		b.WriteString(fmt.Sprintf("keys: <code>%d</code>", len(t)))
	case []any:
		b.WriteString(fmt.Sprintf("items: <code>%d</code>", len(t)))
	default:
		b.WriteString("<code>" + html.EscapeString(truncate(fmt.Sprint(v), 200)) + "</code>")
	}
	return Result{HTML: b.String(), Text: action}
}

// RawJSONHTML wraps raw JSON in expandable details (Bot API 10.x <details>).
func RawJSONHTML(raw []byte, ru bool) string {
	sum := "📄 JSON"
	if ru {
		sum = "📄 Сырой JSON"
	}
	body := html.EscapeString(truncate(prettyJSON(raw), 8000))
	return "<details><summary>" + sum + "</summary><pre language=\"json\">" + body + "</pre></details>"
}

func prettyJSON(raw []byte) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return string(raw)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(b)
}

func firstStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func asInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	}
	return 0
}

func num(m map[string]any, path ...string) int {
	cur := any(m)
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return 0
		}
		cur = mm[p]
	}
	return asInt(cur)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
