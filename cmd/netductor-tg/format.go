package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/PavelNeyman/netductor/internal/addons"
)

func yn(ok bool, good, bad string) string {
	if ok {
		return good
	}
	return bad
}

func formatLampacHTML() string {
	st := addons.CollectLampac()
	ru := getLang() != "en"
	var b strings.Builder
	b.WriteString("📺 <b>Lampac</b>\n\n")
	if !st.Installed {
		if ru {
			b.WriteString("⚪ Не установлен\n")
			b.WriteString("<i>Установка: <code>netductor install lampac</code></i>")
		} else {
			b.WriteString("⚪ Not installed\n")
			b.WriteString("<i>Install: <code>netductor install lampac</code></i>")
		}
		return b.String()
	}
	runIcon := yn(st.Running, "🟢", "🔴")
	healthIcon := yn(st.Healthy, "✅", "⚠️")
	if ru {
		b.WriteString(fmt.Sprintf("%s <b>Контейнер:</b> %s\n", runIcon, yn(st.Running, "запущен", "остановлен")))
		b.WriteString(fmt.Sprintf("%s <b>Health:</b> %s\n", healthIcon, yn(st.Healthy, "healthy", "unhealthy")))
	} else {
		b.WriteString(fmt.Sprintf("%s <b>Container:</b> %s\n", runIcon, yn(st.Running, "running", "stopped")))
		b.WriteString(fmt.Sprintf("%s <b>Health:</b> %s\n", healthIcon, yn(st.Healthy, "healthy", "unhealthy")))
	}
	if st.Image != "" {
		b.WriteString(fmt.Sprintf("📦 <b>Image:</b> <code>%s</code>\n", esc(st.Image)))
	}
	b.WriteString(fmt.Sprintf("🔌 <b>Bind:</b> <code>%s</code>\n", esc(st.Bind)))
	if st.VersionHash != "" {
		b.WriteString(fmt.Sprintf("🏷 <b>Version:</b> <code>%s</code>\n", esc(st.VersionHash)))
	}
	if st.CPU != "" || st.Mem != "" {
		b.WriteString(fmt.Sprintf("📊 <b>CPU / RAM:</b> %s · %s\n", esc(st.CPU), esc(st.Mem)))
	}
	b.WriteString(fmt.Sprintf("🏓 <b>Ping:</b> %s\n", yn(st.PingOK, "✅", "—")))
	b.WriteString(fmt.Sprintf("🧭 <b>Chromium:</b> %s\n", yn(st.ChromiumOK, "✅", "—")))
	b.WriteByte('\n')
	if ru {
		b.WriteString("🔗 UI: <code>" + esc(st.UIURL) + "</code>\n")
		b.WriteString("🛠 Admin: <code>" + esc(st.AdminURL) + "</code>\n")
		b.WriteString("\n<i>Только localhost — доступ через VPN/SSH-туннель</i>")
	} else {
		b.WriteString("🔗 UI: <code>" + esc(st.UIURL) + "</code>\n")
		b.WriteString("🛠 Admin: <code>" + esc(st.AdminURL) + "</code>\n")
		b.WriteString("\n<i>Loopback only — reach via VPN/SSH tunnel</i>")
	}
	return b.String()
}

func formatAddonsHTML() string {
	ru := getLang() != "en"
	var b strings.Builder
	if ru {
		b.WriteString("🧩 <b>Аддоны</b>\n\nВыберите компонент:")
	} else {
		b.WriteString("🧩 <b>Addons</b>\n\nPick a component:")
	}
	return b.String()
}

// formatEdgeListHTML turns tab-separated edge list into cards.
func formatEdgeListHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if getLang() != "en" {
			return "📡 <i>Нет устройств</i>"
		}
		return "📡 <i>No devices</i>"
	}
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("📡 <b>Роутеры / edge</b>\n\n")
	} else {
		b.WriteString("📡 <b>Routers / edge</b>\n\n")
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		// id status online board host ip=… last=
		id := parts[0]
		rest := ""
		if len(parts) > 1 {
			rest = strings.Join(parts[1:], " · ")
		}
		icon := "⚪"
		low := strings.ToLower(line)
		if strings.Contains(low, "online") {
			icon = "🟢"
		} else if strings.Contains(low, "offline") {
			icon = "🔴"
		} else if strings.Contains(low, "pending") {
			icon = "⏳"
		}
		b.WriteString(fmt.Sprintf("%s <code>%s</code>\n   %s\n", icon, esc(id), esc(rest)))
	}
	return b.String()
}

func formatPendingHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if getLang() != "en" {
			return "⏳ <b>Ожидают approve</b>\n\n<i>Список пуст</i>"
		}
		return "⏳ <b>Pending</b>\n\n<i>Empty</i>"
	}
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("⏳ <b>Ожидают approve</b>\n\n")
	} else {
		b.WriteString("⏳ <b>Pending enroll</b>\n\n")
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		id := parts[0]
		extra := ""
		if len(parts) > 1 {
			extra = strings.Join(parts[1:], " · ")
		}
		b.WriteString(fmt.Sprintf("• <code>%s</code>", esc(id)))
		if extra != "" {
			b.WriteString(" — " + esc(extra))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func formatTemplatesHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("📋 <b>Шаблоны</b>\n\n")
	} else {
		b.WriteString("📋 <b>Templates</b>\n\n")
	}
	if raw == "" {
		b.WriteString("<i>—</i>")
		return b.String()
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		b.WriteString("• <code>" + esc(line) + "</code>\n")
	}
	return b.String()
}

func formatSessionHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	// expect token line(s) from vpn session
	var token string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if len(line) >= 32 && isHexish(line) {
			token = line
			break
		}
	}
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("🔑 <b>Session token</b>\n\n")
	} else {
		b.WriteString("🔑 <b>Session token</b>\n\n")
	}
	if token != "" {
		b.WriteString("<code>" + esc(token) + "</code>\n\n")
		if getLang() != "en" {
			b.WriteString("<i>Для Admin SPA / API (Authorization: Bearer …)</i>")
		} else {
			b.WriteString("<i>For Admin SPA / API (Authorization: Bearer …)</i>")
		}
	} else {
		b.WriteString("<pre>" + esc(raw) + "</pre>")
	}
	return b.String()
}

func isHexish(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func formatDoctorHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	var b strings.Builder
	b.WriteString("🩺 <b>Doctor</b>\n\n")
	ok, fail, warn := 0, 0, 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "OK"):
			ok++
			b.WriteString("✅ " + esc(strings.TrimSpace(strings.TrimPrefix(line, "OK"))) + "\n")
		case strings.HasPrefix(line, "FAIL"):
			fail++
			b.WriteString("❌ " + esc(strings.TrimSpace(strings.TrimPrefix(line, "FAIL"))) + "\n")
		case strings.HasPrefix(line, "WARN"):
			warn++
			b.WriteString("⚠️ " + esc(strings.TrimSpace(strings.TrimPrefix(line, "WARN"))) + "\n")
		case strings.HasPrefix(line, "INFO"):
			b.WriteString("ℹ️ " + esc(strings.TrimSpace(strings.TrimPrefix(line, "INFO"))) + "\n")
		case strings.HasPrefix(line, "Summary"):
			b.WriteString("\n<b>" + esc(line) + "</b>\n")
		default:
			if strings.HasPrefix(line, "Netductor doctor") {
				b.WriteString("<b>" + esc(line) + "</b>\n")
			}
		}
	}
	_ = ok
	_ = fail
	_ = warn
	return b.String()
}



func formatVPNListPretty(raw string) string {
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>#</th><th>user</th><th>state</th></tr>" + nl)
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 1 {
			parts = strings.Fields(line)
		}
		name := parts[0]
		if name == "relay-uplink" {
			continue
		}
		en := ""
		if len(parts) > 1 {
			en = parts[1]
		}
		n++
		icon := "🟢"
		if en == "off" {
			icon = "🔴"
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s %s</td></tr>"+nl, n, esc(name), icon, esc(en)))
	}
	if n == 0 {
		b.WriteString("<tr><td>—</td><td>—</td><td>—</td></tr>" + nl)
	}
	b.WriteString("</table>")
	return b.String()
}

