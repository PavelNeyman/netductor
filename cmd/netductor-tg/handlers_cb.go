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

	if strings.HasPrefix(data, "u:") {
		parts := strings.SplitN(data, ":", 3)
		if len(parts) == 3 {
			action, name := parts[1], parts[2]
			switch action {
			case "link":
				deliverVPNLink(token, chat, msgID, name)
			case "hy2qr":
				showVPNQR(token, chat, msgID, name, "hy2", true)
			case "vlessqr":
				showVPNQR(token, chat, msgID, name, "vless", true)
			case "enable":
				reply(token, chat, msgID, "✅ <pre>"+esc(runVPN("enable", name))+"</pre>", backKeyboard())
			case "disable":
				reply(token, chat, msgID, "🚫 <pre>"+esc(runVPN("disable", name))+"</pre>", backKeyboard())
			case "revoke":
				reply(token, chat, msgID, "🗑 <pre>"+esc(runVPN("revoke", name))+"</pre>", backKeyboard())
			}
		}
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
	case "m:cat:vpn":
		setState(chat, "", "")
		reply(token, chat, msgID, T("cat_vpn_title"), vpnKeyboard())
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
	case "m:edge_apply":
		setState(chat, "wait_edge_apply", "")
		reply(token, chat, msgID, T("apply_prompt"), backTo("routers"))
	case "m:edge_bind":
		setState(chat, "wait_edge_bind_dev", "")
		reply(token, chat, msgID, T("bind_prompt"), backTo("routers"))
		case "m:cat:relay":
		// Relay is part of Nodes
		reply(token, chat, msgID, T("nodes_title")+string([]byte{10, 10})+formatNodesListHTML()+string([]byte{10, 10})+"<i>relay = role in nodes</i>", nodesKeyboard())
	case "m:relay:export":
		out := runND("relay", "export", "-o", "/tmp/nd-relay-bundle.json", "--sni", "ya.ru")
		b, err := os.ReadFile("/tmp/nd-relay-bundle.json")
		msg := out
		if err == nil {
			msg = string(b)
		}
		reply(token, chat, msgID, "📦 <b>bundle</b>"+string([]byte{10})+"<pre>"+esc(truncate(msg, 3500))+"</pre>", relayKeyboard())
	case "m:relay:enroll":
		setState(chat, "wait_relay_host", "")
		reply(token, chat, msgID, T("enroll_title")+string([]byte{10,10})+T("enroll_ip"), backTo("nodes"))
	case "m:relay:oneline":
		reply(token, chat, msgID, formatRelayOneline(), relayKeyboard())
	case "m:relay:sync":
		out := runND("relay", "sync")
		reply(token, chat, msgID, "🔄 <b>Sync</b>"+string([]byte{10})+"<pre>"+esc(out)+"</pre>"+string([]byte{10})+formatRelayListHTML(), relayKeyboard())
	case "m:relay:exit:menu":
		reply(token, chat, msgID, T("ru_exit_help"), relayKeyboard())
	case "m:relay:exit:on":
		out := runND("relay", "exit", "on")
		reply(token, chat, msgID, "🇷🇺 <b>RU exit ON</b>"+string([]byte{10})+"<pre>"+esc(out)+"</pre>"+string([]byte{10})+"<i>Трафик с core уходит через РФ (доступ к RU-сервисам из-за границы)</i>", relayKeyboard())
	case "m:relay:exit:off":
		out := runND("relay", "exit", "off")
		reply(token, chat, msgID, "✈️ <b>RU exit OFF</b>"+string([]byte{10})+"<pre>"+esc(out)+"</pre>", relayKeyboard())
	case "m:relay:list":
		reply(token, chat, msgID, formatRelayListHTML(), relayKeyboard())
	case "m:addons":
		editHTML(token, cq.Message.Chat.ID, cq.Message.MessageID, formatAddonsHTML(), addonsKeyboard())
	case "m:addon:lampac":
		editHTML(token, cq.Message.Chat.ID, cq.Message.MessageID, formatLampacHTML(), addonsKeyboard())
	case "m:status":
		reply(token, chat, msgID, formatStatusPretty(), backKeyboard())
	case "m:vpn_list":
		reply(token, chat, msgID, formatVPNListPretty(runVPN("list")), vpnUsersKeyboard())
	case "m:vpn_add":
		setState(chat, "wait_vpn_add_name", "")
		reply(token, chat, msgID, T("add_prompt"), backTo("vpn"))
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
	case "m:admin":
		reply(token, chat, msgID, T("admin_body"), backKeyboard())
	case "m:session":
		setState(chat, "wait_session_hours", "")
		reply(token, chat, msgID, T("session_prompt"), backKeyboard())
	default:
		reply(token, chat, msgID, T("unknown"), mainKeyboard())
	}
}

