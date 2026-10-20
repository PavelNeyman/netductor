package main

import (
	"fmt"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func handleGitShip(token string, chat int64, msgID int, rest string) bool {
	if strings.HasPrefix(rest, "ship:") {
		shipSub := strings.TrimPrefix(rest, "ship:")
		nl := string([]byte{10})
		if shipSub == "jobs" {
			jobs, _ := gitstore.ListShipJobs()
			var b strings.Builder
			b.WriteString("🚀 <b>" + T("ship_jobs_title") + "</b>" + nl)
			if len(jobs) == 0 {
				b.WriteString("<i>(empty)</i>" + nl)
			}
			b.WriteString("<table bordered striped compact>" + nl + "<tr><th>#</th><th>project</th><th>state</th></tr>" + nl)
			n := 0
			for _, j := range jobs {
				if n >= 15 {
					break
				}
				n++
				b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td></tr>"+nl, n, esc(j.Project), esc(j.State)))
			}
			b.WriteString("</table>" + nl)
			b.WriteString(`<tg-button-row align="left">`)
			for i, j := range jobs {
				if i >= 12 {
					break
				}
				b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" data="m:git:ship:job:%s">%d</tg-button>`, j.ID, i+1))
			}
			if len(jobs) > 0 {
				b.WriteString(`</tg-button-row>` + nl)
			}
			kb := map[string]any{"inline_keyboard": [][]map[string]any{
				{btn("🔄", "m:git:ship:jobs", ""), btn(T("projects_back"), "m:git:proj:list", "primary")},
				{btn(T("main_menu"), "m:menu", "")},
			}}
			reply(token, chat, msgID, b.String(), kb)
			return true
		}
		if strings.HasPrefix(shipSub, "job:") {
			id := strings.TrimPrefix(shipSub, "job:")
			j, err := gitstore.GetShipJob(id)
			if err != nil {
				reply(token, chat, msgID, "❌ "+esc(err.Error()), navKeyboard("m:git:ship:jobs", "Jobs"))
				return true
			}
			var b strings.Builder
			b.WriteString("🚀 <b>" + esc(j.ID) + "</b>" + nl)
			b.WriteString("project <code>" + esc(j.Project) + "</code> state <code>" + esc(j.State) + "</code>" + nl)
			b.WriteString(fmt.Sprintf("cursor=%d slot=%v tag=%s"+nl, j.Cursor, j.VpsSlot, esc(j.Tag)))
			for i, s := range j.Steps {
				b.WriteString(fmt.Sprintf("%d. %s — %s", i, esc(s.Name), esc(s.Status)))
				if s.DurationSec > 0 {
					b.WriteString(fmt.Sprintf(" (%ds)", s.DurationSec))
				}
				b.WriteString(nl)
			}
			if j.Error != "" {
				b.WriteString("❌ " + esc(j.Error) + nl)
			}
			kb := map[string]any{"inline_keyboard": [][]map[string]any{
				{btn("⏸", "m:git:ship:pause:"+id, ""), btn("▶️", "m:git:ship:resume:"+id, "primary")},
				{btn("📜", "m:git:ship:log:"+id, ""), btn("🗑", "m:git:ship:rm:"+id, "danger")},
				{btn(T("jobs_back"), "m:git:ship:jobs", ""), btn(T("main_menu"), "m:menu", "")},
			}}
			reply(token, chat, msgID, b.String(), kb)
			return true
		}
		if strings.HasPrefix(shipSub, "pause:") {
			id := strings.TrimPrefix(shipSub, "pause:")
			err := gitstore.RequestShipPause(id)
			msg := "✅ pause requested"
			if err != nil {
				msg = "❌ " + err.Error()
			}
			reply(token, chat, msgID, msg, navKeyboard("m:git:ship:jobs", "Jobs"))
			return true
		}
		if strings.HasPrefix(shipSub, "resume:") {
			id := strings.TrimPrefix(shipSub, "resume:")
			job, out, err := gitstore.ResumeShip(id)
			msg := "<pre>" + esc(truncateRunes(out, 2500)) + "</pre>"
			if job != nil {
				msg = "state <code>" + esc(job.State) + "</code>\n" + msg
			}
			if err != nil {
				msg = "❌ " + esc(err.Error()) + "\n" + msg
			}
			reply(token, chat, msgID, msg, navKeyboard("m:git:ship:jobs", "Jobs"))
			return true
		}
		if strings.HasPrefix(shipSub, "log:") {
			id := strings.TrimPrefix(shipSub, "log:")
			s, err := gitstore.ReadShipLog(id, 3500)
			msg := "<pre>" + esc(s) + "</pre>"
			if err != nil {
				msg = "❌ " + esc(err.Error())
			}
			reply(token, chat, msgID, msg, navKeyboard("m:git:ship:job:"+id, "Job"))
			return true
		}
		if strings.HasPrefix(shipSub, "rm:") {
			id := strings.TrimPrefix(shipSub, "rm:")
			err := gitstore.RemoveShipJob(id)
			msg := "✅ removed " + id
			if err != nil {
				msg = "❌ " + err.Error()
			}
			reply(token, chat, msgID, msg, navKeyboard("m:git:ship:jobs", "Jobs"))
			return true
		}
	}
	return false
}
