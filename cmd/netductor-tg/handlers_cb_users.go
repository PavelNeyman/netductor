package main

import (
	"strings"

	"github.com/PavelNeyman/netductor/internal/vpn"
)

func handleCBUsers(token string, chat int64, msgID int, data string) bool {
	if strings.HasPrefix(data, "u:") {
		// u:open:name | u:access:name:mode | u:rename:name | u:enable:name | ...
		parts := strings.Split(data, ":")
		if len(parts) >= 3 {
			action, name := parts[1], parts[2]
			switch action {
			case "workcfg":
				// u:workcfg:name | u:workcfg:name:profile
				if len(parts) >= 4 && parts[3] != "" {
					sendSRProfileDocument(token, chat, parts[3])
				} else if showWorkProfileButton(name) {
					showSRProfilePicker(token, chat, msgID, name)
				} else {
					sendSRProfileDocument(token, chat, vpn.SRProfileFamily)
				}
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
			case "subrot":
				if _, err := vpn.IssueSubToken(name); err != nil {
					reply(token, chat, msgID, "❌ "+esc(err.Error()), userHubKeyboard(name))
					return true
				}
				showUserAccess(token, chat, msgID, name, "sub")
			case "access":
				mode := "vless"
				if len(parts) >= 4 && parts[3] != "" {
					mode = parts[3]
				}
				showUserAccess(token, chat, msgID, name, mode)
			case "link":
				showUserAccess(token, chat, msgID, name, "vless")
			case "vlessqr":
				showUserAccess(token, chat, msgID, name, "vless")
			case "subprof":
				next, err := vpn.CycleSubProfile(name)
				if err != nil {
					reply(token, chat, msgID, "❌ "+esc(err.Error()), userHubKeyboard(name))
					return true
				}
				msg := "✅ sub profile → <code>" + esc(next) + "</code>\n<i>subscription URL content replaced; client refreshes on next poll</i>"
				if getLang() != "en" {
					msg = "✅ профиль подписки → <code>" + esc(next) + "</code>\n<i>содержимое URL заменено; клиент подтянет при обновлении</i>"
				}
				reply(token, chat, msgID, msg+string([]byte{10, 10})+formatUserHubHTML(name), userHubKeyboard(name))
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
			case "kind":
				// u:kind:name:user|device
				k := "user"
				if len(parts) >= 4 && parts[3] != "" {
					k = parts[3]
				}
				if err := vpn.SetKind(name, k); err != nil {
					reply(token, chat, msgID, "❌ "+esc(err.Error()), userHubKeyboard(name))
					return true
				}
				msg := "✅ kind → <code>" + esc(k) + "</code>"
				if getLang() != "en" {
					msg = "✅ тип → <code>" + esc(k) + "</code>"
				}
				reply(token, chat, msgID, msg+string([]byte{10, 10})+formatUserHubHTML(name), userHubKeyboard(name))
			}
		}
		return true
	}
	return false
}