func formatStatusPretty() string {
	nl := string([]byte{10})
	ru := getLang() != "en"
	host, _ := os.Hostname()
	title := "📊 <b>Core status</b>"
	if ru {
		title = "📊 <b>Статус core</b>"
	}
	var b strings.Builder
	b.WriteString(title + nl)
	b.WriteString("<code>" + esc(host) + "</code>" + nl + nl)
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>service</th><th>state</th></tr>" + nl)
	for _, u := range []string{"sing-box", "blocky", "netductor-api", "netductor-telegram-bot"} {
		out, _ := exec.Command("systemctl", "is-active", u).CombinedOutput()
		st := strings.TrimSpace(string(out))
		icon := "🔴"
		if st == "active" {
			icon = "🟢"
		}
		b.WriteString("<tr><td>" + esc(u) + "</td><td>" + icon + " " + esc(st) + "</td></tr>" + nl)
	}
	b.WriteString("</table>" + nl + nl)
	nodesTitle := "🗂 <b>Nodes</b>"
	if ru {
		nodesTitle = "🗂 <b>Ноды</b>"
	}
	b.WriteString(nodesTitle + nl + formatNodesListHTML() + nl + nl)
	b.WriteString("👥 <b>VPN</b>" + nl)
	b.WriteString(formatVPNListPretty(runVPN("list")))
	return b.String()
}

func formatVPNLinkHTML(name string) (caption string, vless, hy2, sub string) {
	vless = strings.TrimSpace(runVPN("link", name, "vless"))
	hy2 = strings.TrimSpace(runVPN("link", name, "hy2"))
	sub = strings.TrimSpace(runVPN("link", name))
	if sub != "" {
		first := strings.TrimSpace(strings.Split(sub, "\n")[0])
		if strings.HasPrefix(first, "vless://") {
			vless = first
		}
	}
	if strings.Contains(vless, "not found") || strings.Contains(vless, "exit status") {
		vless = ""
	}
	if strings.Contains(hy2, "not found") || strings.Contains(hy2, "exit status") {
		hy2 = ""
	}
	if strings.Contains(sub, "not found") || strings.Contains(sub, "exit status") {
		sub = ""
	}
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("🔗 <b>VPN · " + esc(name) + "</b>" + nl + nl)
	if vless != "" {
		b.WriteString("<b>VLESS · primary</b>" + nl + "<code>" + esc(vless) + "</code>" + nl + nl)
		coreL := strings.TrimSpace(runVPN("link", name, "core"))
		if coreL != "" && coreL != vless && !strings.Contains(coreL, "not found") {
			b.WriteString("<b>VLESS · core (домашний, быстрее)</b>" + nl + "<code>" + esc(coreL) + "</code>" + nl + nl)
		}
	}
	if hy2 != "" {
		b.WriteString("<b>HY2</b>" + nl + "<code>" + esc(hy2) + "</code>" + nl)
	}
	if vless == "" && hy2 == "" {
		b.WriteString("❌ no links")
	}
	return b.String(), vless, hy2, sub
}

func formatRelayListHTML() string {
	_ = runND("relay", "status")
	ex := strings.TrimSpace(runND("relay", "exit"))
	nl := string([]byte{10})
	body := formatNodesListHTML()
	return "📡 <b>Nodes / relay</b>" + nl + "RU exit: <code>" + esc(ex) + "</code>" + nl + nl + body
}





// nodeCard is the single view-model for TG node screens (core, relay, edge).
type nodeCard struct {
	ID       string
	Host     string
	Role     string
	IP       string
	Status   string
	CPU      float64
	MemUsed  int64
	MemTotal int64
	Load     float64
	Lines    []string // optional extra lines (already HTML)
}

func nodeRole(id string) string {
	for _, l := range strings.Split(runND("nodes", "list"), "\n") {
		if !strings.Contains(l, "id="+id) {
			continue
		}
		for _, p := range strings.Split(l, "\t") {
			if strings.HasPrefix(p, "role=") {
				return strings.TrimPrefix(p, "role=")
			}
		}
	}
	return ""
}

func parseNodeFields(line string) map[string]string {
	m := map[string]string{}
	for _, p := range strings.Split(line, "\t") {
		if i := strings.IndexByte(p, '='); i > 0 {
			m[p[:i]] = p[i+1:]
		}
	}
	return m
}

