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
	HTML string
	Text string
}

// API formats known action payloads; always returns readable HTML (never raw-only).
func API(actionID string, raw []byte, lang string) Result {
	ru := lang == "ru"
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		s := string(raw)
		return Result{
			HTML: "<b>📦 " + html.EscapeString(actionID) + "</b><br><pre>" + html.EscapeString(truncate(s, 3500)) + "</pre>",
			Text: s,
		}
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
	case "health", "bot":
		return formatFlatMap(actionID, emojiFor(actionID), v, ru)
	case "status":
		return formatStatus(v, ru)
	case "metrics", "metrics-hist", "latest":
		return formatFlatMap(actionID, "📊", v, ru)
	case "nodes", "nodes-self", "sites", "edge-devices", "edge-pending", "git-repos",
		"sessions", "ssh-hosts", "addons", "sni", "sni-presets", "probes", "probes-cfg",
		"audit", "mtls-certs", "secondary", "secondary-links", "edge-metrics",
		"nvr-config", "nvr-storage", "nvr-events", "nvr-segments", "git-pipelines",
		"reg-status", "backup-peer", "sec-export", "probes-uptime":
		return formatSmart(actionID, v, ru)
	default:
		return formatSmart(actionID, v, ru)
	}
}

func emojiFor(id string) string {
	switch id {
	case "health":
		return "💚"
	case "bot":
		return "🤖"
	case "doctor":
		return "🩺"
	default:
		return "📦"
	}
}

func formatDoctor(v any, ru bool) Result {
	m, _ := asMap(v)
	if m == nil {
		return formatSmart("doctor", v, ru)
	}
	okN, failN, warnN := 0, 0, 0
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
	b.WriteString(fmt.Sprintf("🖥 <code>%s</code> · %s<br>", esc(host), esc(role)))
	b.WriteString(fmt.Sprintf("✅ <b>%d</b> · ⚠️ <b>%d</b> · ❌ <b>%d</b><br><br>", okN, warnN, failN))

	var fails, warns []string
	if checks, ok := m["checks"].([]any); ok {
		for _, c := range checks {
			cm, _ := asMap(c)
			if cm == nil {
				continue
			}
			id, _ := cm["id"].(string)
			st, _ := cm["status"].(string)
			line := esc(id)
			if d, _ := cm["detail"].(string); d != "" {
				line += " — " + esc(d)
			}
			switch st {
			case "fail":
				fails = append(fails, "❌ "+line)
			case "warn":
				warns = append(warns, "⚠️ "+line)
			}
		}
	}
	if len(fails)+len(warns) > 0 {
		b.WriteString("<table bordered striped compact><tr><th>issue</th></tr>")
		for _, line := range fails {
			b.WriteString("<tr><td>" + line + "</td></tr>")
		}
		for _, line := range warns {
			b.WriteString("<tr><td>" + line + "</td></tr>")
		}
		b.WriteString("</table>")
	} else if ru {
		b.WriteString("✨ Всё в порядке.")
	} else {
		b.WriteString("✨ All clear.")
	}
	return Result{HTML: b.String(), Text: fmt.Sprintf("doctor ok=%d warn=%d fail=%d", okN, warnN, failN)}
}

func formatDomain(v any, ru bool) Result {
	m, _ := asMap(v)
	dom, _ := m["domain"].(map[string]any)
	if dom == nil {
		return formatSmart("domain", v, ru)
	}
	title := "🌐 Domain"
	if ru {
		title = "🌐 Домен"
	}
	return Result{HTML: "<b>" + title + "</b><br>" + mapTable(dom, 40), Text: "domain"}
}

