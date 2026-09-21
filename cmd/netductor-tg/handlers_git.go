package main

import (
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
		var rows [][]map[string]any
		for _, n := range list {
			rows = append(rows, []map[string]any{btn(n, "m:git:repo:"+n, "")})
		}
		if len(rows) == 0 {
			msg := "Репозиториев нет. CLI: <code>netductor git init name</code>"
			if !ru {
				msg = "No repos. CLI: <code>netductor git init name</code>"
			}
			rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
			reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": rows})
			return true
		}
		rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
		title := "📦 <b>Git</b> — выберите репозиторий"
		if !ru {
			title = "📦 <b>Git</b> — pick a repo"
		}
		reply(token, chat, msgID, title, map[string]any{"inline_keyboard": rows})
		return true
	}
	rest := strings.TrimPrefix(data, "m:git:")
	if strings.HasPrefix(rest, "repo:") {
		name := strings.TrimPrefix(rest, "repo:")
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("📜 Log", "m:git:log:"+name, ""), btn("🔍 HEAD", "m:git:show:"+name, "")},
			{btn("▶️ Pipeline", "m:git:pipe:"+name, ""), btn("⚙️ Workflow", "m:git:wf:"+name, "")},
			{btn("📄 Artifacts", "m:git:art:"+name, ""), btn("🗑 Del", "m:git:del:"+name, "")},
			{btn("«", "m:git", "")},
		}}
		reply(token, chat, msgID, "📦 <b>"+esc(name)+"</b>", kb)
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
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:git", "")}}})
		return true
	}
	if strings.HasPrefix(rest, "pipe:") {
		name := strings.TrimPrefix(rest, "pipe:")
		_ = gitstore.EnsureSamplePipeline()
		pipes, _ := gitstore.ListPipelines()
		var rows [][]map[string]any
		for _, p := range pipes {
			rows = append(rows, []map[string]any{btn(p, "m:git:run:"+name+":"+p, "")})
		}
		rows = append(rows, []map[string]any{btn("«", "m:git:repo:"+name, "")})
		reply(token, chat, msgID, "Pipeline:", map[string]any{"inline_keyboard": rows})
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