func sampleHostMetrics() (cpu float64, memUsed, memTotal int64, load1 float64) {
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		fmt.Sscanf(string(b), "%f", &load1)
	}
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, avail int64
		for _, line := range strings.Split(string(b), string([]byte{10})) {
			if strings.HasPrefix(line, "MemTotal:") {
				fmt.Sscanf(line, "MemTotal: %d", &total)
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				fmt.Sscanf(line, "MemAvailable: %d", &avail)
			}
		}
		memTotal = total / 1024
		if total > 0 {
			memUsed = (total - avail) / 1024
		}
	}
	cpu = load1 * 50
	if cpu > 100 {
		cpu = 100
	}
	return
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func table2(rows [][2]string) string {
	w := 0
	for _, r := range rows {
		if len(r[0]) > w {
			w = len(r[0])
		}
	}
	if w < 8 {
		w = 8
	}
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(padRight(r[0], w))
		b.WriteString("  ")
		b.WriteString(r[1])
		b.WriteByte(10)
	}
	return b.String()
}

// formatNodeCardHTML — one template for all roles (pre tables).
func formatNodeCardHTML(c nodeCard) string {
	nl := string([]byte{10})
	host := c.Host
	if host == "" {
		host = c.ID
	}
	role := c.Role
	if role == "" {
		role = "—"
	}
	st := c.Status
	if st == "" {
		st = "—"
	}
	idShort := c.ID
	if len(idShort) > 18 {
		idShort = idShort[:8] + "…" + idShort[len(idShort)-6:]
	}
	var b strings.Builder
	b.WriteString("🖥 <b>" + esc(host) + "</b>" + nl + nl)
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
	b.WriteString("<tr><td>role</td><td>" + esc(role) + "</td></tr>" + nl)
	b.WriteString("<tr><td>id</td><td>" + esc(idShort) + "</td></tr>" + nl)
	if c.IP != "" {
		b.WriteString("<tr><td>ip</td><td>" + esc(c.IP) + "</td></tr>" + nl)
	}
	b.WriteString("<tr><td>status</td><td>" + esc(st) + "</td></tr>" + nl)
	b.WriteString("</table>" + nl + nl)
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>metric</th><th>value</th></tr>" + nl)
	b.WriteString(fmt.Sprintf("<tr><td>cpu</td><td>%.1f%%</td></tr>"+nl, c.CPU))
	b.WriteString(fmt.Sprintf("<tr><td>mem</td><td>%d / %d MB</td></tr>"+nl, c.MemUsed, c.MemTotal))
	b.WriteString(fmt.Sprintf("<tr><td>load</td><td>%.2f</td></tr>"+nl, c.Load))
	b.WriteString("</table>")
	if len(c.Lines) > 0 {
		b.WriteString(nl + nl)
		for _, line := range c.Lines {
			b.WriteString(line + nl)
		}
	}
	return strings.TrimRight(b.String(), nl)
}

func loadNodeCard(id string) nodeCard {
	c := nodeCard{ID: id}
	for _, l := range strings.Split(runND("nodes", "list"), "\n") {
		if !strings.Contains(l, "id="+id) {
			continue
		}
		f := parseNodeFields(l)
		c.Host = f["host"]
		c.Role = f["role"]
		c.IP = f["ip"]
		c.Status = f["status"]
		break
	}
	if c.Role == "" {
		c.Role = nodeRole(id)
	}
	isRelay := c.Role == "relay" || strings.HasPrefix(id, "relay-")
	if isRelay {
		detail := runND("relay", "device", id)
		for _, dl := range strings.Split(detail, "\n") {
			dl = strings.TrimSpace(dl)
			if strings.HasPrefix(dl, "status:") {
				c.Status = strings.TrimSpace(strings.TrimPrefix(dl, "status:"))
			}
			var sb string
			_, _ = fmt.Sscanf(dl, "sb= %s cpu= %f mem= %d / %d load= %f", &sb, &c.CPU, &c.MemUsed, &c.MemTotal, &c.Load)
			if strings.HasPrefix(dl, "pending:") {
				c.Lines = append(c.Lines, "⏳ <code>"+esc(dl)+"</code>")
			}
			if strings.HasPrefix(dl, "last_cmd:") {
				c.Lines = append(c.Lines, "✅ <code>"+esc(dl)+"</code>")
			}
		}
		if c.Status == "" {
			c.Status = "online"
		}
		return c
	}
	c.CPU, c.MemUsed, c.MemTotal, c.Load = sampleHostMetrics()
	if c.Status == "" {
		c.Status = "online"
	}
	var sb strings.Builder
	sb.WriteString("<table bordered striped>" + string([]byte{10}))
	sb.WriteString("<tr><th>service</th><th>state</th></tr>" + string([]byte{10}))
	for _, u := range []string{"sing-box", "netductor-api", "netductor-telegram-bot"} {
		out, _ := exec.Command("systemctl", "is-active", u).CombinedOutput()
		st := strings.TrimSpace(string(out))
		if st == "active" {
			st = "🟢 active"
		} else {
			st = "🔴 " + st
		}
		sb.WriteString("<tr><td>" + esc(u) + "</td><td>" + esc(st) + "</td></tr>" + string([]byte{10}))
	}
	sb.WriteString("</table>")
	c.Lines = append(c.Lines, sb.String())
	if lb, err := os.ReadFile("/var/lib/netductor/core-upgrade.log"); err == nil {
		s := strings.TrimSpace(string(lb))
		if s != "" && s != "started" {
			if len(s) > 350 {
				s = s[len(s)-350:]
			}
			c.Lines = append(c.Lines, "📋 <b>last upgrade</b>", "<pre>"+esc(s)+"</pre>")
		}
	}
	return c
}

