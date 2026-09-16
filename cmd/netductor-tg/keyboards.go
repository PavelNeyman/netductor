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
	// Flat top: 4 clear areas. No VPN-tools duplicate.
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn(T("status"), "m:status", "primary")},
			{btn(T("users"), "m:users", "primary")},
			{btn(T("fleet"), "m:fleet", "primary")},
			{btn(T("operator"), "m:operator", "")},
			{btn(T("lang"), "m:lang", ""), btn(T("help"), "m:help", "")},
		},
	}
}

// fleet = nodes + routers + sites
func fleetKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("nodes"), "m:cat:nodes", "primary")},
		{btn(T("cat_routers"), "m:cat:routers", "primary")},
		{btn(T("sites"), "m:cat:sites", "")},
		{btn(T("addons"), "m:addons", "")},
		{btn(T("main_menu"), "m:menu", "")},
	}}
}

// operator = session / admin / audit (not day-to-day user VPN)
func operatorKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("session"), "m:session", "primary"), btn(T("admin"), "m:admin", "")},
		{btn(T("sessions"), "m:sessions", ""), btn(T("audit"), "m:audit", "")},
		{btn(T("refresh_links"), "m:vpn_refresh", "")},
		{btn(T("main_menu"), "m:menu", "")},
	}}
}

func vpnKeyboard() map[string]any {
	// legacy alias → users
	return usersListKeyboard()
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
	_ = id
	// Navigation only; actions are <tg-button> in formatNodeCardHTML.
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("nodes"), "m:cat:nodes", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
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

// User hub: navigation only under the message. Actions live in formatUserHubHTML <tg-button>.
func userHubKeyboard(name string) map[string]any {
	_ = name
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("users"), "m:users", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}

// mode: vless | core | hy2 — navigation under message; mode switch is in access HTML body.
func userAccessKeyboard(name, mode string) map[string]any {
	style := func(want string) string {
		if mode == want || (want == "vless" && (mode == "" || mode == "vless")) {
			return "primary"
		}
		return ""
	}
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{
			btn("VLESS", "u:access:"+name+":vless", style("vless")),
			btn("Core", "u:access:"+name+":core", style("core")),
			btn("HY2", "u:access:"+name+":hy2", style("hy2")),
		},
		{btn(T("user_card"), "u:open:"+name, ""), btn(T("users"), "m:users", "")},
		{btn(T("main_menu"), "m:menu", "")},
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
	// Navigation only. Add user + per-user actions are <tg-button> in formatUsersListHTML.
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
}
