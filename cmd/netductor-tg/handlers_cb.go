package main

import (
	"fmt"
	"os"
	"strings"

	ndstack "github.com/PavelNeyman/netductor/internal/stack"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/format"
	"github.com/PavelNeyman/netductor/internal/notify"
)

func handleCallback(token string, cq *callbackQuery, admin int64) {
	if cq == nil {
		return
	}
	data := strings.TrimSpace(cq.Data)
	// Channel alerts must never spawn operator menus in the channel.
	// Prefer deep-link via answerCallbackQuery url (opens bot DM). Ack at most once.
	if cq.Message != nil && cq.Message.Chat.Type == "channel" {
		payload := "menu"
		if data == "m:updates" || strings.HasPrefix(data, "m:updates") {
			payload = "updates"
		}
		link := notify.DeepLinkStart(payload)
		if link != "" {
			_, _ = apiPost(token, "answerCallbackQuery", map[string]any{
				"callback_query_id": cq.ID,
				"url":               link,
			})
		} else {
			answerCallbackText(token, cq.ID, "Open the bot in a private chat")
		}
		fmt.Fprintf(os.Stderr, "tg callback blocked in channel chat=%d data=%q\n", cq.Message.Chat.ID, data)
		return
	}
	// Always ack the spinner first (rich in-body buttons included).
	if strings.HasPrefix(data, "m:nd:u:") || strings.HasPrefix(data, "m:nd:r:") {
		answerCallbackText(token, cq.ID, "⏳ …")
	} else if data != "" {
		answerCallback(token, cq.ID)
	}
	if admin != 0 && cq.From.ID != admin {
		fmt.Fprintf(os.Stderr, "tg callback ignored: from=%d admin=%d data=%q\n", cq.From.ID, admin, data)
		return
	}
	chat := int64(0)
	msgID := 0
	if cq.Message != nil {
		chat = cq.Message.Chat.ID
		msgID = cq.Message.MessageID
	}
	if chat == 0 {
		// Some clients omit message on rich-button callbacks — fall back to admin chat.
		chat = admin
		if chat == 0 {
			fmt.Fprintf(os.Stderr, "tg callback: no chat for data=%q\n", data)
			return
		}
	}
	if data == "" {
		fmt.Fprintln(os.Stderr, "tg callback: empty data")
		return
	}
	fmt.Fprintf(os.Stderr, "tg callback: chat=%d msg=%d data=%q\n", chat, msgID, data)
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Fprintf(os.Stderr, "tg callback panic data=%q: %v\n", data, rec)
			reply(token, chat, msgID, "❌ internal error (see bot log)", backKeyboard())
		}
	}()

	if handleJobsMenu(token, chat, msgID, data) {
		return
	}
	if handleCBEdgeApprove(token, chat, msgID, data) {
		return
	}

	if handleCanaryCB(token, chat, msgID, data) {
		return
	}
	if handlePresetsCB(token, chat, msgID, data) {
		return
	}
	if handleDevicesCB(token, chat, msgID, data) {
		return
	}
	if handleIncidentCB(token, chat, msgID, data) {
		return
	}
	if handleCBTools(token, chat, msgID, data) {
		return
	}
	if handleCBHubs(token, chat, msgID, data) {
		return
	}
	if handleCBUsers(token, chat, msgID, data) {
		return
	}

	if handleCBNodes(token, chat, msgID, data) {
		return
	}

	if handleRebootCB(token, chat, msgID, data) {
		return
	}
	if handleSSHCB(token, chat, msgID, data) {
		return
	}
	switch data {
	case "m:noop":
		// Disabled / busy button — brief toast
		answerCallbackText(token, cq.ID, "⏳")
		return
	case "m:updates:unlock":
		_ = ndstack.ForceClearApply()
		msg := "✅ Apply lock cleared"
		if getLang() != "en" {
			msg = "✅ Lock обновления сброшен"
		}
		reply(token, chat, msgID, msg, navKeyboard("m:tools", parentTools()))
		return
	case "m:topics", "m:topics:reconcile", "m:topics:recreate", "m:topics:flat", "m:alerts-chat":
		// Legacy m:topics* same as alerts-chat (channel / flat DM).
		reply(token, chat, msgID, alertsChatPromptHTML(getLang() != "en"), alertsChatKeyboard())
	case "m:alerts-chat:set":
		setState(chat, "wait_alerts_chat", "")
		ru := getLang() != "en"
		hint := "Send the channel id (−100…) or <b>forward a post</b> from the channel."
		if ru {
			hint = "Пришлите id канала (−100…) или <b>перешлите пост</b> из канала."
		}
		reply(token, chat, msgID, "✏️ "+hint, alertsChatKeyboard())
	case "m:alerts-chat:clear":
		_ = notify.ClearAlertsChatID()
		reply(token, chat, msgID, alertsChatPromptHTML(getLang() != "en"), alertsChatKeyboard())
	case "m:alerts-chat:test":
		if err := notify.SendTestAlert("🔔 test alert routing"); err != nil {
			reply(token, chat, msgID, "❌ test send: "+esc(err.Error())+"\n\n"+alertsChatPromptHTML(getLang() != "en"), alertsChatKeyboard())
		} else {
			dest := notify.AlertsChatID()
			if dest == "" {
				dest = "admin DM"
			}
			reply(token, chat, msgID, "✅ test sent → <code>"+esc(dest)+"</code>\n\n"+alertsChatPromptHTML(getLang() != "en"), alertsChatKeyboard())
		}
	case "m:hub:reset":
		// Explicit singleton reset (user deleted hub "for me" / cleared chat).
		forceHub(token, chat, menuText(), mainKeyboard())
	case "m:digest":
		// Fleet digest card (not Updates). Parent = Status.
		reply(token, chat, msgID, formatFleetDigestHTML(), statusKeyboard())
	case "m:dr":
		reply(token, chat, msgID, formatDisasterHTML(), mainKeyboard())
	case "m:menu", "m:help":
		setState(chat, "", "")
		// Singleton hub in admin DM (edit-in-place).
		body := menuText()
		if data == "m:help" {
			body = helpText()
		}
		reply(token, chat, msgID, body, mainKeyboard())
	case "m:fleet":
		reply(token, chat, msgID, fleetHubHTML(), fleetKeyboard())
	case "m:operator":
		reply(token, chat, msgID, operatorHubHTML(), operatorKeyboard())
	case "m:cat:users":
		reply(token, chat, msgID, formatUsersListHTML(), usersListKeyboard())
	case "m:cat:vpn":
		setState(chat, "", "")
		reply(token, chat, msgID, formatUsersListHTML(), usersListKeyboard())
	case "m:cat:routers":
		setState(chat, "", "")
		reply(token, chat, msgID, routersHubHTML(), routersKeyboard())
	case "m:cat:sites":
		reply(token, chat, msgID, sitesHubHTML()+"\n"+formatSitesHTML(), sitesKeyboard())
	case "m:sites:list":
		reply(token, chat, msgID, T("sites_title")+string([]byte{10, 10})+formatSitesHTML(), sitesListKeyboard())
	case "m:sites:rsc":
		reply(token, chat, msgID, formatSitesRSCHTML(), sitesListKeyboard())
	case "m:cat:nodes":
		setState(chat, "", "")
		reply(token, chat, msgID, nodesHubHTML()+"\n<i>"+T("nodes_hint")+"</i>", nodesKeyboard())
	case "m:nodes_list":
		reply(token, chat, msgID, T("nodes_title")+string([]byte{10, 10})+formatNodesListHTML()+string([]byte{10, 10})+"<i>"+T("nodes_hint")+"</i>", nodesListKeyboard())
	case "m:node_rename":
		setState(chat, "", "")
		reply(token, chat, msgID, formatNodesRenameHTML(), nodesRenameKeyboard())
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
		reply(token, chat, msgID, msg, backTo("routers"))
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
		reply(token, chat, msgID, T("nodes_title")+string([]byte{10, 10})+formatNodesListHTML()+string([]byte{10, 10})+"<i>role=secondary = RU ingress node</i>", nodesKeyboard())
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
		reply(token, chat, msgID, T("enroll_title")+string([]byte{10, 10})+T("enroll_ip"), backTo("nodes"))
	case "m:relay:oneline", "m:secondary:oneline":
		reply(token, chat, msgID, formatRelayOneline(), relayKeyboard())
	case "m:relay:sync", "m:secondary:sync":
		out := runND("secondary", "sync")
		nl := string([]byte{10})
		title := "🔄 <b>Sync secondary</b>" + nl + "<i>Push profiles/config to RU node</i>" + nl
		if getLang() != "en" {
			title = "🔄 <b>Синхронизация secondary</b>" + nl + "<i>Отправка профилей/конфига на RU-ноду</i>" + nl
		}
		reply(token, chat, msgID, title+"<pre>"+esc(out)+"</pre>"+nl+formatRelayListHTML(), relayKeyboard())
	case "m:relay:exit:menu", "m:secondary:exit:menu":
		reply(token, chat, msgID, T("ru_exit_help"), relayKeyboard())
	case "m:relay:exit:on", "m:secondary:exit:on":
		out := runND("secondary", "exit", "on")
		reply(token, chat, msgID, func() string {
			if getLang() != "en" {
				return "🇷🇺 <b>RU-выход ВКЛ</b>" + string([]byte{10}) + "<pre>" + esc(out) + "</pre>" + string([]byte{10}) + "<i>Трафик с primary через РФ</i>" + string([]byte{10}) + formatRelayActionsHTML()
			}
			return "🇷🇺 <b>RU exit ON</b>" + string([]byte{10}) + "<pre>" + esc(out) + "</pre>" + string([]byte{10}) + "<i>Primary traffic exits via RU</i>" + string([]byte{10}) + formatRelayActionsHTML()
		}(), relayKeyboard())
	case "m:relay:exit:off", "m:secondary:exit:off":
		out := runND("secondary", "exit", "off")
		nl := string([]byte{10})
		off := "✈️ <b>RU exit OFF</b>" + nl + "<pre>" + esc(out) + "</pre>"
		if getLang() != "en" {
			off = "✈️ <b>RU-выход ВЫКЛ</b>" + nl + "<pre>" + esc(out) + "</pre>"
		}
		reply(token, chat, msgID, off, relayKeyboard())
	case "m:relay:list", "m:secondary:list":
		reply(token, chat, msgID, formatRelayListHTML(), relayKeyboard())
	case "m:probes":
		out := runND("probe")
		r := format.API("probes", []byte(out), catalogLang())
		reply(token, chat, msgID, r.HTML, map[string]any{"inline_keyboard": [][]map[string]any{{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")}}})
	case "m:metrics":
		// folded into main Status (no separate Metrics screen)
		reply(token, chat, msgID, formatStatusPretty(), backKeyboard())
	case "m:secondary:status":
		out := runND("secondary", "status")
		r := format.API("secondary", []byte(out), catalogLang())
		nl2 := string([]byte{10})
		title := "🖥 <b>Secondary</b>" + nl2
		if getLang() != "en" {
			title = "🖥 <b>Secondary · RU</b>" + nl2
		}
		body := title + r.HTML
		if strings.TrimSpace(r.HTML) == "" || (strings.Contains(r.HTML, "<pre>") && len(out) < 20) {
			body = title + "<pre>" + esc(truncate(out, 3500)) + "</pre>"
		}
		reply(token, chat, msgID, body, map[string]any{"inline_keyboard": [][]map[string]any{{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")}}})
	case "m:addons":
		reply(token, chat, msgID, formatAddonsHTML(), addonsKeyboard())
	case "m:addons:update":
		out := runND("addons", "update")
		reply(token, chat, msgID, formatAddonsHTML()+"\n<pre>"+esc(truncate(out, 1200))+"</pre>", addonsKeyboard())
	case "m:addon:lampac":
		pl := "Addons"
		if getLang() != "en" {
			pl = "Аддоны"
		}
		reply(token, chat, msgID, formatLampacHTML(), navKeyboard("m:addons", pl))
	case "m:actions":
		reply(token, chat, msgID, fleet.FormatActionsHTML(getLang() != "en"), statusKeyboard())
	case "m:status":
		reply(token, chat, msgID, formatStatusPretty(), statusKeyboard())
	case "m:netmap":
		reply(token, chat, msgID, formatNetmapHTML(), statusKeyboard())
	case "m:channel":
		reply(token, chat, msgID, formatChannelDetailHTML(), statusKeyboard())
	case "m:vpn_devices":
		reply(token, chat, msgID, formatDevicesListHTML(), usersListKeyboard())
	case "m:users", "m:vpn_list":
		reply(token, chat, msgID, formatUsersListHTML(), usersListKeyboard())
	case "m:vpn_list_legacy":
		reply(token, chat, msgID, formatVPNListPretty(runVPN("list")), vpnUsersKeyboard())
	case "m:vpn_add":
		setState(chat, "wait_vpn_add_name", "")
		reply(token, chat, msgID, T("add_prompt"), backTo("vpn"))
	case "m:vpn_refresh":
		out := runND("vpn", "refresh-links")
		reply(token, chat, msgID, "🔄 <pre>"+esc(out)+"</pre>", operatorSubKeyboard())
	case "m:audit":
		out := runND("audit", "tail")
		if strings.TrimSpace(out) == "" {
			out = "(empty)"
		}
		title := "📋 <b>Audit</b> (last 100; includes stack/SSH/reboot)\n"
		if getLang() != "en" {
			title = "📋 <b>Аудит</b> (последние 100; stack/SSH/reboot)\n"
		}
		reply(token, chat, msgID, title+"<pre>"+esc(truncate(out, 3500))+"</pre>", operatorSubKeyboard())
	case "m:sessions":
		out := runND("vpn", "session", "list")
		if strings.TrimSpace(out) == "" {
			out = "(none)"
		}
		ru := getLang() != "en"
		title := "🔑 <b>Sessions</b>\n<pre>" + esc(out) + "</pre>\n"
		rev := "🗑 Revoke all"
		if ru {
			title = "🔑 <b>Сессии</b>\n<pre>" + esc(out) + "</pre>\n"
			rev = "🗑 Отозвать все"
		}
		title += `<tg-button-row align="left"><tg-button type="callback_data" style="danger" data="m:sessions:revoke">` + rev + `</tg-button></tg-button-row>`
		reply(token, chat, msgID, title, operatorSubKeyboard())
	case "m:sessions:revoke":
		_ = runND("vpn", "session", "revoke-all")
		msg := "✅ revoked all sessions"
		if getLang() != "en" {
			msg = "✅ все сессии отозваны"
		}
		// re-show list pattern
		out := runND("vpn", "session", "list")
		if strings.TrimSpace(out) == "" {
			out = "(none)"
		}
		reply(token, chat, msgID, msg+"\n<pre>"+esc(out)+"</pre>", operatorSubKeyboard())
	case "m:vpn_rename":
		setState(chat, "wait_vpn_rename", "")
		reply(token, chat, msgID, T("vpn_rename_hint"), backKeyboard())
	case "m:vpn_sub":
		reply(token, chat, msgID, T("sub_removed"), usersListKeyboard())
	case "m:vpn_link", "m:vpn_disable", "m:vpn_enable", "m:vpn_revoke":
		action := strings.TrimPrefix(data, "m:")
		// map m:vpn_link -> link
		act := strings.TrimPrefix(action, "vpn_")
		title := map[string]string{
			"link":    "🔗 <b>Ссылка / QR</b> — выберите пользователя:",
			"disable": "🚫 <b>Disable</b> — выберите пользователя:",
			"enable":  "✅ <b>Enable</b> — выберите пользователя:",
			"revoke":  "🗑 <b>Revoke</b> — выберите пользователя:",
		}[act]
		if getLang() == "en" {
			title = map[string]string{
				"link":    "🔗 <b>Link / QR</b> — pick a user:",
				"disable": "🚫 <b>Disable</b> — pick a user:",
				"enable":  "✅ <b>Enable</b> — pick a user:",
				"revoke":  "🗑 <b>Revoke</b> — pick a user:",
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

	case "m:api_public_toggle":
		st := runND("api-public", "status")
		if strings.Contains(st, "armed=true") {
			_ = runND("api-public", "disarm")
		} else {
			_ = runND("api-public", "arm", "--ttl", "15m")
		}
		reply(token, chat, msgID, operatorHubHTML(), operatorKeyboard())
	case "m:users_sp_toggle":
		pol := runND("svc-paths", "failover", "status")
		on := strings.Contains(pol, `"users_to_sp_on_vless_down": true`) || strings.Contains(pol, `"users_to_sp_on_vless_down":true`)
		if on {
			_ = runND("svc-paths", "failover", "set-users-sp", "off")
		} else {
			_ = runND("svc-paths", "failover", "set-users-sp", "on")
		}
		reply(token, chat, msgID, operatorHubHTML(), operatorKeyboard())
	case "m:svc_paths":
		out := runND("svc-paths", "status")
		title := "🔗 <b>Service paths SP/PS</b>"
		if getLang() != "en" {
			title = "🔗 <b>Служебные каналы SP/PS</b>"
		}
		reply(token, chat, msgID, title+"\n<pre>"+esc(out)+"</pre>", operatorSubKeyboard())
	case "m:svc_paths_apply":
		out := runND("svc-paths", "apply")
		reply(token, chat, msgID, "svc-paths apply\n<pre>"+esc(out)+"</pre>", operatorSubKeyboard())

	case "m:admin":
		reply(token, chat, msgID, T("admin_body"), operatorSubKeyboard())
	case "m:session":
		setState(chat, "wait_session_hours", "")
		reply(token, chat, msgID, T("session_prompt"), operatorSubKeyboard())
	default:
		fmt.Fprintf(os.Stderr, "tg unknown callback data=%q\n", data)
		msg := T("unknown") + "\n<code>" + esc(data) + "</code>"
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("🧰 Tools", "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
		}})
	}
}
