package main

import (
	"fmt"
	"net/url"
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
		{btn(T("nodes_enroll"), "m:secondary:enroll", "primary")},
		{btn(T("relay_exit_on"), "m:secondary:exit:on", "success"), btn(T("relay_exit_off"), "m:secondary:exit:off", "danger")},
		{btn(T("relay_sync"), "m:secondary:sync", ""), btn(T("nodes"), "m:cat:nodes", "primary")},
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
			{btn("🧰 Tools", "m:tools", ""), btn(T("operator"), "m:operator", "")},
			{btn(T("lang"), "m:lang", ""), btn(T("help"), "m:help", "")},
		},
	}
}

// fleet = nodes + routers + sites
func toolsKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⏱ Guest VPN", "m:guest", ""), btn("📡 Guest Wi‑Fi", "m:edgeguest", "")},
		{btn("🛡 DNS", "m:dns", "")},
		{btn("🗓 Backup", "m:backup", ""), btn("📍 Locations", "m:loc", "")},
		{btn("🎥 NVR", "m:nvr", "")},
		{btn("🔄 Updates", "m:updates", ""), btn(T("mtls"), "m:mtls", "")},
		{btn("📦 Git", "m:git", ""), btn("🗄 Registry", "m:registry", "")},
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
}

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
			{btn(T("nodes_enroll"), "m:secondary:enroll", "primary")},
			{btn(T("nodes_sync"), "m:secondary:sync", ""), btn(T("nodes_exit"), "m:secondary:exit:menu", "primary")},
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
	kb = append(kb, []map[string]any{btn(T("nodes_enroll"), "m:secondary:enroll", "primary")})
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
			{btn(T("edge_recovery"), "m:edge_recovery", "primary"), btn(T("edge_register"), "m:edge_register", "")},
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
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("user_access"), "u:access:"+name+":vless", "primary")},
		{btn("50 GiB", "m:quota:"+name+":50", ""), btn("200 GiB", "m:quota:"+name+":200", ""), btn("∞", "m:quota:"+name+":0", ""), btn("Custom", "m:quota:"+name+":custom", "")},
		{btn(T("vpn_rename"), "u:rename:"+name, "")},
		{btn(T("vpn_enable"), "u:enable:"+name, "success"), btn(T("vpn_disable"), "u:disable:"+name, "danger"), btn(T("vpn_revoke"), "u:revoke:"+name, "danger")},
		{btn(T("users"), "m:users", ""), btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func btnURL(text, u string) map[string]any {
	return map[string]any{"text": text, "url": u}
}

func userAccessKeyboard(name, mode string) map[string]any {
	if mode == "" {
		mode = "vless"
	}
	uri := accessPayload(name, mode)
	rows := [][]map[string]any{}
	if uri != "" {
		enc := url.PathEscape(uri)
		var deepRow []map[string]any
		if link := importRedirectURL("shadowrocket://add/" + enc); link != "" {
			deepRow = append(deepRow, btnURL("Shadowrocket", link))
		} else {
			deepRow = append(deepRow, btn("Shadowrocket", "u:app:"+name+":"+mode+":sr", ""))
		}
		if link := importRedirectURL("happ://add/" + enc); link != "" {
			deepRow = append(deepRow, btnURL("Happ", link))
		} else {
			deepRow = append(deepRow, btn("Happ", "u:app:"+name+":"+mode+":happ", ""))
		}
		if link := importRedirectURL("incy://add/" + enc); link != "" {
			deepRow = append(deepRow, btnURL("INCY", link))
		} else {
			deepRow = append(deepRow, btn("INCY", "u:app:"+name+":"+mode+":incy", ""))
		}
		if len(deepRow) > 0 {
			rows = append(rows, deepRow)
		}
	}
	styleV, styleC, styleH := "", "", ""
	switch mode {
	case "core":
		styleC = "primary"
	case "hy2":
		styleH = "primary"
	default:
		styleV = "primary"
	}
	rows = append(rows, []map[string]any{
		btn("VLESS", "u:access:"+name+":vless", styleV),
		btn("Core", "u:access:"+name+":core", styleC),
		btn("HY2", "u:access:"+name+":hy2", styleH),
	})
	if showWorkProfileButton(name) {
		rows = append(rows, []map[string]any{btn("📥 SR Config", "u:workcfg:"+name, "")})
	}
	rows = append(rows, []map[string]any{
		btn(T("user_card"), "u:open:"+name, ""), btn(T("users"), "m:users", ""),
	})
	rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
	return map[string]any{"inline_keyboard": rows}
}

func usersListKeyboard() map[string]any {
	raw := runVPN("list")
	rows := [][]map[string]any{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 1 {
			parts = strings.Fields(line)
		}
		if len(parts) < 1 || parts[0] == "" || parts[0] == "relay-uplink" {
			continue
		}
		name := parts[0]
		rows = append(rows, []map[string]any{btn("👤 "+name, "u:open:"+name, "")})
	}
	rows = append(rows, []map[string]any{btn(T("vpn_add"), "m:vpn_add", "primary")})
	rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "")})
	return map[string]any{"inline_keyboard": rows}
}


func vpnUsersKeyboard() map[string]any { return usersListKeyboard() }
func vpnUsersKeyboardFor(action string) map[string]any { return usersListKeyboard() }
