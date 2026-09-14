package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func btn(text, data, style string) map[string]any {
	b := map[string]any{"text": text, "callback_data": data}
	if style != "" {
		b["style"] = style
	}
	return b
}

func btnCopy(text, copyPayload string) map[string]any {
	return map[string]any{
		"text":      text,
		"copy_text": map[string]string{"text": copyPayload},
	}
}

func relayKeyboard() map[string]any {
	// nested under Nodes — same actions, back goes to nodes
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("nodes_enroll"), "m:relay:enroll", "primary")},
		{btn(T("relay_exit_on"), "m:relay:exit:on", "success"), btn(T("relay_exit_off"), "m:relay:exit:off", "danger")},
		{btn(T("relay_sync"), "m:relay:sync", ""), btn(T("nodes"), "m:cat:nodes", "primary")},
		{btn(T("main_menu"), "m:menu", "")},
	}}
}

func mainKeyboard() map[string]any {
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn(T("status"), "m:status", "primary")},
			{btn(T("users"), "m:users", "primary"), btn(T("cat_routers"), "m:cat:routers", "primary")},
			{btn(T("nodes"), "m:cat:nodes", "primary"), btn(T("sites"), "m:cat:sites", "")},
			{btn(T("session"), "m:session", ""), btn(T("admin"), "m:admin", "")},
			{btn(T("addons"), "m:addons", ""), btn(T("vpn_tools"), "m:cat:vpn", "")},
			{btn(T("audit"), "m:audit", ""), btn(T("sessions"), "m:sessions", "")},
			{btn(T("lang"), "m:lang", ""), btn(T("help"), "m:help", "")},
		},
	}
}

func vpnKeyboard() map[string]any {
	// system tools only — per-user actions live on user card
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn(T("users"), "m:users", "primary"), btn(T("vpn_add"), "m:vpn_add", "success")},
			{btn(T("refresh_links"), "m:vpn_refresh", "")},
			{btn(T("main_menu"), "m:menu", "")},
		},
	}
}

func addonsKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("Lampac", "m:addon:lampac", "primary")},
		{btn(T("back"), "m:menu", "")},
	}}
}

func sitesKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("sites_list"), "m:sites:list", "primary")},
		{btn(T("sites_rsc"), "m:sites:rsc", "")},
		{btn(T("main_menu"), "m:menu", "")},
	}}
}

func nodesKeyboard() map[string]any {
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn(T("nodes_list_btn"), "m:nodes_list", "primary"), btn(T("nodes_rename_btn"), "m:node_rename", "")},
			{btn(T("nodes_enroll"), "m:relay:enroll", "primary")},
			{btn(T("nodes_sync"), "m:relay:sync", ""), btn(T("nodes_exit"), "m:relay:exit:menu", "primary")},
			{btn(T("ssh_hosts"), "m:sshhosts", "primary")},
			{btn(T("backup"), "m:backup", "")},
			{btn(T("main_menu"), "m:menu", "")},
		},
	}
}

type nodeRow struct {
	ID, Host, Role, Kind, IP, Status, Desired string
}

func parseNodesList() []nodeRow {
	out, err := exec.Command(netductorBin(), "nodes", "list").CombinedOutput()
	if err != nil {
		return nil
	}
	var rows []nodeRow
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "(") {
			continue
		}
		// id\thost=...\trole=...
		parts := strings.Split(line, "	")
		if len(parts) == 0 {
			continue
		}
		r := nodeRow{}
		for i, p := range parts {
			switch {
			case strings.HasPrefix(p, "id="):
				r.ID = strings.TrimPrefix(p, "id=")
			case strings.HasPrefix(p, "host="):
				r.Host = strings.TrimPrefix(p, "host=")
			case strings.HasPrefix(p, "role="):
				r.Role = strings.TrimPrefix(p, "role=")
			case strings.HasPrefix(p, "kind="):
				r.Kind = strings.TrimPrefix(p, "kind=")
			case strings.HasPrefix(p, "ip="):
				r.IP = strings.TrimPrefix(p, "ip=")
			case strings.HasPrefix(p, "status="):
				r.Status = strings.TrimPrefix(p, "status=")
			case strings.HasPrefix(p, "desired="):
				r.Desired = strings.TrimPrefix(p, "desired=")
			case i == 0 && !strings.Contains(p, "="):
				r.ID = p // legacy
			}
		}
		if r.Host == "" {
			r.Host = r.ID
		}
		rows = append(rows, r)
	}
	return rows
}


