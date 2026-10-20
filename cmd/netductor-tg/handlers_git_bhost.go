package main

import (
	"fmt"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func handleGitBuildHost(token string, chat int64, msgID int, rest string) bool {
	if rest == "build-host" || strings.HasPrefix(rest, "build-host:") {
		list := gitstore.ListBuildHostStates()
		cfg := gitstore.LoadBuildHost()
		nl := string([]byte{10})
		var b strings.Builder
		b.WriteString("🏗 <b>" + T("build_host_title") + "</b>" + nl)
		b.WriteString(esc(gitstore.BuildHostStatusLine()) + nl)
		if len(list) == 0 {
			b.WriteString("<i>" + T("build_host_none") + "</i>" + nl)
		}
		for _, s := range list {
			on := T("offline")
			if s.Online {
				on = T("online")
			}
			b.WriteString(fmt.Sprintf("• <code>%s</code> %s <b>%s</b> disk=%.1fG docker=%v jobs=%d last=%s%s", esc(s.ID), esc(s.Label), on, s.DiskFreeGB, s.Docker, s.JobsDone, esc(s.LastJobID), nl))
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("🔄 "+T("probe"), "m:git:build-host:watch", "primary"), btn(T("git_back"), "m:git", "")},
			{btn(T("main_menu"), "m:menu", "")},
		}}
		if strings.HasPrefix(rest, "build-host:watch") {
			gitstore.WatchBuildHosts()
			b.WriteString(nl + "<i>" + T("probed") + "</i>")
		}
		_ = cfg
		reply(token, chat, msgID, b.String(), kb)
		return true
	}
	return false
}
