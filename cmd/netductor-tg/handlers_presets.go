package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/devices"
	"github.com/PavelNeyman/netductor/internal/policy"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

type presetDraft struct {
	Title         string   `json:"title"`
	AllowInternet bool     `json:"allow_internet"`
	ServicesMode  string   `json:"services_mode"`
	Services      []string `json:"services"`
}

func loadDraft(chat int64) presetDraft {
	var d presetDraft
	if chatExtra[chat] != "" {
		_ = json.Unmarshal([]byte(chatExtra[chat]), &d)
	}
	if d.ServicesMode == "" {
		d.ServicesMode = "list"
		d.AllowInternet = true
	}
	if d.Services == nil {
		d.Services = []string{}
	}
	return d
}

func saveDraft(chat int64, d presetDraft) {
	b, _ := json.Marshal(d)
	chatExtra[chat] = string(b)
}

func handlePresetsCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:presets" && !strings.HasPrefix(data, "m:presets:") {
		return false
	}
	if data == "m:presets" {
		reply(token, chat, msgID, formatPresetsHub(), presetsHubKB())
		return true
	}
	rest := strings.TrimPrefix(data, "m:presets:")
	switch {
	case rest == "new":
		d := presetDraft{AllowInternet: true, ServicesMode: "list"}
		saveDraft(chat, d)
		setState(chat, "preset_draft", chatExtra[chat])
		reply(token, chat, msgID, formatPresetDraft(d), presetDraftKB(d))
	case rest == "list":
		reply(token, chat, msgID, formatPresetsHub(), presetsHubKB())
	case rest == "draft:net":
		d := loadDraft(chat)
		d.AllowInternet = !d.AllowInternet
		saveDraft(chat, d)
		reply(token, chat, msgID, formatPresetDraft(d), presetDraftKB(d))
	case rest == "draft:mode":
		d := loadDraft(chat)
		if d.ServicesMode == "all" {
			d.ServicesMode = "list"
		} else {
			d.ServicesMode = "all"
		}
		saveDraft(chat, d)
		reply(token, chat, msgID, formatPresetDraft(d), presetDraftKB(d))
	case strings.HasPrefix(rest, "draft:svc:"):
		id := strings.TrimPrefix(rest, "draft:svc:")
		d := loadDraft(chat)
		found := false
		var next []string
		for _, s := range d.Services {
			if s == id {
				found = true
				continue
			}
			next = append(next, s)
		}
		if !found {
			next = append(next, id)
		}
		d.Services = next
		d.ServicesMode = "list"
		saveDraft(chat, d)
		reply(token, chat, msgID, formatPresetDraft(d), presetDraftKB(d))
	case rest == "draft:save":
		setState(chat, "wait_preset_name", chatExtra[chat])
		msg := "Send preset <b>name</b> (or /cancel):"
		if getLang() != "en" {
			msg = "Пришлите <b>имя</b> пресета (или /cancel):"
		}
		reply(token, chat, msgID, msg, presetsHubKB())
	case strings.HasPrefix(rest, "del:"):
		id := strings.TrimPrefix(rest, "del:")
		_ = policy.DeleteCustomPreset(id)
		reply(token, chat, msgID, "✅ deleted <code>"+esc(id)+"</code>\n"+formatPresetsHub(), presetsHubKB())
	case strings.HasPrefix(rest, "use:"):
		pid := strings.TrimPrefix(rest, "use:")
		reply(token, chat, msgID, formatPresetUseUsers(pid), presetUseUsersKB(pid))
	case strings.HasPrefix(rest, "set:"):
		parts := strings.SplitN(strings.TrimPrefix(rest, "set:"), ":", 2)
		if len(parts) != 2 {
			reply(token, chat, msgID, "❌", presetsHubKB())
			return true
		}
		pid, uname := parts[0], parts[1]
		cur, _ := vpn.GetUserPolicy(uname)
		next := policy.ApplyNamedPreset(pid, cur)
		if err := vpn.SetUserPolicy(uname, next, "tg"); err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), presetsHubKB())
			return true
		}
		_, _ = policy.ApplyRoutes()
		reply(token, chat, msgID, fmt.Sprintf("✅ preset <code>%s</code> → <b>%s</b>", esc(pid), esc(uname)), presetsHubKB())
	}
	return true
}

func formatPresetsHub() string {
	list, _ := policy.LoadCustomPresets()
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("📋 <b>Пресеты доступа</b>\nВстроенные: full, media, none\n")
	} else {
		b.WriteString("📋 <b>Access presets</b>\nBuilt-in: full, media, none\n")
	}
	if len(list) == 0 {
		b.WriteString("<i>No custom presets yet.</i>\n")
	}
	for _, p := range list {
		b.WriteString(fmt.Sprintf("• <code>%s</code> — %s (net=%v mode=%s svc=%v)\n",
			esc(p.ID), esc(p.Title), p.AllowInternet, esc(p.ServicesMode), p.Services))
	}
	return b.String()
}

