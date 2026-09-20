package main

import (
	"encoding/base64"
	"os"
	"strings"
)

func formatRelayHelp() string {
	nl := string([]byte{10})
	if getLang() != "en" {
		return "Управление промежуточным VPS в РФ." + nl + nl + "1) <b>Export</b> / <b>One-liner</b>" + nl + "2) На RU выполнить команду" + nl + "3) Мобильным — secondary / Access links"
	}
	return "RU intermediate VPS." + nl + nl + "1) Export / One-liner" + nl + "2) Run on RU" + nl + "3) Mobile uses secondary Access links"
}

func formatRelayOneline() string {
	_ = runND("secondary", "export", "-o", "/tmp/nd-relay-bundle.json", "--sni", "ya.ru")
	b, err := os.ReadFile("/tmp/nd-relay-bundle.json")
	if err != nil {
		return "❌ export failed: " + esc(err.Error())
	}
	enc := base64.StdEncoding.EncodeToString(b)
	cmd := "wget -qO /usr/local/bin/netductor https://github.com/PavelNeyman/netductor/releases/download/v0.8.5/netductor-linux-amd64 && chmod 755 /usr/local/bin/netductor && echo " + enc + " | base64 -d > /root/bundle.json && netductor secondary join /root/bundle.json"
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
	return "🔐 <b>" + esc(T("ssh_hosts")) + "</b>" + nl + "<i>" + T("ssh_hosts_hint") + "</i>" + nl + nl + "<pre>" + esc(out) + "</pre>"
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
