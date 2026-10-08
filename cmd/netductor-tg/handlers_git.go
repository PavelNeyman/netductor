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
		ru := getLang() != "en"
		list, err := gitstore.List()
		projs, _ := gitstore.LoadProjects()
		nl := string([]byte{10})
		var b strings.Builder
		if ru {
			b.WriteString("📦 <b>Git</b>" + nl)
		} else {
			b.WriteString("📦 <b>Git</b>" + nl)
		}
		if err != nil {
			b.WriteString(esc(err.Error()) + nl)
		}
		// Projects first (GH mirrors) — primary UX
		b.WriteString(nl)
		if ru {
			b.WriteString("<b>Проекты</b> (mirror GitHub)" + nl)
		} else {
			b.WriteString("<b>Projects</b> (GitHub mirror)" + nl)
		}
		if len(projs) == 0 {
			if ru {
				b.WriteString("<i>Пусто — ➕ или CLI: git project migrate-from-gh OWNER --sync</i>" + nl)
			} else {
				b.WriteString("<i>Empty — ➕ Add project</i>" + nl)
			}
		} else {
			for _, pr := range projs {
				b.WriteString("• <code>" + esc(pr.Name) + "</code> " + esc(pr.Upstream) + nl)
			}
		}
		if len(list) > 0 {
			b.WriteString(nl)
			if ru {
				b.WriteString("<b>Bare repos</b>" + nl)
			} else {
				b.WriteString("<b>Bare repos</b>" + nl)
			}
			for i, n := range list {
				b.WriteString(fmt.Sprintf("%d. <code>%s</code>"+nl, i+1, esc(n)))
			}
		}
		rows := [][]map[string]any{}
		addLab := "➕ Add project"
		allLab := "📁 All projects"
		if ru {
			addLab = "➕ Добавить проект"
			allLab = "📁 Все проекты"
		}
		rows = append(rows, []map[string]any{btn(allLab, "m:git:proj:list", "primary"), btn(addLab, "m:git:proj:add", "")})
		for _, pr := range projs {
			if len(rows) > 8 {
				break
			}
			rows = append(rows, []map[string]any{
				btn("📋 "+pr.Name, "m:git:proj:card:"+pr.Name, "primary"),
				btn("⬇️", "m:git:proj:sync:"+pr.Name, ""),
				btn("▶️", "m:git:proj:build:"+pr.Name, ""),
			})
		}
		tokLab := "🔑 GitHub token"
		if ru {
			tokLab = "🔑 GitHub token"
		}
		rows = append(rows, []map[string]any{btn(tokLab, "m:updates:token:set", "")})
		rows = append(rows, []map[string]any{btn("⬅️ "+parentTools(), "m:tools", ""), btn(T("main_menu"), "m:menu", "")})
		reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": rows})
		return true
	}
	rest := strings.TrimPrefix(data, "m:git:")
	
	if strings.HasPrefix(rest, "proj:") {
		sub := strings.TrimPrefix(rest, "proj:")
		if sub == "list" || sub == "" {
			projs, err := gitstore.LoadProjects()
			var b strings.Builder
			nl := string([]byte{10})
			b.WriteString("📁 <b>Projects</b> (GH mirror + workflow/pipeline)" + nl)
			if err != nil {
				b.WriteString(esc(err.Error()))
			} else if len(projs) == 0 {
				b.WriteString("<i>Empty — ➕ or CLI: git project migrate-from-gh OWNER --sync</i>" + nl)
			} else {
				for _, pr := range projs {
					b.WriteString("• <code>" + esc(pr.Name) + "</code> " + esc(pr.Upstream) + nl)
				}
			}
			rows := [][]map[string]any{}
			for _, pr := range projs {
				rows = append(rows, []map[string]any{
					btn("📋 "+pr.Name, "m:git:proj:card:"+pr.Name, "primary"),
				})
				rows = append(rows, []map[string]any{
					btn("⬇️", "m:git:proj:sync:"+pr.Name, ""),
					btn("▶️", "m:git:proj:build:"+pr.Name, ""),
				})
			}
			addLab := "➕ Add project"
			if ru {
				addLab = "➕ Добавить проект"
			}
			rows = append(rows, []map[string]any{btn(addLab, "m:git:proj:add", "primary")})
			rows = append(rows, []map[string]any{btn("⬅️ Git", "m:git", "primary"), btn(T("main_menu"), "m:menu", "")})
			reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": rows})
			return true
		}
		if sub == "add" {
			setState(chat, "wait_git_proj_name", "")
			msg := "Project <b>name</b> (e.g. M4tg_bot):"
			if ru {
				msg = "Имя проекта (напр. M4tg_bot):"
			}
			reply(token, chat, msgID, msg, navKeyboard("m:git:proj:list", "Projects"))
			return true
		}
		if strings.HasPrefix(sub, "card:") {
			name := strings.TrimPrefix(sub, "card:")
			d, err := gitstore.ProjectDetailOf(name)
			if err != nil {
				reply(token, chat, msgID, "❌ "+esc(err.Error()), navKeyboard("m:git:proj:list", "Projects"))
				return true
			}
			nl := string([]byte{10})
			var b strings.Builder
			b.WriteString("📋 <b>" + esc(d.Name) + "</b>" + nl)
			b.WriteString("upstream: <code>" + esc(d.Upstream) + "</code>" + nl)
			host := d.Host
			if host == "" {
				host = "vps"
			}
			b.WriteString("host: <code>" + esc(host) + "</code>")
			if d.BuildOnFetch {
				b.WriteString(" · build_on_fetch")
			}
			b.WriteString(nl)
			if d.Workflow != "" {
				b.WriteString("workflow: <code>" + esc(d.Workflow) + "</code>" + nl)
			}
			if d.Pipeline != "" {
				b.WriteString("pipeline: <code>" + esc(d.Pipeline) + "</code>" + nl)
			}
			if d.Note != "" {
				b.WriteString("<i>" + esc(d.Note) + "</i>" + nl)
			}
			if len(d.Tags) > 0 {
				b.WriteString(nl + "<b>tags</b>" + nl)
				for _, tg := range d.Tags {
					b.WriteString("· <code>" + esc(tg) + "</code>" + nl)
				}
			} else {
				b.WriteString(nl + "<i>no tags in mirror — sync first</i>" + nl)
			}
			if len(d.Artifacts) > 0 {
				b.WriteString(nl + "<b>artifacts</b> (last)" + nl)
				for _, a := range d.Artifacts {
					b.WriteString("· <code>" + esc(a) + "</code>" + nl)
				}
			}
			kb := map[string]any{"inline_keyboard": [][]map[string]any{
				{btn("⬇️ Sync", "m:git:proj:sync:"+name, ""), btn("▶️ Build", "m:git:proj:build:"+name, "primary")},
				{btn("⬅️ Projects", "m:git:proj:list", "primary"), btn(T("main_menu"), "m:menu", "")},
			}}
			reply(token, chat, msgID, b.String(), kb)
			return true
		}
		if strings.HasPrefix(sub, "sync:") {
			name := strings.TrimPrefix(sub, "sync:")
			out, err := gitstore.SyncProject(name)
			msg := "<pre>" + esc(truncateRunes(out, 3000)) + "</pre>"
			if err != nil {
				msg = "⚠️ " + esc(err.Error()) + "\n" + msg
			} else {
				msg = "✅ sync\n"+msg
			}
			reply(token, chat, msgID, msg, navKeyboard("m:git:proj:list", "Projects"))
			return true
		}
		if strings.HasPrefix(sub, "build:") {
			name := strings.TrimPrefix(sub, "build:")
			out, err := gitstore.BuildProject(name, "")
			msg := "<pre>" + esc(truncateRunes(out, 3000)) + "</pre>"
			if err != nil {
				msg = "❌ " + esc(err.Error()) + "\n" + msg
			} else {
				msg = "✅ build\n" + msg
			}
			reply(token, chat, msgID, msg, navKeyboard("m:git:proj:list", "Projects"))
			return true
		}
	}
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
		reply(token, chat, msgID, b.String(), navKeyboard("m:git", "Git"))
		return true
	}
	if strings.HasPrefix(rest, "log:") {
		name := strings.TrimPrefix(rest, "log:")
		out, err := gitstore.Log(name, 25)
		body := "<pre>" + esc(truncateRunes(out, 3500)) + "</pre>"
		if err != nil {
			body = "⚠️ " + esc(err.Error()) + "\n" + body
		}
		reply(token, chat, msgID, body, navKeyboard("m:git", "Git"))
		return true
	}
	if strings.HasPrefix(rest, "show:") {
		name := strings.TrimPrefix(rest, "show:")
		out, err := gitstore.Show(name, "HEAD")
		body := "<pre>" + esc(truncateRunes(out, 3500)) + "</pre>"
		if err != nil {
			body = "⚠️ " + esc(err.Error()) + "\n" + body
		}
		reply(token, chat, msgID, body, navKeyboard("m:git", "Git"))
		return true
	}
	if strings.HasPrefix(rest, "del:") {
		name := strings.TrimPrefix(rest, "del:")
		err := gitstore.Delete(name)
		msg := "✅ deleted " + name
		if err != nil {
			msg = "⚠️ " + err.Error()
		}
		reply(token, chat, msgID, msg, navKeyboard("m:git", "Git"))
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
		reply(token, chat, msgID, b.String(), navKeyboard("m:git", "Git"))
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
		reply(token, chat, msgID, body, navKeyboard("m:git", "Git"))
		return true
	}
	if strings.HasPrefix(rest, "wf:") {
		name := strings.TrimPrefix(rest, "wf:")
		out, err := gitstore.RunWorkflow(name, "")
		body := "<pre>" + esc(out) + "</pre>"
		if err != nil {
			body = "❌ " + esc(err.Error()) + "\n" + body
		}
		reply(token, chat, msgID, body, navKeyboard("m:git", "Git"))
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
		reply(token, chat, msgID, body, navKeyboard("m:git", "Git"))
		return true
	}
	return true
}
