package main

import (
	"fmt"
	"os"
	"strings"
)

func handleCallback(token string, cq *callbackQuery, admin int64) {
	if cq.From.ID != admin {
		return
	}
	chat := cq.Message.Chat.ID
	msgID := 0
	if cq.Message != nil {
		msgID = cq.Message.MessageID
	}
	data := cq.Data
	// hourglass toast for long node ops — same UX core & relay
	if strings.HasPrefix(data, "m:nd:u:") || strings.HasPrefix(data, "m:nd:r:") {
		answerCallbackText(token, cq.ID, "⏳ …")
	} else {
		answerCallback(token, cq.ID)
	}

	
	if strings.HasPrefix(data, "e:appr:") {
		id := strings.TrimPrefix(data, "e:appr:")
		out := runND("edge", "approve", id)
		// mask token=... in output
		out = maskTokenKV(out)
		reply(token, chat, msgID, "✅ <pre>"+esc(out)+"</pre>", routersKeyboard())
		return
	}
	if strings.HasPrefix(data, "e:deny:") {
		id := strings.TrimPrefix(data, "e:deny:")
		out := runND("edge", "deny", id)
		reply(token, chat, msgID, "🚫 <pre>"+esc(out)+"</pre>", routersKeyboard())
		return
	}

	if data == "m:mtls" || strings.HasPrefix(data, "m:mtls:") {
		handleMtlsCB(token, chat, msgID, data)
		return
	}

	if data == "m:tools" {
		reply(token, chat, msgID, toolsHubHTML(), toolsKeyboard())
		return
	}
	if data == "m:nvr" || strings.HasPrefix(data, "m:nvr:") {
		handleNVRCB(token, chat, msgID, data)
		return
	}
	if data == "m:updates" || strings.HasPrefix(data, "m:updates:") {
		handleUpdatesCB(token, chat, msgID, data)
		return
	}
	if handleEdgeGuestCB(token, chat, msgID, data) {
		return
	}
	if handleCertsCB(token, chat, msgID, data) {
		return
	}
	if handleGitCB(token, chat, msgID, data) {
		return
	}
	if data == "m:guest" || strings.HasPrefix(data, "m:guest:") {
		handleGuestCB(token, chat, msgID, data)
		return
	}
	if data == "m:dns" || strings.HasPrefix(data, "m:dns:") {
		handleDNSCB(token, chat, msgID, data)
		return
	}
	if data == "m:backup" || strings.HasPrefix(data, "m:backup:") {
		handleBackupCB(token, chat, msgID, data)
		return
	}
	if data == "m:loc" || strings.HasPrefix(data, "m:loc:") {
		handleLocationCB(token, chat, msgID, data)
		return
	}
	if strings.HasPrefix(data, "m:quota:") {
		handleQuotaCB(token, chat, msgID, data)
		return
	}
if strings.HasPrefix(data, "u:") {
		// u:open:name | u:access:name:mode | u:rename:name | u:enable:name | ...
		parts := strings.Split(data, ":")
		if len(parts) >= 3 {
			action, name := parts[1], parts[2]
			switch action {
			case "workcfg":
				sendWorkProfileDocument(token, chat)
			case "open":
				reply(token, chat, msgID, formatUserHubHTML(name), userHubKeyboard(name))
			case "app":
				// u:app:name:mode:sr|happ|incy — deep-link as code (TG forbids custom schemes in url buttons)
				mode, client := "vless", "sr"
				if len(parts) >= 4 && parts[3] != "" {
					mode = parts[3]
				}
				if len(parts) >= 5 && parts[4] != "" {
					client = parts[4]
				}
				sendAppDeepLink(token, chat, name, mode, client)
			case "access":
				mode := "vless"
				if len(parts) >= 4 && parts[3] != "" {
					mode = parts[3]
				}
				showUserAccess(token, chat, msgID, name, mode)
			case "link":
				showUserAccess(token, chat, msgID, name, "vless")
			case "hy2qr":
				showUserAccess(token, chat, msgID, name, "hy2")
			case "vlessqr":
				showUserAccess(token, chat, msgID, name, "vless")
			case "rename":
				setState(chat, "wait_rename_new:"+name, "")
				reply(token, chat, msgID, Tf("rename_new_hint", name), backTo("users"))
			case "enable":
				out := runVPN("enable", name)
				reply(token, chat, msgID, "✅ <pre>"+esc(out)+"</pre>"+string([]byte{10, 10})+formatUserHubHTML(name), userHubKeyboard(name))
			case "disable":
				out := runVPN("disable", name)
				reply(token, chat, msgID, "🚫 <pre>"+esc(out)+"</pre>"+string([]byte{10, 10})+formatUserHubHTML(name), userHubKeyboard(name))
			case "revoke":
				out := runVPN("revoke", name)
				reply(token, chat, msgID, "🗑 <pre>"+esc(out)+"</pre>", usersListKeyboard())
			}
		}
		return
	}

	if strings.HasPrefix(data, "m:ssh:rm:") {
		id := strings.TrimPrefix(data, "m:ssh:rm:")
		out := runND("ssh-hosts", "forget", id)
		reply(token, chat, msgID, fmt.Sprintf(T("ssh_forgot"), esc(id))+string([]byte{10})+"<pre>"+esc(out)+"</pre>"+string([]byte{10, 10})+formatSSHHostsHTML(), sshHostsKeyboard())
		return
	}
	if strings.HasPrefix(data, "m:nd:") {
		// m:nd:o|m|u|r:<id>
		rest := strings.TrimPrefix(data, "m:nd:")
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			act, id := parts[0], parts[1]
			switch act {
			case "o": // open card
				reply(token, chat, msgID, formatNodeDetailHTML(id), nodeCardKeyboard(id))
			case "m": // metrics
				reply(token, chat, msgID, formatNodeDetailHTML(id), nodeCardKeyboard(id))
			case "u": // upgrade
				_ = enqueueNodeCmd(id, "upgrade")
				reply(token, chat, msgID, formatCmdQueuedHTML("upgrade", id, ""), nodeCardKeyboard(id))
			case "r": // reboot
				_ = enqueueNodeCmd(id, "reboot")
				reply(token, chat, msgID, formatCmdQueuedHTML("reboot", id, ""), nodeCardKeyboard(id))
			case "j": // journal
				reply(token, chat, msgID, formatJournalHTML(id), nodeCardKeyboard(id))
			case "s": // restart sing-box
				out := restartSingBox(id)
				reply(token, chat, msgID, "♻️ <b>Restart sing-box</b>"+string([]byte{10})+"<pre>"+esc(out)+"</pre>", nodeCardKeyboard(id))
			}
		}
		return
	}

	if strings.HasPrefix(data, "m:lang:") {
		l := strings.TrimPrefix(data, "m:lang:")
		setLang(l)
		name := "Русский"
		if l == "en" {
			name = "English"
		}
		reply(token, chat, msgID, Tf("lang_set", name), mainKeyboard())
		return
	}

	if strings.HasPrefix(data, "m:nr:") {
		id := strings.TrimPrefix(data, "m:nr:")
		setState(chat, "wait_node_newname", id)
		host := id
		for _, r := range parseNodesList() {
			if r.ID == id {
				if r.Host != "" {
					host = r.Host
				}
				break
			}
		}
		prompt := fmt.Sprintf(T("nodes_pick"), esc(host))
		prompt += string([]byte{10}) + "id: <code>" + esc(id) + "</code>"
		reply(token, chat, msgID, prompt, backTo("nodes"))
		return
	}

	switch data {
	case "m:menu", "m:help":
		setState(chat, "", "")
		if data == "m:help" {
			reply(token, chat, msgID, helpText(), mainKeyboard())
		} else {
			reply(token, chat, msgID, menuText(), mainKeyboard())
		}
	case "m:fleet":
		reply(token, chat, msgID, T("fleet_title"), fleetKeyboard())
	case "m:operator":
		reply(token, chat, msgID, T("operator_title"), operatorKeyboard())
	case "m:cat:users":
		reply(token, chat, msgID, formatUsersListHTML(), usersListKeyboard())
	case "m:cat:vpn":
		setState(chat, "", "")
		reply(token, chat, msgID, formatUsersListHTML(), usersListKeyboard())
	case "m:cat:routers":
		setState(chat, "", "")
		reply(token, chat, msgID, T("cat_routers_title"), routersKeyboard())
	case "m:cat:sites":
		reply(token, chat, msgID, T("sites_title")+string([]byte{10, 10})+formatSitesHTML(), sitesKeyboard())
	case "m:sites:list":
		reply(token, chat, msgID, T("sites_title")+string([]byte{10, 10})+formatSitesHTML(), sitesKeyboard())
	case "m:sites:rsc":
		reply(token, chat, msgID, formatSitesRSCHTML(), sitesKeyboard())
	case "m:cat:nodes":
		setState(chat, "", "")
		reply(token, chat, msgID, T("nodes_title")+string([]byte{10, 10})+T("nodes_hint"), nodesKeyboard())
	case "m:nodes_list":
		reply(token, chat, msgID, T("nodes_title")+string([]byte{10, 10})+formatNodesListHTML()+string([]byte{10, 10})+"<i>"+T("nodes_hint")+"</i>", nodesListKeyboard())
	case "m:node_rename":
		setState(chat, "", "")
		reply(token, chat, msgID, T("nodes_rename")+string([]byte{10, 10})+formatNodesListHTML(), nodesRenameKeyboard())
	case "m:lang":
		cur := getLang()
		label := "Русский"
		if cur == "en" {
			label = "English"
		}
		reply(token, chat, msgID, Tf("lang_now", label), langKeyboard())
	case "m:routers":
		reply(token, chat, msgID, formatEdgeListHTML(routersText()), backTo("routers"))
	case "m:pending":
		t := pendingText()
		reply(token, chat, msgID, formatPendingHTML(t), pendingKeyboard(t))
	case "m:templates":
		reply(token, chat, msgID, formatTemplatesHTML(templatesText()), backTo("routers"))
	case "m:edge_recovery":
		out := runND("edge", "recovery")
		lines := strings.Split(strings.TrimSpace(out), "\n")
		code := ""
		if len(lines) > 0 {
			code = strings.TrimSpace(lines[0])
		}
		msg := T("edge_recovery_done") + "\n\n<code>" + esc(code) + "</code>"
		if len(lines) > 1 {
			msg += "\n<pre>" + esc(strings.Join(lines[1:], "\n")) + "</pre>"
		}
		reply(token, chat, msgID, msg, routersKeyboard())
	case "m:edge_register":
		setState(chat, "wait_edge_register", "")
		reply(token, chat, msgID, T("edge_register_prompt"), backTo("routers"))
	case "m:edge_set_site":
		setState(chat, "wait_edge_set_site", "")
		reply(token, chat, msgID, T("edge_set_site_prompt"), backTo("routers"))
	case "m:edge_apply":
		setState(chat, "wait_edge_apply", "")
		reply(token, chat, msgID, T("apply_prompt"), backTo("routers"))
	case "m:edge_bind":
		setState(chat, "wait_edge_bind_dev", "")
		reply(token, chat, msgID, T("bind_prompt"), backTo("routers"))
		case "m:cat:relay":
		// Relay is part of Nodes
		reply(token, chat, msgID, T("nodes_title")+string([]byte{10, 10})+formatNodesListHTML()+string([]byte{10, 10})+"<i>secondary = RU node role</i>", nodesKeyboard())
	case "m:relay:export", "m:secondary:export":
		out := runND("secondary", "export", "-o", "/tmp/nd-secondary-bundle.json", "--sni", "ya.ru")
		b, err := os.ReadFile("/tmp/nd-secondary-bundle.json")
		msg := out
		if err == nil {
			msg = string(b)
		}
		reply(token, chat, msgID, "📦 <b>bundle</b>"+string([]byte{10})+"<pre>"+esc(truncate(msg, 3500))+"</pre>", relayKeyboard())
	case "m:relay:enroll", "m:secondary:enroll":
		setState(chat, "wait_relay_host", "")
		reply(token, chat, msgID, T("enroll_title")+string([]byte{10,10})+T("enroll_ip"), backTo("nodes"))
	case "m:relay:oneline", "m:secondary:oneline":
		reply(token, chat, msgID, formatRelayOneline(), relayKeyboard())
	case "m:relay:sync", "m:secondary:sync":
		out := runND("secondary", "sync")
		reply(token, chat, msgID, "🔄 <b>Sync</b>"+string([]byte{10})+"<pre>"+esc(out)+"</pre>"+string([]byte{10})+formatRelayListHTML(), relayKeyboard())
	case "m:relay:exit:menu", "m:secondary:exit:menu":
		reply(token, chat, msgID, T("ru_exit_help"), relayKeyboard())
	case "m:relay:exit:on", "m:secondary:exit:on":
		out := runND("secondary", "exit", "on")
		reply(token, chat, msgID, "🇷🇺 <b>RU exit ON</b>"+string([]byte{10})+"<pre>"+esc(out)+"</pre>"+string([]byte{10})+"<i>Трафик с core уходит через РФ (доступ к RU-сервисам из-за границы)</i>", relayKeyboard())
	case "m:relay:exit:off", "m:secondary:exit:off":
		out := runND("secondary", "exit", "off")
		reply(token, chat, msgID, "✈️ <b>RU exit OFF</b>"+string([]byte{10})+"<pre>"+esc(out)+"</pre>", relayKeyboard())
	case "m:relay:list", "m:secondary:list":
		reply(token, chat, msgID, formatRelayListHTML(), relayKeyboard())
	case "m:addons":
		editHTML(token, cq.Message.Chat.ID, cq.Message.MessageID, formatAddonsHTML(), addonsKeyboard())
	case "m:addon:lampac":
		editHTML(token, cq.Message.Chat.ID, cq.Message.MessageID, formatLampacHTML(), addonsKeyboard())
	case "m:status":
		reply(token, chat, msgID, formatStatusPretty(), backKeyboard())
	case "m:users", "m:vpn_list":
		reply(token, chat, msgID, formatUsersListHTML(), usersListKeyboard())
	case "m:vpn_list_legacy":
		reply(token, chat, msgID, formatVPNListPretty(runVPN("list")), vpnUsersKeyboard())
	case "m:vpn_add":
		setState(chat, "wait_vpn_add_name", "")
		reply(token, chat, msgID, T("add_prompt"), backTo("vpn"))
	case "m:vpn_refresh":
		out := runND("vpn", "refresh-links")
		reply(token, chat, msgID, "🔄 <pre>"+esc(out)+"</pre>", vpnKeyboard())
	case "m:audit":
		out := runND("audit", "tail")
		if strings.TrimSpace(out) == "" {
			out = "(empty)"
		}
		reply(token, chat, msgID, "📋 <b>Audit</b>\n<pre>"+esc(truncate(out, 3500))+"</pre>", mainKeyboard())
	case "m:sessions":
		out := runND("vpn", "session", "list")
		if strings.TrimSpace(out) == "" {
			out = "(none)"
		}
		reply(token, chat, msgID, "🔑 <b>Sessions</b>\n<pre>"+esc(out)+"</pre>",
			map[string]any{"inline_keyboard": [][]map[string]any{
				{btn("🗑 Revoke all", "m:sessions:revoke", "danger")},
				{btn(T("main_menu"), "m:menu", "")},
			}})
	case "m:sessions:revoke":
		_ = runND("vpn", "session", "revoke-all")
		reply(token, chat, msgID, "✅ revoked all sessions", mainKeyboard())
	case "m:vpn_rename":
		setState(chat, "wait_vpn_rename", "")
		reply(token, chat, msgID, T("vpn_rename_hint"), backKeyboard())
	case "m:vpn_sub":
		reply(token, chat, msgID, "ℹ️ Subscription removed. Use Access → VLESS / Core / HY2.", usersListKeyboard())
	case "m:vpn_link", "m:vpn_disable", "m:vpn_enable", "m:vpn_revoke":
		action := strings.TrimPrefix(data, "m:")
		// map m:vpn_link -> link
		act := strings.TrimPrefix(action, "vpn_")
		title := map[string]string{
			"link": "🔗 <b>Ссылка / QR</b> — выберите пользователя:",
			"disable": "🚫 <b>Disable</b> — выберите пользователя:",
			"enable": "✅ <b>Enable</b> — выберите пользователя:",
			"revoke": "🗑 <b>Revoke</b> — выберите пользователя:",
		}[act]
		if getLang() == "en" {
			title = map[string]string{
				"link": "🔗 <b>Link / QR</b> — pick a user:",
				"disable": "🚫 <b>Disable</b> — pick a user:",
				"enable": "✅ <b>Enable</b> — pick a user:",
				"revoke": "🗑 <b>Revoke</b> — pick a user:",
			}[act]
		}
		reply(token, chat, msgID, title+string([]byte{10, 10})+formatVPNListPretty(runVPN("list")), vpnUsersKeyboardFor(act))
	case "m:backup":
		st := runND("backup", "peer-status")
		reply(token, chat, msgID, "💾 <b>"+esc(T("backup"))+"</b>"+string([]byte{10})+"<i>"+T("backup_hint")+"</i>"+string([]byte{10, 10})+"<pre>"+esc(st)+"</pre>",
			map[string]any{"inline_keyboard": [][]map[string]any{
				{btn(T("backup_run"), "m:backup:run", "primary"), btn(T("backup_set"), "m:backup:set", "")},
				{btn(T("back"), "m:cat:nodes", "primary"), btn(T("main_menu"), "m:menu", "")},
			}})
	case "m:backup:run":
		out := runND("backup")
		reply(token, chat, msgID, "💾 <pre>"+esc(truncate(out, 2000))+"</pre>", backTo("nodes"))
	case "m:backup:set":
		setState(chat, "wait_backup_peer", "")
		reply(token, chat, msgID, "root@HOST:/var/lib/netductor/backups/peers/core/", backKeyboard())
	case "m:sshhosts":
		reply(token, chat, msgID, formatSSHHostsHTML(), sshHostsKeyboard())
	case "m:ssh:clear:mt":
		_ = runND("ssh-hosts", "clear", "--kind", "mt")
		reply(token, chat, msgID, T("ssh_cleared")+" (mt)"+string([]byte{10, 10})+formatSSHHostsHTML(), sshHostsKeyboard())
	case "m:ssh:clear:relay":
		_ = runND("ssh-hosts", "clear", "--kind", "secondary")
		reply(token, chat, msgID, T("ssh_cleared")+" (relay)"+string([]byte{10, 10})+formatSSHHostsHTML(), sshHostsKeyboard())
	case "m:ssh:forget":
		setState(chat, "wait_ssh_forget", "")
		reply(token, chat, msgID, T("ssh_forget")+string([]byte{10})+"host or host:port", backKeyboard())
	case "m:admin":
		reply(token, chat, msgID, T("admin_body"), backKeyboard())
	case "m:session":
		setState(chat, "wait_session_hours", "")
		reply(token, chat, msgID, T("session_prompt"), backKeyboard())
	default:
		reply(token, chat, msgID, T("unknown"), mainKeyboard())
	}
}