func formatNodeDetailHTML(id string) string {
	return formatNodeCardHTML(loadNodeCard(id))
}

func formatCmdQueuedHTML(kind, nodeID, raw string) string {
	nl := string([]byte{10})
	c := loadNodeCard(nodeID)
	host := c.Host
	if host == "" {
		host = nodeID
	}
	title := "🔄 <b>Upgrade</b>"
	if kind == "reboot" {
		title = "♻️ <b>Reboot</b>"
	}
	var b strings.Builder
	b.WriteString(title + nl + nl)
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
	b.WriteString("<tr><td>node</td><td>" + esc(host) + "</td></tr>" + nl)
	b.WriteString("<tr><td>role</td><td>" + esc(c.Role) + "</td></tr>" + nl)
	b.WriteString("<tr><td>status</td><td>queued</td></tr>" + nl)
	b.WriteString("</table>" + nl)
	b.WriteString("<i>" + T("cmd_wait_hint") + "</i>")
	return b.String()
}

func enqueueNodeCmd(id, cmd string) string {
	role := nodeRole(id)
	isRelay := role == "relay" || strings.HasPrefix(id, "relay-")
	if !isRelay {
		return runND("nodes", "local-cmd", cmd)
	}
	out := runND("relay", "cmd", id, cmd)
	low := strings.ToLower(out)
	if strings.Contains(low, "does not exist") || strings.Contains(low, "not found") || strings.Contains(low, "exit status") {
		return runND("nodes", "local-cmd", cmd)
	}
	return out
}