func formatNodesListHTML() string {
	rows := parseNodesList()
	if len(rows) == 0 {
		return T("nodes_empty")
	}
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>#</th><th>host</th><th>role</th><th>ip</th><th>status</th></tr>" + nl)
	for i, r := range rows {
		label := r.Host
		if label == "" {
			label = r.ID
		}
		icon := "•"
		st := strings.ToLower(r.Status)
		switch {
		case st == "online":
			icon = "🟢"
		case st == "offline":
			icon = "🔴"
		}
		ip := r.IP
		if ip == "" {
			ip = "—"
		}
		role := r.Role
		if role == "" {
			role = "—"
		}
		status := icon + " " + r.Status
		if r.Desired != "" {
			status += " → " + r.Desired
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>"+nl,
			i+1, esc(label), esc(role), esc(ip), esc(status)))
	}
	b.WriteString("</table>")
	return b.String()
}

func nodeCardKeyboard(id string) map[string]any {
	pending := strings.Contains(runND("relay", "device", id), "pending:")
	rows := [][]map[string]any{
		{btn(T("node_metrics"), "m:nd:m:"+id, "primary")},
		{btn(T("node_journal"), "m:nd:j:"+id, ""), btn(T("node_restart_sb"), "m:nd:s:"+id, "")},
	}
	if !pending {
		rows = append(rows, []map[string]any{
			btn(T("node_upgrade"), "m:nd:u:"+id, ""),
			btn(T("node_reboot"), "m:nd:r:"+id, "danger"),
		})
	} else {
		rows = append(rows, []map[string]any{btn("⏳ …", "m:nd:m:"+id, "")})
	}
	rows = append(rows, []map[string]any{btn(T("nodes"), "m:cat:nodes", "primary"), btn(T("main_menu"), "m:menu", "")})
	return map[string]any{"inline_keyboard": rows}
}

func nodesListKeyboard() map[string]any {
	rows := parseNodesList()
	kb := [][]map[string]any{}
	for i, r := range rows {
		name := r.Host
		if name == "" {
			name = r.ID
		}
		label := fmt.Sprintf("%d. %s", i+1, name)
		if r.Role != "" {
			label += " [" + r.Role + "]"
		}
		if len(label) > 40 {
			label = label[:40]
		}
		data := "m:nd:o:" + r.ID
		if len(data) > 64 {
			data = data[:64]
		}
		kb = append(kb, []map[string]any{btn(label, data, "")})
	}
	kb = append(kb, []map[string]any{btn(T("nodes_enroll"), "m:relay:enroll", "primary")})
	kb = append(kb, []map[string]any{btn(T("main_menu"), "m:menu", "")})
	return map[string]any{"inline_keyboard": kb}
}

func nodesRenameKeyboard() map[string]any {
	rows := parseNodesList()
	kb := [][]map[string]any{}
	for i, r := range rows {
		name := r.Host
		if name == "" {
			name = r.ID
		}
		label := fmt.Sprintf("%d. %s", i+1, name)
		if len(label) > 40 {
			label = label[:40]
		}
		// callback max 64 bytes
		data := "m:nr:" + r.ID
		if len(data) > 64 {
			data = data[:64]
		}
		kb = append(kb, []map[string]any{btn(label, data, "")})
	}
	kb = append(kb, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
	return map[string]any{"inline_keyboard": kb}
}


func routersKeyboard() map[string]any {
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn(T("devices"), "m:routers", "primary"), btn(T("pending"), "m:pending", "primary")},
			{btn(T("templates"), "m:templates", ""), btn(T("bind_tmpl"), "m:edge_bind", "")},
			{btn(T("apply_tmpl"), "m:edge_apply", "primary")},
			{btn(T("main_menu"), "m:menu", "")},
		},
	}
}

