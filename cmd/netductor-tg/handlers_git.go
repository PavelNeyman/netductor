package main

import (
	"fmt"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func handleGitCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:git" && !strings.HasPrefix(data, "m:git:") {
		return false
	}
	ru := getLang() != "en"
	if data == "m:git" {
		list, err := gitstore.List()
		if err != nil {
			reply(token, chat, msgID, esc(err.Error()), map[string]any{"inline_keyboard": [][]map[string]any{{btn(T("main_menu"), "m:menu", "primary")}}})
			return true
		}
		nl := string([]byte{10})
		var b strings.Builder
		if ru {
			b.WriteString("📦 <b>Git</b>" + nl)
		} else {
			b.WriteString("📦 <b>Git</b>" + nl)
		}
		if len(list) == 0 {
			if ru {
				b.WriteString("<i>Репозиториев нет. CLI: <code>netductor git init name</code></i>")
			} else {
				b.WriteString("<i>No repos. CLI: <code>netductor git init name</code></i>")
			}
			reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{{btn(T("main_menu"), "m:menu", "primary")}}})
			return true
		}
		if ru {
			b.WriteString("<i># открыть репозиторий</i>" + nl)
		} else {
			b.WriteString("<i># open repo</i>" + nl)
		}
		b.WriteString("<table bordered striped compact>" + nl + "<tr><th>#</th><th>repo</th></tr>" + nl)
		for i, n := range list {
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td></tr>"+nl, i+1, esc(n)))
		}
		b.WriteString("</table>" + nl)
		const per = 5
		for i, n := range list {
			if i%per == 0 {
				if i > 0 {
					b.WriteString(`</tg-button-row>` + nl)
				}
				b.WriteString(`<tg-button-row align="left">`)
			}
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:git:repo:%s">%d</tg-button>`, n, i+1))
		}
		b.WriteString(`</tg-button-row>` + nl)
		reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{{btn(T("main_menu"), "m:menu", "primary")}}})
		return true
	}
	rest := strings.TrimPrefix(data, "m:git:")
	if strings.HasPrefix(rest, "repo:") {
		name := strings.TrimPrefix(rest, "repo:")
		nl := string([]byte{10})
		var b strings.Builder
		b.WriteString("📦 <b>" + esc(name) + "</b>" + nl)
		if ru {
			b.WriteString("<i>📜 log · 🔍 HEAD · ▶️ pipeline · ⚙️ workflow · 📄 artifacts · 🗑 delete</i>" + nl)
		} else {
			b.WriteString("<i>📜 log · 🔍 HEAD · ▶️ pipeline · ⚙️ workflow · 📄 artifacts · 🗑 delete</i>" + nl)
		}
		b.WriteString(`<tg-button-row align="left">`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:git:log:` + name + `">📜</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:git:show:` + name + `">🔍</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:git:pipe:` + name + `">▶️</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:git:wf:` + name + `">⚙️</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:git:art:` + name + `">📄</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="danger" data="m:git:del:` + name + `">🗑</tg-button>`)
		b.WriteString(`</tg-button-row>` + nl)
		reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{{btn("« Git", "m:git", "primary"), btn(T("main_menu"), "m:menu", "")}}})
		return true
	}
	if strings.HasPrefix(rest, "log:") {
		name := strings.TrimPrefix(rest, "log:")
		out, err := gitstore.Log(name, 25)
		body := "<pre>" + esc(truncateRunes(out, 3500)) + "</pre>"
		if err != nil {
			body = "⚠️ " + esc(err.Error()) + "\n" + body
		}
		reply(token, chat, msgID, body, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:git:repo:"+name, "")}}})
		return true
	}
	if strings.HasPrefix(rest, "show:") {
		name := strings.TrimPrefix(rest, "show:")
		out, err := gitstore.Show(name, "HEAD")
		body := "<pre>" + esc(truncateRunes(out, 3500)) + "</pre>"
		if err != nil {
			body = "⚠️ " + esc(err.Error()) + "\n" + body
		}
		reply(token, chat, msgID, body, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:git:repo:"+name, "")}}})
		return true
	}
	if strings.HasPrefix(rest, "del:") {
		name := strings.TrimPrefix(rest, "del:")
		err := gitstore.Delete(name)
		msg := "✅ deleted " + name
		if err != nil {
			msg = "⚠️ " + err.Error()
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{{btn("« Git", "m:git", "primary")}}})
		return true
	}
	if strings.HasPrefix(rest, "pipe:") {
		name := strings.TrimPrefix(rest, "pipe:")
		_ = gitstore.EnsureSamplePipeline()
		pipes, _ := gitstore.ListPipelines()
		nl := string([]byte{10})
		var b strings.Builder
		b.WriteString("Pipeline" + nl + "<table bordered striped compact>" + nl + "<tr><th>#</th><th>name</th></tr>" + nl)
		for i, p := range pipes {
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td></tr>"+nl, i+1, esc(p)))
		}
		b.WriteString("</table>" + nl)
		b.WriteString(`<tg-button-row align="left">`)
		for i, p := range pipes {
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:git:run:%s:%s">%d</tg-button>`, name, p, i+1))
		}
		b.WriteString(`</tg-button-row>` + nl)
		reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:git:repo:"+name, "primary")}}})
		return true
	}
	if strings.HasPrefix(rest, "run:") {
		rest2 := strings.TrimPrefix(rest, "run:")
		// name:pipeline
		i := strings.Index(rest2, ":")
		if i < 0 {
			return true
		}
		name, pipe := rest2[:i], rest2[i+1:]
		out, err := gitstore.RunPipeline(name, pipe)
		body := "<pre>" + esc(truncateRunes(out, 3500)) + "</pre>"
		if err != nil {
			body = "⚠️ " + esc(err.Error()) + "\n" + body
		} else {
			body = "✅\n" + body
		}
		reply(token, chat, msgID, body, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:git:repo:"+name, "")}}})
		return true
	}
	if strings.HasPrefix(rest, "wf:") {
		name := strings.TrimPrefix(rest, "wf:")
		out, err := gitstore.RunWorkflow(name, "")
		body := "<pre>" + esc(out) + "</pre>"
		if err != nil {
			body = "❌ " + esc(err.Error()) + "\n" + body
		}
		reply(token, chat, msgID, body, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:git:repo:"+name, "")}}})
		return true
	}
	if strings.HasPrefix(rest, "art:") {
		name := strings.TrimPrefix(rest, "art:")
		list, err := gitstore.ListArtifacts(name)
		body := "(empty)"
		if err != nil {
			body = esc(err.Error())
		} else if len(list) > 0 {
			body = "<pre>" + esc(strings.Join(list, "\n")) + "</pre>"
		}
		reply(token, chat, msgID, body, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:git:repo:"+name, "")}}})
		return true
	}
	return true
}
