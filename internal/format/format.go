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
	case "health":
		return formatHealth(v, ru)
	case "bot":
		return formatBot(v, ru)
	case "status":
		return formatStatus(v, ru)
	case "metrics":
		return formatMetrics(v, ru)
	case "metrics-hist":
		return formatMetricsHist(v, ru)
	case "latest":
		return formatMetrics(v, ru)
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


func formatHealth(v any, ru bool) Result {
	m, _ := asMap(v)
	title := "💚 Health"
	if ru {
		title = "💚 Health"
	}
	if m == nil {
		return Result{HTML: "<b>" + title + "</b><br><code>" + esc(fmt.Sprint(v)) + "</code>", Text: "health"}
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b><br>")
	ok := m["ok"] == true
	if ok {
		b.WriteString("✅ <b>ok</b><br>")
	} else {
		b.WriteString("❌ <b>not ok</b><br>")
	}
	for _, k := range []string{"service", "version", "time"} {
		if val, ok := m[k]; ok {
			b.WriteString("• <b>" + esc(k) + "</b>: <code>" + esc(fmt.Sprint(val)) + "</code><br>")
		}
	}
	return Result{HTML: b.String(), Text: "health ok"}
}

func formatBot(v any, ru bool) Result {
	m, _ := asMap(v)
	title := "🤖 Bot"
	var b strings.Builder
	b.WriteString("<b>" + title + "</b><br>")
	if m == nil {
		b.WriteString("<code>" + esc(fmt.Sprint(v)) + "</code>")
		return Result{HTML: b.String(), Text: "bot"}
	}
	if m["ok"] == true {
		b.WriteString("✅ ok<br>")
	}
	if s, ok := m["bot"].(string); ok {
		emoji := "ℹ️"
		if s == "active" {
			emoji = "✅"
		} else if s == "inactive" {
			emoji = "❌"
		}
		b.WriteString(emoji + " status: <b>" + esc(s) + "</b><br>")
	}
	return Result{HTML: b.String(), Text: "bot"}
}

func formatMetrics(v any, ru bool) Result {
	m, _ := asMap(v)
	title := "📊 Metrics"
	if ru {
		title = "📊 Метрики"
	}
	if m == nil {
		return formatSmart("metrics", v, ru)
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b><br>")
	if h, ok := m["hostname"].(string); ok {
		b.WriteString("🖥 <code>" + esc(h) + "</code><br>")
	}
	if c, ok := m["cpu_pct"]; ok {
		b.WriteString("CPU: <b>" + esc(fmt.Sprintf("%.1f", asFloat(c))) + "%</b><br>")
	}
	if la, ok := m["loadavg"].(map[string]any); ok {
		b.WriteString(fmt.Sprintf("Load: <code>%.2f</code> / <code>%.2f</code> / <code>%.2f</code><br>",
			asFloat(la["1"]), asFloat(la["5"]), asFloat(la["15"])))
	}
	if mem, ok := m["mem"].(map[string]any); ok {
		used := asFloat(mem["used"])
		total := asFloat(mem["total"])
		if total > 0 {
			b.WriteString(fmt.Sprintf("RAM: <b>%.1f</b> / %.1f GiB<br>", used/(1<<30), total/(1<<30)))
		}
	}
	if disk, ok := m["disk"].(map[string]any); ok {
		b.WriteString(fmt.Sprintf("Disk: <b>%v%%</b> used<br>", disk["use_pct"]))
	}
	if sv, ok := m["services"].(map[string]any); ok {
		b.WriteString("<br><b>Services</b><br><table bordered striped compact><tr><th>unit</th><th>state</th></tr>")
		keys := make([]string, 0, len(sv))
		for k := range sv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			st := fmt.Sprint(sv[k])
			em := "❓"
			if st == "active" {
				em = "✅"
			} else if st == "inactive" || st == "failed" {
				em = "❌"
			}
			b.WriteString("<tr><td>" + esc(k) + "</td><td>" + em + " " + esc(st) + "</td></tr>")
		}
		b.WriteString("</table>")
	}
	return Result{HTML: b.String(), Text: "metrics"}
}

func formatMetricsHist(v any, ru bool) Result {
	m, _ := asMap(v)
	title := "📈 Metrics history"
	if ru {
		title = "📈 История метрик"
	}
	var list []any
	if m != nil {
		list, _ = m["points"].([]any)
	}
	if list == nil {
		if a, ok := v.([]any); ok {
			list = a
		}
	}
	var b strings.Builder
	b.WriteString("<b>" + title + "</b> · " + fmt.Sprintf("%d", len(list)) + "<br>")
	b.WriteString("<table bordered striped compact><tr><th>ts</th><th>cpu%</th><th>load1</th><th>mem%</th></tr>")
	// show last 15 points
	start := 0
	if len(list) > 15 {
		start = len(list) - 15
	}
	for _, item := range list[start:] {
		im, _ := asMap(item)
		if im == nil {
			continue
		}
		ts := fmt.Sprint(im["ts"])
		cpu := fmt.Sprintf("%.1f", asFloat(im["cpu_pct"]))
		load := "—"
		if la, ok := im["loadavg"].(map[string]any); ok {
			load = fmt.Sprintf("%.2f", asFloat(la["1"]))
		}
		mempct := "—"
		if mem, ok := im["mem"].(map[string]any); ok {
			total := asFloat(mem["total"])
			used := asFloat(mem["used"])
			if total > 0 {
				mempct = fmt.Sprintf("%.0f", used*100/total)
			}
		}
		b.WriteString("<tr><td><code>" + esc(ts) + "</code></td><td>" + esc(cpu) + "</td><td>" + esc(load) + "</td><td>" + esc(mempct) + "</td></tr>")
	}
	b.WriteString("</table>")
	return Result{HTML: b.String(), Text: "metrics history"}
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
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
