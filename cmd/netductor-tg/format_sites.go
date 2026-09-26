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
	ru := getLang() != "en"
	title := "📜 <b>MikroTik RSC</b>"
	hint := "<i>Paste into RouterOS / import. From site template.</i>"
	if ru {
		hint = "<i>Вставить в терминал RouterOS / import. Из шаблона площадки.</i>"
	}
	if out == "" {
		if ru {
			out = "(пусто — нет площадок или шаблона)"
		} else {
			out = "(empty — no sites/template)"
		}
	}
	return title + nl + hint + nl + nl + "<pre>" + esc(out) + "</pre>"
}


func formatSSHHostsHTML() string {
	out := strings.TrimSpace(runND("ssh-hosts", "list"))
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("🔐 <b>" + esc(T("ssh_hosts")) + "</b>" + nl)
	if h := strings.TrimSpace(T("ssh_hosts_hint")); h != "" {
		b.WriteString("<i>" + esc(h) + "</i>" + nl)
	}
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
	// numbered forget per host
	if len(ids) > 0 {
		b.WriteString(`<tg-button-row align="left">`)
		for i, id := range ids {
			if i >= 12 {
				break
			}
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="danger" data="m:ssh:rm:%s">%d</tg-button>`, id, i+1))
		}
		b.WriteString(`</tg-button-row>` + nl)
	}
	// bulk actions in body
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="danger" data="m:ssh:clear:mt">` + esc(T("ssh_clear_mt")) + `</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="danger" data="m:ssh:clear:relay">` + esc(T("ssh_clear_rel")) + `</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:ssh:forget">` + esc(T("ssh_forget")) + `</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	return b.String()
}

func sshHostsKeyboard() map[string]any {
	return navKeyboard("m:cat:nodes", parentNodes())
}