func formatPresetDraft(d presetDraft) string {
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("🛠 <b>Черновик пресета</b>\n")
	} else {
		b.WriteString("🛠 <b>Preset draft</b>\n")
	}
	b.WriteString(fmt.Sprintf("internet=%v mode=%s services=%v\n", d.AllowInternet, d.ServicesMode, d.Services))
	b.WriteString("<i>Toggle services, then Save → send name</i>")
	return b.String()
}

func presetsHubKB() map[string]any {
	rows := [][]map[string]any{
		{btn("➕ New", "m:presets:new", "primary"), btn("🔄", "m:presets", "")},
	}
	list, _ := policy.LoadCustomPresets()
	for _, p := range list {
		rows = append(rows, []map[string]any{
			btn("📌 "+p.Title, "m:presets:use:"+p.ID, "primary"),
			btn("🗑", "m:presets:del:"+p.ID, "danger"),
		})
	}
	rows = append(rows, []map[string]any{btn("« Users", "m:users", ""), btn(T("main_menu"), "m:menu", "")})
	return map[string]any{"inline_keyboard": rows}
}

func presetDraftKB(d presetDraft) map[string]any {
	netMark := "☐ net"
	if d.AllowInternet {
		netMark = "☑ net"
	}
	mode := "mode:list"
	if d.ServicesMode == "all" {
		mode = "mode:all"
	}
	rows := [][]map[string]any{
		{btn(netMark, "m:presets:draft:net", ""), btn(mode, "m:presets:draft:mode", "")},
	}
	cat, _ := policy.EnsureCatalog()
	allowed := map[string]struct{}{}
	for _, s := range d.Services {
		allowed[s] = struct{}{}
	}
	if cat != nil {
		var row []map[string]any
		for _, s := range cat.Services {
			if s.Disabled || s.Kind == policy.KindEgress || s.ID == "internet" {
				continue
			}
			mark := "☐ " + s.ID
			if d.ServicesMode == "all" {
				mark = "☑ " + s.ID
			} else if _, ok := allowed[s.ID]; ok {
				mark = "☑ " + s.ID
			}
			row = append(row, btn(mark, "m:presets:draft:svc:"+s.ID, ""))
			if len(row) == 2 {
				rows = append(rows, row)
				row = nil
			}
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	rows = append(rows, []map[string]any{
		btn("💾 Save", "m:presets:draft:save", "success"), btn("«", "m:presets", ""),
	})
	return map[string]any{"inline_keyboard": rows}
}

func formatPresetUseUsers(pid string) string {
	return "Apply preset <code>" + esc(pid) + "</code> to user:"
}

func presetUseUsersKB(pid string) map[string]any {
	users, _ := vpn.List()
	var rows [][]map[string]any
	for _, u := range users {
		if u.Name == "relay-uplink" {
			continue
		}
		rows = append(rows, []map[string]any{btn(u.Name, "m:presets:set:"+pid+":"+u.Name, "primary")})
	}
	rows = append(rows, []map[string]any{btn("«", "m:presets", "")})
	return map[string]any{"inline_keyboard": rows}
}

func handlePresetNameText(token string, chat int64, text string) bool {
	if chatState[chat] != "wait_preset_name" {
		return false
	}
	d := loadDraft(chat)
	name := strings.TrimSpace(text)
	if name == "" {
		return true
	}
	p := policy.CustomPreset{
		Title: name, AllowInternet: d.AllowInternet,
		ServicesMode: d.ServicesMode, Services: d.Services,
	}
	out, err := policy.UpsertCustomPreset(p)
	setState(chat, "", "")
	if err != nil {
		sendHTML(token, chat, "❌ "+esc(err.Error()), presetsHubKB())
		return true
	}
	sendHTML(token, chat, fmt.Sprintf("✅ preset <code>%s</code> saved", esc(out.ID)), presetsHubKB())
	return true
}

func handleDevicesCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:devices" && !strings.HasPrefix(data, "m:devices:") {
		return false
	}
	_, _ = devices.RefreshFromJournal(120)
	list := devices.List()
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("📱 <b>Устройства VPN</b> (по логам)\n")
	} else {
		b.WriteString("📱 <b>VPN devices</b> (from logs)\n")
	}
	if len(list) == 0 {
		b.WriteString("<i>empty — traffic generates entries</i>\n")
	}
	n := 0
	for _, d := range list {
		b.WriteString(fmt.Sprintf("• <b>%s</b> <code>%s</code> hits=%d last=%s\n",
			esc(d.User), esc(d.SrcIP), d.Hits, d.LastSeen.Format("15:04")))
		n++
		if n >= 25 {
			break
		}
	}
	reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("🔄", "m:devices", ""), btn("« Users", "m:users", ""), btn(T("main_menu"), "m:menu", "")},
	}})
	return true
}

func handleIncidentCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:incident" {
		return false
	}
	out := runND("channel", "status")
	msg := "📦 <b>Incident snapshot</b>\n<pre>" + esc(trimRun(out, 2500)) + "</pre>\n"
	msg += "Full pack: API <code>/api/incident/collect</code>\n"
	reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("Channel", "m:channel", ""), btn(T("main_menu"), "m:menu", "")},
	}})
	return true
}

func trimRun(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
