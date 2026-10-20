package main

import (
	"fmt"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func handleGitProj(token string, chat int64, msgID int, rest string, ru bool) bool {
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
					btn("🚀", "m:git:proj:ship:"+pr.Name, ""),
				})
			}
			rows = append(rows, []map[string]any{btn(T("git_add_proj"), "m:git:proj:add", "primary"), btn(T("git_migrate"), "m:git:proj:migrate", "")})
			rows = append(rows, []map[string]any{btn(T("git_sync_all"), "m:git:proj:sync-all", ""), btn(T("git_build_all"), "m:git:proj:build-all", "")})
			rows = append(rows, []map[string]any{btn(T("git_vps_queue"), "m:git:proj:vps-queue", ""), btn(T("ship_jobs"), "m:git:ship:jobs", "primary")})
			rows = append(rows, []map[string]any{btn(T("build_host_btn"), "m:git:build-host", "")})
			rows = append(rows, []map[string]any{btn(T("git_mirror_all"), "m:git:proj:mirror-all", "")})
			rows = append(rows, []map[string]any{btn(T("git_back"), "m:git", "primary"), btn(T("main_menu"), "m:menu", "")})
			reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": rows})
			return true
		}
		if sub == "mirror-all" {
			list, err := gitstore.LoadProjects()
			var b strings.Builder
			nl := string([]byte{10})
			if err != nil {
				reply(token, chat, msgID, "❌ "+esc(err.Error()), navKeyboard("m:git:proj:list", "Projects"))
				return true
			}
			fail := 0
			for _, pr := range list {
				out, err := gitstore.MirrorPush(pr.Name)
				line := "✅ " + pr.Name
				if err != nil {
					fail++
					line = "❌ " + pr.Name + " — " + err.Error()
				}
				if len(b.String())+len(line) > 3500 {
					b.WriteString("…" + nl)
					break
				}
				b.WriteString(esc(line) + nl)
				_ = out
			}
			b.WriteString(fmt.Sprintf("fail=%d%s", fail, nl))
			reply(token, chat, msgID, b.String(), navKeyboard("m:git:proj:list", "Projects"))
			return true
		}
		if strings.HasPrefix(sub, "vps-cancel:") {
			id := strings.TrimPrefix(sub, "vps-cancel:")
			err := gitstore.CancelVpsJob(id)
			msg := "✅ cancelled " + esc(id)
			if err != nil {
				msg = "❌ " + esc(err.Error())
			}
			reply(token, chat, msgID, msg, navKeyboard("m:git:proj:vps-queue", "VPS queue"))
			return true
		}
		if sub == "vps-queue" {
			jobs := gitstore.ListVpsQueue(false)
			var b strings.Builder
			nl := string([]byte{10})
			b.WriteString("📋 <b>VPS build queue</b> (one at a time)" + nl)
			if len(jobs) == 0 {
				b.WriteString("<i>empty</i>" + nl)
			}
			rows := [][]map[string]any{}
			for _, j := range jobs {
				line := fmt.Sprintf("%s %s %s %s", j.Status, j.Kind, j.Project, j.ID)
				if j.Error != "" {
					line += " — " + j.Error
				}
				if len(b.String())+len(line) > 3000 {
					b.WriteString("…" + nl)
					break
				}
				b.WriteString(esc(line) + nl)
				if j.Status == "pending" || j.Status == "running" {
					rows = append(rows, []map[string]any{btn("🛑 "+j.Project, "m:git:proj:vps-cancel:"+j.ID, "")})
				}
			}
			rows = append(rows, []map[string]any{btn(T("projects_back"), "m:git:proj:list", "primary"), btn(T("main_menu"), "m:menu", "")})
			reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": rows})
			return true
		}
		if sub == "sync-all" {
			results, okN, failN := gitstore.SyncAll()
			var b strings.Builder
			nl := string([]byte{10})
			b.WriteString(fmt.Sprintf("sync-all ok=%d fail=%d%s", okN, failN, nl))
			for _, r := range results {
				st := "✅"
				if !r.OK {
					st = "❌"
				}
				line := st + " " + esc(r.Name)
				if r.Error != "" {
					line += " — " + esc(r.Error)
				}
				if len(b.String())+len(line) > 3500 {
					b.WriteString("…" + nl)
					break
				}
				b.WriteString(line + nl)
			}
			reply(token, chat, msgID, b.String(), navKeyboard("m:git:proj:list", "Projects"))
			return true
		}
		if sub == "build-all" || sub == "ship-all" {
			queued, skipped, err := gitstore.ShipAllVPS()
			nl := string([]byte{10})
			var b strings.Builder
			if err != nil {
				b.WriteString("❌ " + esc(err.Error()))
			} else {
				b.WriteString(fmt.Sprintf("🚀 ship-all queued=%d skipped=%d%s", len(queued), len(skipped), nl))
				for _, id := range queued {
					b.WriteString("· <code>" + esc(id) + "</code>" + nl)
				}
				for _, s := range skipped {
					b.WriteString("skip " + esc(s) + nl)
				}
			}
			reply(token, chat, msgID, b.String(), navKeyboard("m:git:ship:jobs", "Jobs"))
			return true
		}
		if sub == "build-all-legacy-disabled" {
			results, okN, failN, skipN := gitstore.BuildAll(true)
			var b strings.Builder
			nl := string([]byte{10})
			b.WriteString(fmt.Sprintf("build-all ok=%d fail=%d skip=%d%s", okN, failN, skipN, nl))
			for _, r := range results {
				st := "✅"
				if r.Skipped {
					st = "⏭"
				} else if !r.OK {
					st = "❌"
				}
				line := st + " " + esc(r.Name)
				if r.Error != "" {
					line += " — " + esc(r.Error)
				}
				if len(b.String())+len(line) > 3500 {
					b.WriteString("…" + nl)
					break
				}
				b.WriteString(line + nl)
			}
			reply(token, chat, msgID, b.String(), navKeyboard("m:git:proj:list", "Projects"))
			return true
		}
		if sub == "migrate" || strings.HasPrefix(sub, "migrate:") {
			owner := "PavelNeyman"
			dry, sync := true, false
			if strings.HasPrefix(sub, "migrate:") {
				rest := strings.TrimPrefix(sub, "migrate:")
				// migrate:dry | migrate:sync | migrate:owner:NAME
				switch rest {
				case "dry":
					dry, sync = true, false
				case "sync":
					dry, sync = false, true
				default:
					if strings.HasPrefix(rest, "owner:") {
						owner = strings.TrimPrefix(rest, "owner:")
					}
				}
			}
			// confirm menu
			if sub == "migrate" {
				var b strings.Builder
				nl := string([]byte{10})
				if ru {
					b.WriteString("📥 <b>Миграция с GitHub</b>" + nl)
					b.WriteString("owner: <code>" + esc(owner) + "</code>" + nl)
					b.WriteString("netductor пропускается. Swift/iOS → host=mac." + nl)
				} else {
					b.WriteString("📥 <b>Migrate from GitHub</b>" + nl)
					b.WriteString("owner: <code>" + esc(owner) + "</code>" + nl)
					b.WriteString("Skips netductor. Swift/iOS → host=mac." + nl)
				}
				kb := map[string]any{"inline_keyboard": [][]map[string]any{
					{btn("👁 dry-run", "m:git:proj:migrate:dry", "primary"), btn("✅ sync", "m:git:proj:migrate:sync", "")},
					{btn(T("projects_back"), "m:git:proj:list", ""), btn(T("main_menu"), "m:menu", "")},
				}}
				reply(token, chat, msgID, b.String(), kb)
				return true
			}
			added, skipped, err := gitstore.MigrateFromGitHub(gitstore.MigrateOpts{Owner: owner, DryRun: dry, Sync: sync})
			var b strings.Builder
			nl := string([]byte{10})
			if err != nil {
				reply(token, chat, msgID, "❌ "+esc(err.Error()), navKeyboard("m:git:proj:list", "Projects"))
				return true
			}
			b.WriteString(fmt.Sprintf("added=%d skipped=%d dry=%v sync=%v%s", len(added), len(skipped), dry, sync, nl))
			for _, a := range added {
				if len(b.String()) > 3500 {
					b.WriteString("…" + nl)
					break
				}
				b.WriteString("+ " + esc(a) + nl)
			}
			for _, s := range skipped {
				if len(b.String()) > 3500 {
					break
				}
				b.WriteString("- " + esc(s) + nl)
			}
			reply(token, chat, msgID, b.String(), navKeyboard("m:git:proj:list", "Projects"))
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
				b.WriteString(" · ⚠️ <b>build_on_fetch ON</b> (heavy)")
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
			if d.LastVpsStatus != "" {
				b.WriteString(nl + "<b>last VPS</b> " + esc(d.LastVpsStatus))
				if d.LastVpsKind != "" {
					b.WriteString(" · " + esc(d.LastVpsKind))
				}
				if d.LastVpsAt != "" {
					b.WriteString(" · " + esc(d.LastVpsAt))
				}
				b.WriteString(nl)
				if d.LastVpsError != "" {
					b.WriteString("❌ " + esc(d.LastVpsError) + nl)
				}
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
				{btn("⬇️ Sync", "m:git:proj:sync:"+name, ""), btn(T("ship_btn"), "m:git:proj:ship:"+name, "primary")},
				{btn(T("projects_back"), "m:git:proj:list", "primary"), btn(T("main_menu"), "m:menu", "")},
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
				msg = "✅ sync\n" + msg
			}
			reply(token, chat, msgID, msg, navKeyboard("m:git:proj:list", "Projects"))
			return true
		}
		if strings.HasPrefix(sub, "build:") || strings.HasPrefix(sub, "ship:") {
			name := strings.TrimPrefix(sub, "build:")
			name = strings.TrimPrefix(name, "ship:")
			tag := gitstore.NextProjectTag(name)
			job, err := gitstore.EnqueueShipJob(gitstore.ShipOpts{Project: name, Ref: "", Tag: tag})
			msg := ""
			if job != nil {
				msg = "🚀 ship queued\njob <code>" + esc(job.ID) + "</code> state=<code>" + esc(job.State) + "</code> tag=<code>" + esc(job.Tag) + "</code>"
			}
			if err != nil {
				msg = "❌ " + esc(err.Error())
			}
			kb := navKeyboard("m:git:proj:list", "Projects")
			if job != nil {
				kb = map[string]any{"inline_keyboard": [][]map[string]any{
					{btn("📋 Jobs", "m:git:ship:jobs", "primary"), btn(T("projects_back"), "m:git:proj:list", "")},
					{btn(T("main_menu"), "m:menu", "")},
				}}
			}
			reply(token, chat, msgID, msg, kb)
			return true
		}
		if sub == "ship-jobs" || sub == "jobs" {
			// fallthrough handled below via m:git:ship
		}
	}
	return false
}