func ensureQRFile(path, payload string) string {
	if payload == "" {
		return ""
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	// Always regenerate — cached PNG may still encode core IP while link is relay.
	if err := qrcode.WriteFile(payload, qrcode.Medium, 512, path); err != nil {
		return ""
	}
	_ = os.Chmod(path, 0o600)
	return path
}

func deliverVPNLink(token string, chat int64, msgID int, name string) {
	// Replace the message that contained the user button (list / card).
	showVPNQR(token, chat, msgID, name, "vless", msgID > 0)
}

func vpnQRCaption(name, mode, vless, hy2 string) string {
	ru := getLang() != "en"
	nl := string([]byte{10})
	var b strings.Builder
	if mode == "hy2" {
		b.WriteString("📱 <b>Hysteria2</b> · " + esc(name) + nl + nl)
		if hy2 != "" {
			b.WriteString("<code>" + esc(hy2) + "</code>")
		}
	} else {
		b.WriteString("📱 <b>VLESS Reality</b> · " + esc(name) + nl + nl)
		if vless != "" {
			b.WriteString("<code>" + esc(vless) + "</code>")
		}
		if hy2 != "" {
			if ru {
				b.WriteString(nl + nl + "<i>HY2 — кнопка «HY2 QR» выше</i>")
			} else {
				b.WriteString(nl + nl + "<i>HY2 — use «HY2 QR» button</i>")
			}
		}
	}
	return b.String()
}

func showVPNQR(token string, chat int64, msgID int, name, mode string, edit bool) {
	_, vless, hy2, sub := formatVPNLinkHTML(name)
	if mode != "hy2" {
		mode = "vless"
	}
	kb := userCardKeyboardMode(name, mode)
	dir := filepath.Join("/etc/netductor/clients", name)
	var path, payload string
	if mode == "hy2" {
		payload = hy2
		path = ensureQRFile(filepath.Join(dir, "qr-hy2.png"), hy2)
	} else {
		payload = vless
		if payload == "" {
			payload = sub
		}
		path = ensureQRFile(filepath.Join(dir, "qr-vless.png"), vless)
		if path == "" {
			path = ensureQRFile(filepath.Join(dir, "qr.png"), vless)
		}
	}
	cap := vpnQRCaption(name, mode, vless, hy2)
	if payload == "" && path == "" {
		sendHTML(token, chat, cap, kb)
		return
	}
	if path == "" {
		sendHTML(token, chat, cap, kb)
		return
	}
	if edit && msgID > 0 {
		if err := editPhotoFile(token, chat, msgID, path, cap, kb); err != nil {
			// Text list message cannot become a photo — delete and put QR in its place.
			_ = deleteMessage(token, chat, msgID)
			if err2 := sendPhotoFile(token, chat, path, cap, kb); err2 != nil {
				sendHTML(token, chat, cap+string([]byte{10})+"⚠️ <code>"+esc(err2.Error())+"</code>", kb)
			}
		}
		return
	}
	if err := sendPhotoFile(token, chat, path, cap, kb); err != nil {
		sendHTML(token, chat, cap+string([]byte{10})+"⚠️ <code>"+esc(err.Error())+"</code>", kb)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func formatRelayHelp() string {
	nl := string([]byte{10})
	if getLang() != "en" {
		return "Управление промежуточным VPS в РФ." + nl + nl + "1) <b>Export</b> / <b>One-liner</b>" + nl + "2) На RU выполнить команду" + nl + "3) Мобильным — relay links"
	}
	return "RU intermediate VPS." + nl + nl + "1) Export / One-liner" + nl + "2) Run on RU" + nl + "3) Mobile uses relay links"
}

func formatRelayOneline() string {
	_ = runND("relay", "export", "-o", "/tmp/nd-relay-bundle.json", "--sni", "ya.ru")
	b, err := os.ReadFile("/tmp/nd-relay-bundle.json")
	if err != nil {
		return "❌ export failed: " + esc(err.Error())
	}
	enc := base64.StdEncoding.EncodeToString(b)
	cmd := "wget -qO /usr/local/bin/netductor https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64 && chmod 755 /usr/local/bin/netductor && echo " + enc + " | base64 -d > /root/bundle.json && netductor relay join /root/bundle.json"
	nl := string([]byte{10})
	if getLang() != "en" {
		return "🧾 <b>Одна команда на RU VPS</b>" + nl + nl + "<code>" + esc(cmd) + "</code>"
	}
	return "🧾 <b>One command on RU VPS</b>" + nl + nl + "<code>" + esc(cmd) + "</code>"
}



func formatJournalHTML(id string) string {
	nl := string([]byte{10})
	role := nodeRole(id)
	var out string
	if role == "relay" || strings.HasPrefix(id, "relay-") {
		out = runND("relay", "cmd", id, "metrics") // soft; journal on relay via agent later
		out = "relay journal: use Metrics / last_cmd for now\n" + out
	} else {
		b, _ := exec.Command("journalctl", "-u", "sing-box", "-n", "40", "--no-pager", "-o", "short-iso").CombinedOutput()
		out = string(b)
	}
	if len(out) > 3500 {
		out = out[len(out)-3500:]
	}
	return "📋 <b>Journal</b>" + nl + "<pre>" + esc(out) + "</pre>"
}

func restartSingBox(id string) string {
	role := nodeRole(id)
	if role == "relay" || strings.HasPrefix(id, "relay-") {
		return runND("relay", "cmd", id, "restart:sing-box")
	}
	b, err := exec.Command("systemctl", "restart", "sing-box").CombinedOutput()
	if err != nil {
		return string(b) + "\n" + err.Error()
	}
	return "ok\n" + string(b)
}
