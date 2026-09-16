package main

import (
	"github.com/PavelNeyman/netductor/internal/sites"
	"github.com/PavelNeyman/netductor/internal/vpn"
	"github.com/PavelNeyman/netductor/internal/install"
	"fmt"
	"strconv"
	"strings"
)

func handleMessage(token string, m *message, admin int64) {
	if m.Chat.ID != admin {
		return
	}
	chat := m.Chat.ID
	text := strings.TrimSpace(m.Text)

	if text == "/cancel" {
		setState(chat, "", "")
		sendHTML(token, chat, T("cancelled"), mainKeyboard())
		return
	}

	st := chatState[chat]
	if st == "wait_node_newname" {
		id := chatExtra[chat]
		newName := strings.Fields(text)
		setState(chat, "", "")
		if id == "" || len(newName) == 0 {
			sendHTML(token, chat, T("nodes_rename"), nodesRenameKeyboard())
			return
		}
		out := runND("nodes", "rename", id, newName[0])
		// try apply on local if this is the VPS
		_ = runND("nodes", "sync-local")
		sendHTML(token, chat, fmt.Sprintf(T("nodes_done"), esc(id), esc(newName[0]))+string([]byte{10, 10})+"<pre>"+esc(out)+"</pre>"+string([]byte{10})+formatNodesListHTML(), nodesKeyboard())
		return
	}
	if strings.HasPrefix(st, "wait_loc_rename:") {
		id := strings.TrimPrefix(st, "wait_loc_rename:")
		setState(chat, "", "")
		s, ok := sites.Get(id)
		if !ok {
			sendHTML(token, chat, "❌ not found", nil)
			return
		}
		s.Name = strings.TrimSpace(text)
		if _, err := sites.Upsert(s); err != nil {
			sendHTML(token, chat, "❌ "+esc(err.Error()), nil)
			return
		}
		sendHTML(token, chat, "✅ "+esc(s.Name), map[string]any{"inline_keyboard": [][]map[string]any{{btn("📍", "m:loc:open:"+id, ""), btn(T("main_menu"), "m:menu", "primary")}}})
		return
	}
	if strings.HasPrefix(st, "wait_quota:") {
		name := strings.TrimPrefix(st, "wait_quota:")
		setState(chat, "", "")
		gb, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || gb < 0 {
			sendHTML(token, chat, "❌ Число GiB, например 100", nil)
			return
		}
		_ = vpn.SetSoftLimitGB(name, gb)
		sendHTML(token, chat, formatUserHubHTML(name), userHubKeyboard(name))
		return
	}
	if st == "wait_backup_time" {
		setState(chat, "", "")
		h, m, err := install.ParseHHMM(text)
		if err != nil {
			sendHTML(token, chat, "❌ "+esc(err.Error()), nil)
			return
		}
		s := install.BackupSchedule{Hour: h, Minute: m, UTC: true}
		if err := install.SaveBackupSchedule(s); err != nil {
			sendHTML(token, chat, "❌ "+esc(err.Error()), nil)
			return
		}
		sendHTML(token, chat, "✅ Backup schedule: <code>"+install.FormatBackupSchedule(s)+"</code>", map[string]any{"inline_keyboard": [][]map[string]any{{btn("🗓 Backup", "m:backup", ""), btn(T("main_menu"), "m:menu", "primary")}}})
		return
	}
	if st == "wait_ssh_forget" {
		id := strings.Fields(text)[0]
		out := runND("ssh-hosts", "forget", id)
		setState(chat, "", "")
		sendHTML(token, chat, fmt.Sprintf(T("ssh_forgot"), esc(id))+string([]byte{10})+"<pre>"+esc(out)+"</pre>"+string([]byte{10, 10})+formatSSHHostsHTML(), sshHostsKeyboard())
		return
	}
	if st == "wait_backup_peer" {
		setState(chat, "", "")
		out := runND("backup", "peer-set", strings.TrimSpace(text))
		sendHTML(token, chat, "💾 <pre>"+esc(out)+"</pre>", nodesKeyboard())
		return
	}
	if strings.HasPrefix(st, "wait_rename_new:") {
		old := strings.TrimPrefix(st, "wait_rename_new:")
		newName := strings.Fields(text)
		setState(chat, "", "")
		if len(newName) < 1 {
			sendHTML(token, chat, Tf("rename_new_hint", old), userHubKeyboard(old))
			return
		}
		out := runVPN("rename", old, newName[0])
		sendHTML(token, chat, "✏️ <pre>"+esc(out)+"</pre>"+string([]byte{10, 10})+formatUserHubHTML(newName[0]), userHubKeyboard(newName[0]))
		return
	}
	if st == "wait_vpn_rename" {
		fields := strings.Fields(text)
		setState(chat, "", "")
		if len(fields) < 2 {
			sendHTML(token, chat, T("vpn_rename_hint"), vpnKeyboard())
			return
		}
		out := runVPN("rename", fields[0], fields[1])
		sendHTML(token, chat, "✏️ <pre>"+esc(out)+"</pre>", vpnKeyboard())
		return
	}
	if st == "wait_vpn_add_name" {
		fields := strings.Fields(text)
		if len(fields) == 0 {
			sendHTML(token, chat, T("vpn_add"), backKeyboard())
			return
		}
		name := fields[0]
		out := runVPN("add", name)
		setState(chat, "", "")
		sub := runVPN("link", name)
		sendHTML(token, chat, "✅ <pre>"+esc(out)+"</pre>\n\n<pre>"+esc(sub)+"</pre>", userCardKeyboard(name, strings.TrimSpace(sub)))
		return
	}
		if st == "wait_relay_host" {
		setState(chat, "wait_relay_user:"+text, "")
		sendHTML(token, chat, T("enroll_user"), backTo("nodes"))
		return
	}
	if strings.HasPrefix(st, "wait_relay_user:") {
		host := strings.TrimPrefix(st, "wait_relay_user:")
		user := text
		if user == "" {
			user = "root"
		}
		setState(chat, "wait_relay_pass:"+host+"|"+user, "")
		sendHTML(token, chat, T("enroll_pass"), backTo("nodes"))
		return
	}
	if strings.HasPrefix(st, "wait_relay_pass:") {
		rest := strings.TrimPrefix(st, "wait_relay_pass:")
		parts := strings.SplitN(rest, "|", 2)
		host, user := rest, "root"
		if len(parts) == 2 {
			host, user = parts[0], parts[1]
		}
		pass := text
		setState(chat, "", "")
		sendHTML(token, chat, fmt.Sprintf(T("enroll_wait"), esc(host)), nil)
		out := runND("relay", "provision", "--host", host, "--user", user, "--password", pass, "--sni", "ya.ru")
		sendHTML(token, chat, "✅ <b>Relay</b>"+string([]byte{10})+"<pre>"+esc(truncate(out, 3500))+"</pre>"+string([]byte{10})+formatRelayListHTML(), relayKeyboard())
		return
	}

if strings.HasPrefix(st, "wait_vpn_name:") {
		action := strings.TrimPrefix(st, "wait_vpn_name:")
		fields := strings.Fields(text)
		if len(fields) == 0 {
			setState(chat, "", "")
			sendHTML(token, chat, T("unknown"), mainKeyboard())
			return
		}
		name := fields[0]
		setState(chat, "", "")
		switch action {
		case "vpn_link":
			sub := runVPN("link", name, "vless")
			sendHTML(token, chat, "🔗 <b>"+esc(name)+"</b>\n\n<pre>"+esc(sub)+"</pre>", userCardKeyboard(name, strings.TrimSpace(sub)))
		case "vpn_sub":
			sendHTML(token, chat, "ℹ️ Subscription removed. Open user → Access.", userHubKeyboard(name))
		case "vpn_disable":
			sendHTML(token, chat, "🚫 <pre>"+esc(runVPN("disable", name))+"</pre>", backKeyboard())
		case "vpn_enable":
			sendHTML(token, chat, "✅ <pre>"+esc(runVPN("enable", name))+"</pre>", backKeyboard())
		case "vpn_revoke":
			sendHTML(token, chat, "🗑 <pre>"+esc(runVPN("revoke", name))+"</pre>", backKeyboard())
		}
		return
	}
	if st == "wait_session_hours" {
		h := text
		if h == "" {
			h = "72"
		}
		setState(chat, "", "")
		tok := runVPN("session", h)
		fields := strings.Fields(tok)
		copyVal := tok
		if len(fields) > 0 {
			copyVal = fields[0]
		}
		kb := map[string]any{
			"inline_keyboard": [][]map[string]any{
				{btnCopy(T("copy_token"), copyVal)},
				{btn(T("main_menu"), "m:menu", "primary")},
			},
		}
		sendHTML(token, chat, formatSessionHTML(tok), kb)
		return
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		sendHTML(token, chat, menuText(), mainKeyboard())
		return
	}
	cmd := parts[0]
	arg1 := ""
	if len(parts) > 1 {
		arg1 = parts[1]
	}

	switch cmd {
	case "/start", "/menu":
		sendHTML(token, chat, menuText(), mainKeyboard())
	case "/help":
		sendHTML(token, chat, helpText(), mainKeyboard())
	case "/lang":
		if arg1 == "en" || arg1 == "ru" {
			setLang(arg1)
			name := "Русский"
			if arg1 == "en" {
				name = "English"
			}
			sendHTML(token, chat, Tf("lang_set", name), mainKeyboard())
		} else {
			cur := getLang()
			label := "Русский"
			if cur == "en" {
				label = "English"
			}
			sendHTML(token, chat, Tf("lang_now", label), langKeyboard())
		}
	case "/status":
		sendHTML(token, chat, formatStatusPretty(), backKeyboard())
	case "/vpn_list":
		sendHTML(token, chat, formatVPNListPretty(runVPN("list")), backKeyboard())
	case "/vpn_add":
		if arg1 == "" {
			setState(chat, "wait_vpn_add_name", "")
			sendHTML(token, chat, T("add_prompt"), backKeyboard())
		} else {
			out := runVPN("add", arg1)
			sub := runVPN("link", arg1)
			sendHTML(token, chat, "✅ <pre>"+esc(out)+"</pre>\n\n<pre>"+esc(sub)+"</pre>", userCardKeyboard(arg1, strings.TrimSpace(sub)))
		}
	case "/vpn_disable":
		sendHTML(token, chat, "🚫 <pre>"+esc(runVPN("disable", arg1))+"</pre>", backKeyboard())
	case "/vpn_enable":
		sendHTML(token, chat, "✅ <pre>"+esc(runVPN("enable", arg1))+"</pre>", backKeyboard())
	case "/vpn_revoke":
		sendHTML(token, chat, "🗑 <pre>"+esc(runVPN("revoke", arg1))+"</pre>", backKeyboard())
	case "/session":
		h := "72"
		if arg1 != "" {
			h = arg1
		}
		tok := runVPN("session", h)
		fields := strings.Fields(tok)
		copyVal := tok
		if len(fields) > 0 {
			copyVal = fields[0]
		}
		kb := map[string]any{
			"inline_keyboard": [][]map[string]any{
				{btnCopy(T("copy_token"), copyVal)},
				{btn(T("main_menu"), "m:menu", "primary")},
			},
		}
		sendHTML(token, chat, formatSessionHTML(tok), kb)
	case "/admin":
		sendHTML(token, chat, T("admin_body"), mainKeyboard())
	default:
		sendHTML(token, chat, T("unknown_cmd"), mainKeyboard())
	}
}

