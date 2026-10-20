package main

import (
	"fmt"
	"strings"
)

func handleCBNodes(token string, chat int64, msgID int, data string) bool {
	if strings.HasPrefix(data, "m:ssh:rm:") {
		id := strings.TrimPrefix(data, "m:ssh:rm:")
		out := runND("ssh-hosts", "forget", id)
		reply(token, chat, msgID, fmt.Sprintf(T("ssh_forgot"), esc(id))+string([]byte{10})+"<pre>"+esc(out)+"</pre>"+string([]byte{10, 10})+formatSSHHostsHTML(), sshHostsKeyboard())
		return true
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
			case "r": // reboot — challenge code
				handleRebootCB(token, chat, msgID, "m:n:reboot:"+id)
			case "j": // journal
				reply(token, chat, msgID, formatJournalHTML(id), nodeCardKeyboard(id))
			case "s": // restart sing-box
				out := restartSingBox(id)
				reply(token, chat, msgID, "♻️ <b>Restart sing-box</b>"+string([]byte{10})+"<pre>"+esc(out)+"</pre>", nodeCardKeyboard(id))
			}
		}
		return true
	}

	if strings.HasPrefix(data, "m:lang:") {
		l := strings.TrimPrefix(data, "m:lang:")
		setLang(l)
		name := "Русский"
		if l == "en" {
			name = "English"
		}
		reply(token, chat, msgID, Tf("lang_set", name), mainKeyboard())
		return true
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
		return true
	}
	return false
}
