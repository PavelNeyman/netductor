package main

import (
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/format"
)

func handleCBTools(token string, chat int64, msgID int, data string) bool {
	if data == "m:mtls" || strings.HasPrefix(data, "m:mtls:") {
		handleMtlsCB(token, chat, msgID, data)
		return true
	}

	if strings.HasPrefix(data, "m:alerts-chat:pick:") {
		id := strings.TrimPrefix(data, "m:alerts-chat:pick:")
		applyAlertsChatID(token, chat, msgID, id)
		return true
	}
	if data == "m:cleanup" {
		out, _ := exec.Command("netductor", "cleanup", "--json").CombinedOutput()
		reply(token, chat, msgID, "<pre>"+esc(string(out))+"</pre>\n<i>Weekly timer: Sun 04:30 UTC (docker prune + leftovers)</i>\n<code>netductor cleanup --apply</code>", backKeyboard())
		return true
	}
	if data == "m:cleanup:apply" {
		out, _ := exec.Command("netductor", "cleanup", "--apply").CombinedOutput()
		reply(token, chat, msgID, "<pre>"+esc(string(out))+"</pre>", backKeyboard())
		return true
	}
	if data == "m:fw" || data == "m:firewall" {
		body := formatFirewallBlock()
		reply(token, chat, msgID, body, backKeyboard())
		return true
	}
	if data == "m:fw:apply" {
		out, err := exec.Command("netductor", "firewall", "apply").CombinedOutput()
		msg := string(out)
		if err != nil {
			msg = "❌ " + msg + " " + err.Error()
		} else {
			msg = "✅ " + msg
		}
		reply(token, chat, msgID, esc(msg)+"\n\n"+formatFirewallBlock(), backKeyboard())
		return true
	}
	if data == "m:stack" || strings.HasPrefix(data, "m:stack:") {
		handleStackCB(token, chat, msgID, data)
		return true
	}
	if data == "m:tools" {
		reply(token, chat, msgID, toolsHubHTML(), toolsKeyboard())
		return true
	}
	if strings.HasPrefix(data, "m:cat:") {
		handleCatGroup(token, chat, msgID, strings.TrimPrefix(data, "m:cat:"))
		return true
	}
	if strings.HasPrefix(data, "m:ops:") {
		sec := strings.TrimPrefix(data, "m:ops:")
		backTools := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
		}}
		switch sec {
		case "overview", "home":
			// PLAN-TG-MENU: Status is top-level, not under Tools.
			reply(token, chat, msgID, formatStatusPretty(), statusKeyboard())
			return true
		case "vpn", "users":
			reply(token, chat, msgID, formatUsersListHTML(), usersListKeyboard())
			return true
		case "nodes", "fleet":
			if sec == "fleet" {
				reply(token, chat, msgID, fleetHubHTML(), fleetKeyboard())
			} else {
				reply(token, chat, msgID, nodesHubHTML()+"\n<i>"+T("nodes_hint")+"</i>", nodesKeyboard())
			}
			return true
		case "edge", "routers":
			reply(token, chat, msgID, routersHubHTML(), routersKeyboard())
			return true
		case "nvr", "media":
			reply(token, chat, msgID, nvrHubHTML(), nvrKeyboard())
			return true
		case "dns":
			showDNSMenu(token, chat, msgID, "")
			return true
		case "backup":
			handleBackupCB(token, chat, msgID, "m:backup")
			return true
		case "probes", "adv":
			out := runND("probe")
			r := format.API("probes", []byte(out), catalogLang())
			reply(token, chat, msgID, r.HTML, backTools)
			return true
		case "git":
			handleGitCB(token, chat, msgID, "m:git")
			return true
		case "updates":
			handleVersionsCB(token, chat, msgID, "m:versions")
			return true
		case "sites":
			reply(token, chat, msgID, formatSitesHTML(), sitesListKeyboard())
			return true
		}
		// fallback: catalog section with numbered API actions
		reply(token, chat, msgID, catalogSectionTitle(sec), catalogSectionKeyboard(sec))
		return true
	}
	if strings.HasPrefix(data, "m:opraw:") {
		id := strings.TrimPrefix(data, "m:opraw:")
		raw, err := execCatalogActionRaw(id)
		html := formatCatalogHTML(id, raw, true)
		if err != nil {
			html = "⚠️ " + esc(err.Error()) + "<br>" + html
		}
		replyCatalog(token, chat, msgID, html, catalogResultKeyboard(id))
		return true
	}
	if strings.HasPrefix(data, "m:op:") {
		id := strings.TrimPrefix(data, "m:op:")
		raw, err := execCatalogActionRaw(id)
		var html string
		if err != nil {
			html = "⚠️ <b>error</b><br><pre>" + esc(err.Error()) + "</pre>"
			if len(raw) > 0 {
				html += "<br>" + formatCatalogHTML(id, raw, true)
			}
		} else {
			html = formatCatalogHTML(id, raw, false)
		}
		replyCatalog(token, chat, msgID, html, catalogResultKeyboard(id))
		return true
	}
	return false
}
