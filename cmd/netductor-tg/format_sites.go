package main

import (
	"fmt"
	"encoding/base64"
	"os"
	"strings"
)


func formatRelayOneline() string {
	_ = runND("secondary", "export", "-o", "/tmp/nd-secondary-bundle.json", "--sni", "ya.ru")
	b, err := os.ReadFile("/tmp/nd-secondary-bundle.json")
	if err != nil {
		return "❌ export failed: " + esc(err.Error())
	}
	enc := base64.StdEncoding.EncodeToString(b)
	cmd := "wget -qO /usr/local/bin/netductor https://github.com/PavelNeyman/netductor/releases/download/v0.8.22/netductor-linux-amd64 && chmod 755 /usr/local/bin/netductor && echo " + enc + " | base64 -d > /root/bundle.json && netductor secondary join /root/bundle.json"
	nl := string([]byte{10})
	if getLang() != "en" {
		return "🧾 <b>Одна команда на RU VPS</b>" + nl + nl + "<code>" + esc(cmd) + "</code>"
	}
	return "🧾 <b>One command on RU VPS</b>" + nl + nl + "<code>" + esc(cmd) + "</code>"
}



func formatSitesHTML() string {
	out := strings.TrimSpace(runND("sites", "list"))
	if out == "" || strings.Contains(out, "unknown") {
		return T("sites_empty")
	}
	return "<pre>" + esc(out) + "</pre>"
}

func formatSitesRSCHTML() string {
	nl := string([]byte{10})
	out := strings.TrimSpace(runND("sites", "rsc"))
	if out == "" {
		out = runND("sites", "list")
	}
	if len(out) > 3500 {
		out = out[:3500] + "…"
	}
	return "📜 <b>MikroTik RSC</b>" + nl + "<pre>" + esc(out) + "</pre>"
}



func formatSSHHostsHTML() string {
	out := strings.TrimSpace(runND("ssh-hosts", "list"))
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("🔐 <b>" + esc(T("ssh_hosts")) + "</b>" + nl)
	ids := []string{}
	for _, line := range strings.Split(out, nl) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "===") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		ids = append(ids, fields[0])
	}
	b.WriteString("<table bordered striped compact>" + nl + "<tr><th>#</th><th>id</th></tr>" + nl)
	for i, id := range ids {
		if i >= 12 {
			break
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td></tr>"+nl, i+1, esc(id)))
	}
	b.WriteString("</table>" + nl)
	b.WriteString(`<tg-button-row align="left">`)
	for i, id := range ids {
		if i >= 12 {
			break
		}
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="danger" data="m:ssh:rm:%s">%d</tg-button>`, id, i+1))
	}
	b.WriteString(`</tg-button-row>` + nl)
	return b.String()
}

func sshHostsKeyboard() map[string]any {
	nl := string([]byte{10})
	out := runND("ssh-hosts", "list")
	var top [][]map[string]any
	n := 0
	for _, line := range strings.Split(out, nl) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "===") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		id := fields[0]
		label := id
		if len(label) > 28 {
			label = label[:28] + "…"
		}
		top = append(top, []map[string]any{btn("🗑 "+label, "m:ssh:rm:"+id, "danger")})
		n++
		if n >= 12 {
			break
		}
	}
	rows := top
	rows = append(rows,
		[]map[string]any{btn(T("ssh_clear_mt"), "m:ssh:clear:mt", "danger"), btn(T("ssh_clear_rel"), "m:ssh:clear:relay", "danger")},
		[]map[string]any{btn(T("ssh_forget"), "m:ssh:forget", "")},
		[]map[string]any{btn(T("back"), "m:cat:nodes", "primary"), btn(T("main_menu"), "m:menu", "")},
	)
	return map[string]any{"inline_keyboard": rows}
}
