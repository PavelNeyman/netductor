package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func formatRelayListHTML() string {
	_ = runND("secondary", "status")
	ex := strings.TrimSpace(runND("secondary", "exit"))
	nl := string([]byte{10})
	body := formatNodesListHTML()
	return "📡 <b>Nodes / secondary</b>" + nl + "RU exit: <code>" + esc(ex) + "</code>" + nl + nl + body
}





// nodeCard is the single view-model for TG node screens (core, secondary, edge).
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
	// Node actions in body; navigation under message (nodeCardKeyboard)
	id := c.ID
	b.WriteString(nl + nl)
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="primary" data="m:nd:m:` + id + `">` + esc(T("node_metrics")) + `</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" data="m:nd:j:` + id + `">` + esc(T("node_journal")) + `</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" data="m:nd:s:` + id + `">` + esc(T("node_restart_sb")) + `</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" data="m:nd:u:` + id + `">` + esc(T("node_upgrade")) + `</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="danger" data="m:nd:r:` + id + `">` + esc(T("node_reboot")) + `</tg-button>`)
	b.WriteString(`</tg-button-row>`)
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
	isRelay := c.Role == "secondary" || strings.HasPrefix(id, "secondary-")
	if isRelay {
		detail := runND("secondary", "device", id)
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
	isRelay := role == "secondary" || strings.HasPrefix(id, "secondary-")
	if !isRelay {
		return runND("nodes", "local-cmd", cmd)
	}
	out := runND("secondary", "cmd", id, cmd)
	low := strings.ToLower(out)
	if strings.Contains(low, "does not exist") || strings.Contains(low, "not found") || strings.Contains(low, "exit status") {
		return runND("nodes", "local-cmd", cmd)
	}
	return out
}

func formatJournalHTML(id string) string {
	nl := string([]byte{10})
	role := nodeRole(id)
	var out string
	if role == "secondary" || strings.HasPrefix(id, "secondary-") {
		out = runND("secondary", "cmd", id, "metrics") // soft; journal via secondary agent later
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
	if role == "secondary" || strings.HasPrefix(id, "secondary-") {
		return runND("secondary", "cmd", id, "restart:sing-box")
	}
	b, err := exec.Command("systemctl", "restart", "sing-box").CombinedOutput()
	if err != nil {
		return string(b) + "\n" + err.Error()
	}
	return "ok\n" + string(b)
}


