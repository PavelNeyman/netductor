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
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("<table bordered striped compact>" + nl + "<tr><th>#</th><th>line</th></tr>" + nl)
	i := 0
	for _, line := range strings.Split(out, nl) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		i++
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td></tr>"+nl, i, esc(line)))
	}
	b.WriteString("</table>")
	if i == 0 {
		return T("sites_empty")
	}
	return b.String()
}

func formatSitesRSCHTML() string {
	nl := string([]byte{10})
	out := strings.TrimSpace(runND("sites", "rsc"))
	ru := getLang() != "en"
	title := "📜 <b>MikroTik RSC</b>"
	if ru {
		title = "📜 <b>Скрипт MikroTik (RSC)</b>"
	}
	var b strings.Builder
	b.WriteString(title + nl)
	b.WriteString("<table bordered striped compact>" + nl)
	n := 0
	if out != "" {
		n = len(strings.Split(out, nl))
	}
	if ru {
		b.WriteString("<tr><th>поле</th><th>значение</th></tr>" + nl)
		b.WriteString("<tr><td>формат</td><td>RouterOS script (.rsc)</td></tr>" + nl)
		b.WriteString("<tr><td>куда</td><td>терминал Winbox / <code>/import</code></td></tr>" + nl)
		b.WriteString("<tr><td>источник</td><td>шаблон площадки netductor</td></tr>" + nl)
		b.WriteString(fmt.Sprintf("<tr><td>строк</td><td>%d</td></tr>"+nl, n))
	} else {
		b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
		b.WriteString("<tr><td>format</td><td>RouterOS script (.rsc)</td></tr>" + nl)
		b.WriteString("<tr><td>apply</td><td>Winbox terminal / <code>/import</code></td></tr>" + nl)
		b.WriteString("<tr><td>source</td><td>netductor site template</td></tr>" + nl)
		b.WriteString(fmt.Sprintf("<tr><td>lines</td><td>%d</td></tr>"+nl, n))
	}
	b.WriteString("</table>" + nl + nl)
	if out == "" {
		if ru {
			b.WriteString("<i>Нет данных: создайте площадку и шаблон.</i>")
		} else {
			b.WriteString("<i>No data: create a site and template first.</i>")
		}
		return b.String()
	}
	if ru {
		b.WriteString("<b>Текст скрипта</b> <i>(команды RouterOS на английском — так требует ROS)</i>" + nl)
	} else {
		b.WriteString("<b>Script body</b> <i>(RouterOS commands are English by design)</i>" + nl)
	}
	b.WriteString("<pre>" + esc(out) + "</pre>")
	return b.String()
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