func formatVPNUsers(v any, ru bool) Result {
	list := extractList(v, "users", "Users")
	title := "👥 VPN users"
	if ru {
		title = "👥 VPN пользователи"
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b> · " + fmt.Sprintf("%d", len(list)) + "<br>")
	b.WriteString("<table bordered striped compact><tr><th>name</th><th>status</th></tr>")
	for _, item := range list {
		im, _ := asMap(item)
		if im == nil {
			b.WriteString("<tr><td colspan=\"2\">" + esc(fmt.Sprint(item)) + "</td></tr>")
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
		b.WriteString("<tr><td>" + esc(name) + "</td><td>" + esc(st) + "</td></tr>")
	}
	b.WriteString("</table>")
	return Result{HTML: b.String(), Text: fmt.Sprintf("%d users", len(list))}
}

func formatNVRCameras(v any, ru bool) Result {
	list := extractList(v, "cameras", "Cameras")
	title := "🎥 Cameras"
	if ru {
		title = "🎥 Камеры"
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b> · " + fmt.Sprintf("%d", len(list)) + "<br>")
	b.WriteString("<table bordered striped compact><tr><th>id</th><th>name</th><th>IP</th><th>rec</th></tr>")
	for _, item := range list {
		im, _ := asMap(item)
		if im == nil {
			continue
		}
		rec := "—"
		if r, ok := im["record"].(bool); ok {
			if r {
				rec = "⏺"
			} else {
				rec = "⏸"
			}
		}
		b.WriteString("<tr><td><code>" + esc(firstStr(im, "id", "ID")) + "</code></td><td>" +
			esc(firstStr(im, "name", "Name")) + "</td><td>" +
			esc(firstStr(im, "lan_ip", "LANIP", "ip")) + "</td><td>" + rec + "</td></tr>")
	}
	b.WriteString("</table>")
	return Result{HTML: b.String(), Text: fmt.Sprintf("%d cameras", len(list))}
}

func formatStatus(v any, ru bool) Result {
	m, _ := asMap(v)
	if m == nil {
		return formatSmart("status", v, ru)
	}
	title := "📡 Status"
	if ru {
		title = "📡 Статус"
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b><br>")
	// top scalars
	flat := map[string]any{}
	for k, val := range m {
		switch val.(type) {
		case map[string]any, []any:
			continue
		default:
			flat[k] = val
		}
	}
	if len(flat) > 0 {
		b.WriteString(mapTable(flat, 20))
	}
	// nested summary lines
	for _, k := range []string{"metrics", "probes", "mismatch", "secondary_mismatch"} {
		if sub, ok := m[k]; ok {
			b.WriteString("<br><b>" + esc(k) + "</b><br>")
			b.WriteString(valueHTML(sub, 0))
		}
	}
	return Result{HTML: b.String(), Text: "status"}
}

func formatFlatMap(action, emoji string, v any, ru bool) Result {
	m, _ := asMap(v)
	if m == nil {
		return formatSmart(action, v, ru)
	}
	return Result{HTML: "<b>" + emoji + " " + esc(action) + "</b><br>" + mapTable(m, 40), Text: action}
}

// formatSmart: arrays → table of items; objects → key/value (nested summarized).
func formatSmart(action string, v any, ru bool) Result {
	title := emojiFor(action) + " " + action
	var b strings.Builder
	b.WriteString("<b>" + esc(title) + "</b><br>")
	b.WriteString(valueHTML(v, 0))
	return Result{HTML: b.String(), Text: action}
}

func valueHTML(v any, depth int) string {
	if depth > 3 {
		return "<code>…</code>"
	}
	switch t := v.(type) {
	case map[string]any:
		// if looks like list wrapper
		for _, k := range []string{"users", "cameras", "nodes", "sites", "devices", "pending", "repos", "sessions", "hosts", "items", "checks", "events", "segments"} {
			if a, ok := t[k].([]any); ok {
				return "<i>" + esc(k) + "</i> · " + fmt.Sprintf("%d", len(a)) + "<br>" + listTable(a, 25)
			}
		}
		return mapTable(t, 30)
	case []any:
		return listTable(t, 25)
	default:
		return "<code>" + esc(truncate(fmt.Sprint(t), 200)) + "</code>"
	}
}

func mapTable(m map[string]any, maxRows int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("<table bordered striped compact><tr><th>key</th><th>value</th></tr>")
	n := 0
	for _, k := range keys {
		b.WriteString("<tr><td>" + esc(k) + "</td><td>" + cellValue(m[k]) + "</td></tr>")
		n++
		if n >= maxRows {
			b.WriteString("<tr><td colspan=\"2\">…</td></tr>")
			break
		}
	}
	b.WriteString("</table>")
	return b.String()
}

func listTable(list []any, maxRows int) string {
	var b strings.Builder
	b.WriteString("<table bordered striped compact><tr><th>#</th><th>item</th></tr>")
	for i, item := range list {
		if i >= maxRows {
			b.WriteString("<tr><td colspan=\"2\">… +" + fmt.Sprintf("%d", len(list)-maxRows) + "</td></tr>")
			break
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td></tr>", i+1, cellValue(item)))
	}
	b.WriteString("</table>")
	return b.String()
}

func cellValue(v any) string {
	switch t := v.(type) {
	case nil:
		return "—"
	case bool:
		if t {
			return "✅"
		}
		return "🚫"
	case float64, int, int64, json.Number:
		return "<code>" + esc(fmt.Sprint(t)) + "</code>"
	case string:
		return "<code>" + esc(truncate(t, 120)) + "</code>"
	case map[string]any:
		// one-line summary of object
		if name := firstStr(t, "name", "id", "host", "path", "repo", "unit"); name != "" {
			return esc(name)
		}
		return "<code>{" + fmt.Sprintf("%d keys", len(t)) + "}</code>"
	case []any:
		return "<code>[" + fmt.Sprintf("%d", len(t)) + "]</code>"
	default:
		return "<code>" + esc(truncate(fmt.Sprint(t), 100)) + "</code>"
	}
}

// RawJSONHTML wraps raw JSON in expandable details (Bot API 10.x).
func RawJSONHTML(raw []byte, ru bool) string {
	sum := "📄 JSON"
	if ru {
		sum = "📄 Сырой JSON"
	}
	body := esc(truncate(prettyJSON(raw), 8000))
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

func extractList(v any, keys ...string) []any {
	switch t := v.(type) {
	case []any:
		return t
	case map[string]any:
		for _, k := range keys {
			if a, ok := t[k].([]any); ok {
				return a
			}
		}
	}
	return nil
}

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
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

func esc(s string) string { return html.EscapeString(s) }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
