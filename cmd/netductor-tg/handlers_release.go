package main

import (
	"os/exec"
	"strings"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

func handleReleaseCB(token string, chat int64, msgID int, data string) {
	nav := releaseNav()
	switch {
	case data == "m:release":
		var b strings.Builder
		b.WriteString("📦 <b>Releases (local store)</b>\n")
		b.WriteString("<i>Prefer /var/lib/netductor/releases before GitHub. Build is async on primary.</i>\n")
		tags, _ := ndupdate.ListLocalTags()
		if len(tags) == 0 {
			b.WriteString("\n(empty local store)\n")
		} else {
			b.WriteString("\n")
			for _, tg := range tags {
				b.WriteString("• <code>" + esc(tg) + "</code>\n")
			}
			b.WriteString("\n<i>Apply uses local store first, then GitHub.</i>\n")
		}
		reply(token, chat, msgID, b.String(), releaseNavWithApply(tags))
	case data == "m:release:local":
		handleReleaseCB(token, chat, msgID, "m:release")
	case data == "m:release:tags":
		out, err := exec.Command("netductor", "git", "tags", "netductor").CombinedOutput()
		msg := "<b>Git mirror tags</b>\n<pre>" + esc(string(out)) + "</pre>"
		if err != nil {
			msg = "⚠️ " + esc(err.Error()) + "\n" + msg + "\n<i>Run Mirror fetch first</i>"
		}
		reply(token, chat, msgID, msg, nav)
	case strings.HasPrefix(data, "m:release:apply:"):
		tag := strings.TrimPrefix(data, "m:release:apply:")
		out, err := exec.Command("netductor", "stack", "apply", tag).CombinedOutput()
		msg := "✅ stack apply <code>" + esc(tag) + "</code>\n<pre>" + esc(trimOut(string(out), 2000)) + "</pre>"
		if err != nil {
			msg = "❌ " + esc(err.Error()) + "\n<pre>" + esc(trimOut(string(out), 1500)) + "</pre>"
		}
		reply(token, chat, msgID, msg, releaseNav())
	case data == "m:release:status":
		out, _ := exec.Command("netductor", "release", "list-local").CombinedOutput()
		b, _ := exec.Command("systemctl", "is-active", "netductor-release-build.service").CombinedOutput()
		msg := "<b>Build unit</b>: <code>" + esc(strings.TrimSpace(string(b))) + "</code>\n<pre>" + esc(string(out)) + "</pre>"
		reply(token, chat, msgID, msg, releaseNav())
	case data == "m:release:mirror":
		_ = exec.Command("netductor", "git", "mirror-ensure", "netductor").Run()
		out, err := exec.Command("netductor", "git", "mirror-fetch", "netductor").CombinedOutput()
		msg := "✅ <b>Mirror fetch</b>\n<pre>" + esc(trimOut(string(out), 2500)) + "</pre>"
		if err != nil {
			msg = "❌ " + esc(err.Error()) + "\n<pre>" + esc(trimOut(string(out), 1500)) + "</pre>"
		}
		reply(token, chat, msgID, msg, nav)
	default:
		reply(token, chat, msgID, "Unknown release action", nav)
	}
}

func trimOut(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}


func releaseNav() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("📦 Local", "m:release:local", "primary"), btn("🏷 Git tags", "m:release:tags", "")},
		{btn("⬇️ Mirror fetch", "m:release:mirror", ""), btn("📊 Status", "m:release:status", "")},
		{btn("⬅️ Tools", "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}

func releaseNavWithApply(tags []string) map[string]any {
	rows := [][]map[string]any{}
	for i, tg := range tags {
		if i >= 4 {
			break
		}
		rows = append(rows, []map[string]any{btn("▶ "+tg, "m:release:apply:"+tg, "primary")})
	}
	rows = append(rows,
		[]map[string]any{btn("📦 Local", "m:release:local", "primary"), btn("🏷 Git tags", "m:release:tags", "")},
		[]map[string]any{btn("⬇️ Mirror fetch", "m:release:mirror", ""), btn("📊 Status", "m:release:status", "")},
		[]map[string]any{btn("⬅️ Tools", "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	)
	return map[string]any{"inline_keyboard": rows}
}
