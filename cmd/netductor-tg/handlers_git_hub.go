package main

import (
	"fmt"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func handleGitHub(token string, chat int64, msgID int, data string) bool {
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
				btn("🚀", "m:git:proj:ship:"+pr.Name, ""),
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
	return false
}