func backKeyboard() map[string]any {
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn(T("main_menu"), "m:menu", "primary")},
		},
	}
}

func backTo(cat string) map[string]any {
	up := "m:menu"
	label := T("main_menu")
	switch cat {
	case "users":
		up, label = "m:users", T("users")
	case "vpn":
		up, label = "m:cat:vpn", T("back_vpn")
	case "routers":
		up, label = "m:cat:routers", T("back_routers")
	case "relay", "nodes":
		up, label = "m:cat:nodes", T("nodes")
	}
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn(label, up, "primary"), btn(T("main_menu"), "m:menu", "")},
		},
	}
}

func langKeyboard() map[string]any {
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn(T("lang_ru"), "m:lang:ru", "primary"), btn(T("lang_en"), "m:lang:en", "")},
			{btn(T("main_menu"), "m:menu", "")},
		},
	}
}

func userCardKeyboard(name, subText string) map[string]any {
	return userHubKeyboard(name)
}

// User hub: all actions for one identity
func userHubKeyboard(name string) map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("user_access"), "u:access:"+name+":vless", "primary")},
		{btn(T("vpn_rename"), "u:rename:"+name, "")},
		{btn(T("vpn_enable"), "u:enable:"+name, "success"), btn(T("vpn_disable"), "u:disable:"+name, "danger")},
		{btn(T("vpn_revoke"), "u:revoke:"+name, "danger")},
		{btn(T("users"), "m:users", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}

// mode: vless | core | hy2 | sub
func userAccessKeyboard(name, mode string) map[string]any {
	if mode == "" {
		mode = "vless"
	}
	label := map[string]string{
		"vless": "VLESS · primary",
		"core":  "VLESS · core",
		"hy2":   "HY2 · optional",
		"sub":   "Subscription",
	}
	cur := label[mode]
	if cur == "" {
		cur = mode
	}
	// cycle order
	order := []string{"vless", "core", "hy2", "sub"}
	idx := 0
	for i, m := range order {
		if m == mode {
			idx = i
			break
		}
	}
	prev := order[(idx+len(order)-1)%len(order)]
	next := order[(idx+1)%len(order)]
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("◀", "u:access:"+name+":"+prev, ""), btn("📱 "+cur, "u:access:"+name+":"+mode, "primary"), btn("▶", "u:access:"+name+":"+next, "")},
		{btn("VLESS", "u:access:"+name+":vless", ""), btn("Core", "u:access:"+name+":core", ""), btn("HY2", "u:access:"+name+":hy2", ""), btn("Sub", "u:access:"+name+":sub", "success")},
		{btn(T("user_card"), "u:open:"+name, "primary"), btn(T("users"), "m:users", "")},
	}}
}

// legacy alias
func userCardKeyboardMode(name, mode string) map[string]any {
	return userAccessKeyboard(name, mode)
}

func vpnUsersKeyboard() map[string]any {
	return usersListKeyboard()
}

func vpnUsersKeyboardFor(action string) map[string]any {
	return usersListKeyboard()
}

func usersListKeyboard() map[string]any {
	raw := runVPN("list")
	rows := [][]map[string]any{}
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 1 {
			parts = strings.Fields(line)
		}
		if len(parts) < 1 || parts[0] == "" {
			continue
		}
		name := parts[0]
		if name == "relay-uplink" {
			continue
		}
		en := ""
		if len(parts) > 1 {
			en = parts[1]
		}
		n++
		icon := "🟢"
		if en == "off" {
			icon = "🔴"
		}
		label := icon + " " + fmt.Sprintf("%d. %s", n, name)
		if en != "" {
			label += " · " + en
		}
		rows = append(rows, []map[string]any{btn(label, "u:open:"+name, "primary")})
		if n >= 30 {
			break
		}
	}
	if n == 0 {
		rows = append(rows, []map[string]any{btn("— empty —", "m:users", "")})
	}
	rows = append(rows, []map[string]any{btn(T("vpn_add"), "m:vpn_add", "success")})
	rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary"), btn(T("vpn_tools"), "m:cat:vpn", "")})
	return map[string]any{"inline_keyboard": rows}
}
