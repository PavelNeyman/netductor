package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func handleJobsMenu(token string, chat int64, msgID int, data string) bool {
	if data == "m:doctor:run" {
		out, err := exec.Command("/usr/local/bin/netductor", "job", "start", "doctor.run").CombinedOutput()
		id := strings.TrimSpace(string(out))
		var msg string
		if err != nil {
			msg = "❌ " + esc(err.Error()) + "\n<code>" + esc(id) + "</code>"
		} else {
			msg = T("jobs_doctor_started") + " <code>" + esc(id) + "</code>\n<i>" + T("jobs_doctor_hint") + "</i>"
		}
		reply(token, chat, msgID, msg, navKeyboard("m:jobs", parentTools()))
		return true
	}
	if data != "m:jobs" && !strings.HasPrefix(data, "m:jobs:") {
		return false
	}
	if data == "m:jobs" || data == "m:jobs:list" {
		text, rows := renderPrimaryJobsTG("")
		reply(token, chat, msgID, text, map[string]any{"inline_keyboard": rows})
		return true
	}
	if data == "m:jobs:running" {
		text, rows := renderPrimaryJobsTG("running")
		reply(token, chat, msgID, text, map[string]any{"inline_keyboard": rows})
		return true
	}
	if strings.HasPrefix(data, "m:jobs:cancel:") {
		id := strings.TrimPrefix(data, "m:jobs:cancel:")
		out, err := exec.Command("/usr/local/bin/netductor", "job", "cancel", id).CombinedOutput()
		msg := T("jobs_cancelled") + " <code>" + esc(id) + "</code>\n<code>" + esc(strings.TrimSpace(string(out))) + "</code>"
		if err != nil {
			msg = "❌ " + esc(err.Error()) + "\n<code>" + esc(strings.TrimSpace(string(out))) + "</code>"
		}
		reply(token, chat, msgID, msg, navKeyboard("m:jobs", parentTools()))
		return true
	}
	return false
}

func renderPrimaryJobsTG(statusFilter string) (string, [][]map[string]any) {
	args := []string{"job", "list"}
	if statusFilter != "" {
		args = append(args, "--status", statusFilter)
	}
	out, err := exec.Command("/usr/local/bin/netductor", args...).CombinedOutput()
	var b strings.Builder
	b.WriteString("📋 <b>" + T("jobs_title_primary") + "</b>\n")
	b.WriteString("<i>" + T("jobs_types_hint") + "</i>\n")
	rows := [][]map[string]any{}
	if err != nil {
		b.WriteString("❌ " + esc(err.Error()) + "\n<code>" + esc(strings.TrimSpace(string(out))) + "</code>\n")
		rows = append(rows, []map[string]any{btn(T("refresh"), "m:jobs:list", "primary")})
		rows = append(rows, []map[string]any{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")})
		return b.String(), rows
	}
	var m map[string]any
	_ = json.Unmarshal(out, &m)
	jobs, _ := m["jobs"].([]any)
	if len(jobs) == 0 {
		b.WriteString("<i>" + T("jobs_empty") + "</i>\n")
	} else {
		b.WriteString("<table bordered striped>\n<tr><th>#</th><th>type</th><th>status</th><th>id</th></tr>\n")
		for i, j := range jobs {
			jm, _ := j.(map[string]any)
			id, _ := jm["id"].(string)
			typ, _ := jm["type"].(string)
			st, _ := jm["status"].(string)
			short := id
			if len(short) > 10 {
				short = short[:10]
			}
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td><code>%s</code></td></tr>\n", i+1, esc(typ), esc(st), esc(short)))
			if st == "running" || st == "queued" {
				lab := "⛔ " + fmt.Sprintf("%d", i+1)
				rows = append(rows, []map[string]any{btn(lab, "m:jobs:cancel:"+id, "danger")})
			}
		}
		b.WriteString("</table>\n")
		b.WriteString("<i>" + T("jobs_cancel_hint") + "</i>\n")
	}
	rows = append(rows, []map[string]any{btn(T("refresh"), "m:jobs:list", "primary"), btn(T("jobs_running_btn"), "m:jobs:running", "")})
	rows = append(rows, []map[string]any{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")})
	return b.String(), rows
}
