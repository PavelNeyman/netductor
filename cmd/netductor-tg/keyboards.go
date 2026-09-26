package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// navKeyboard — единый «назад»: « <имя родителя> + Главное меню.
func navKeyboard(parentCB, parentLabel string) map[string]any {
	if parentLabel == "" {
		parentLabel = T("back")
	}
	back := parentLabel
	if !strings.HasPrefix(back, "«") {
		back = "« " + back
	}
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(back, parentCB, "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}

func parentFleet() string {
	if getLang() != "en" {
		return "Флот"
	}
	return "Fleet"
}
func parentNodes() string {
	if getLang() != "en" {
		return "Ноды"
	}
	return "Nodes"
}
func parentTools() string {
	if getLang() != "en" {
		return "Инструменты"
	}
	return "Tools"
}
func parentOperator() string {
	if getLang() != "en" {
		return "Оператор"
	}
	return "Operator"
}
func parentSites() string {
	if getLang() != "en" {
		return "Площадки"
	}
	return "Sites"
}
func parentRouters() string {
	if getLang() != "en" {
		return "Роутеры"
	}
	return "Routers"
}


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
	// Navigation only — actions in secondary body (formatRelayActionsHTML).
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("nodes"), "m:cat:nodes", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}

func formatRelayActionsHTML() string {
	// Secondary traffic controls only (enroll lives under Nodes hub once).
	return `<tg-button-row align="left">` +
		`<tg-button type="callback_data" style="success" data="m:secondary:exit:on">` + T("relay_exit_on") + `</tg-button>` +
		`<tg-button type="callback_data" style="danger" data="m:secondary:exit:off">` + T("relay_exit_off") + `</tg-button>` +
		`<tg-button type="callback_data" style="link" data="m:secondary:sync">` + T("relay_sync") + `</tg-button>` +
		`</tg-button-row>`
}


func mainKeyboard() map[string]any {
	// Flat top: 4 clear areas. No VPN-tools duplicate.
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn(T("status"), "m:status", "primary")},
			{btn(T("users"), "m:users", "primary")},
			{btn(T("fleet"), "m:fleet", "primary")},
			{btn(T("tools"), "m:tools", ""), btn(T("operator"), "m:operator", "")},
			{btn(T("lang"), "m:lang", ""), btn(T("help"), "m:help", "")},
		},
	}
}

// fleet = nodes + routers + sites

func fleetKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func fleetHubHTML() string {
	ru := getLang() != "en"
	title := "🌐 <b>Fleet</b>"
	if ru {
		title = "🌐 <b>Флот</b>"
	}
	return title + "\n" +
		`<tg-button-row align="left">` +
		`<tg-button type="callback_data" style="primary" data="m:cat:nodes">` + T("nodes") + `</tg-button>` +
		`<tg-button type="callback_data" style="primary" data="m:cat:routers">` + T("cat_routers") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:cat:sites">` + T("sites") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:addons">` + T("addons") + `</tg-button>` +
		`</tg-button-row>`
}


// operator = session / admin / audit (not day-to-day user VPN)
func operatorKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func operatorSubKeyboard() map[string]any {
	return navKeyboard("m:operator", parentOperator())
}

func operatorHubHTML() string {
	ru := getLang() != "en"
	title := "🛠 <b>Operator</b>"
	if ru {
		title = "🛠 <b>Оператор</b>"
	}
	session, help, sessions, audit, refresh := "🔑 Session", "ℹ️ Mac Admin", "📋 Sessions", "📜 Audit", "🔄 Refresh VPN links"
	if ru {
		session, help, sessions, audit, refresh = "🔑 Сессия", "ℹ️ Админ на Mac", "📋 Сессии", "📜 Аудит", "🔄 Обновить ссылки VPN"
	}
	return title + "\n" +
		`<tg-button-row align="left">` +
		`<tg-button type="callback_data" style="primary" data="m:session">` + session + `</tg-button>` +
		`<tg-button type="callback_data" data="m:sessions">` + sessions + `</tg-button>` +
		`<tg-button type="callback_data" data="m:audit">` + audit + `</tg-button>` +
		`</tg-button-row>` +
		`<tg-button-row align="left">` +
		`<tg-button type="callback_data" data="m:vpn_refresh">` + refresh + `</tg-button>` +
		`<tg-button type="callback_data" data="m:admin">` + help + `</tg-button>` +
		`</tg-button-row>`
}

func vpnKeyboard() map[string]any {
	// legacy alias → users
	return usersListKeyboard()
}

func addonsKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("back"), "m:fleet", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}

func sitesListKeyboard() map[string]any {
	return navKeyboard("m:sites", parentSites())
}

func sitesKeyboard() map[string]any {
	return navKeyboard("m:fleet", parentFleet())
}

func sitesHubHTML() string {
	return T("sites_title") + "\n" +
		`<tg-button-row align="left">` +
		`<tg-button type="callback_data" style="primary" data="m:sites:list">` + T("sites_list") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:sites:rsc">` + T("sites_rsc") + `</tg-button>` +
		`</tg-button-row>`
}


func nodesKeyboard() map[string]any {
	return navKeyboard("m:fleet", parentFleet())
}

func nodesHubHTML() string {
	ru := getLang() != "en"
	title := "🖥 <b>Nodes</b>"
	if ru {
		title = "🖥 <b>Ноды</b>"
	}
	// List/rename/SSH + secondary ops (sync/exit). Enroll/deploy — only from Mac op/TUI, not TG.
	return title + "\n" +
		`<tg-button-row align="left">` +
		`<tg-button type="callback_data" style="primary" data="m:nodes_list">` + T("nodes_list_btn") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:node_rename">` + T("nodes_rename_btn") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:sshhosts">` + T("ssh_hosts") + `</tg-button>` +
		`</tg-button-row>` +
		`<tg-button-row align="left">` +
		`<tg-button type="callback_data" data="m:secondary:sync">` + T("nodes_sync") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:secondary:exit:menu">` + T("nodes_exit") + `</tg-button>` +
		`</tg-button-row>`
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
	if getLang() != "en" {
		b.WriteString("<i># открыть карточку ноды</i>" + nl)
	} else {
		b.WriteString("<i># open node card</i>" + nl)
	}
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>#</th><th>host</th><th>role</th><th>status</th></tr>" + nl)
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
		role := displayRole(r.Role)
		if role == "" {
			role = "—"
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td></tr>"+nl,
			i+1, esc(label), esc(role), icon))
	}
	b.WriteString("</table>" + nl)
	const per = 5
	for i, r := range rows {
		if i%per == 0 {
			if i > 0 {
				b.WriteString(`</tg-button-row>` + nl)
			}
			b.WriteString(`<tg-button-row align="left">`)
		}
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nd:o:%s">%d</tg-button>`, r.ID, i+1))
	}
	if len(rows) > 0 {
		b.WriteString(`</tg-button-row>` + nl)
	}
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
	return navKeyboard("m:cat:nodes", parentNodes())
}


func nodesRenameKeyboard() map[string]any {
	return navKeyboard("m:cat:nodes", parentNodes())
}

// formatNodesRenameHTML — pick node by number (body buttons).
func formatNodesRenameHTML() string {
	rows := parseNodesList()
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString(T("nodes_rename") + nl)
	if len(rows) == 0 {
		b.WriteString(T("nodes_empty"))
		return b.String()
	}
	b.WriteString("<table bordered striped compact>" + nl + "<tr><th>#</th><th>host</th><th>role</th></tr>" + nl)
	for i, r := range rows {
		if i >= 20 {
			break
		}
		name := r.Host
		if name == "" {
			name = r.ID
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td></tr>"+nl, i+1, esc(name), esc(displayRole(r.Role))))
	}
	b.WriteString("</table>" + nl)
	b.WriteString(`<tg-button-row align="left">`)
	for i, r := range rows {
		if i >= 20 {
			break
		}
		data := "m:nr:" + r.ID
		if len(data) > 64 {
			data = data[:64]
		}
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="%s">%d</tg-button>`, data, i+1))
	}
	b.WriteString(`</tg-button-row>`)
	return b.String()
}



func routersKeyboard() map[string]any {
	return navKeyboard("m:fleet", parentFleet())
}

func routersHubHTML() string {
	ru := getLang() != "en"
	title := "📡 <b>Routers</b>"
	if ru {
		title = "📡 <b>Роутеры</b>"
	}
	return title + "\n" +
		`<tg-button-row align="left">` +
		`<tg-button type="callback_data" style="primary" data="m:routers">` + T("devices") + `</tg-button>` +
		`<tg-button type="callback_data" style="primary" data="m:pending">` + T("pending") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:edge_recovery">` + T("edge_recovery") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:edge_register">` + T("edge_register") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:templates">` + T("templates") + `</tg-button>` +
		`<tg-button type="callback_data" data="m:edge_bind">` + T("bind_tmpl") + `</tg-button>` +
		`<tg-button type="callback_data" style="primary" data="m:edge_apply">` + T("apply_tmpl") + `</tg-button>` +
		`</tg-button-row>`
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
	_ = mode
	rows := [][]map[string]any{
		{btn(T("user_card"), "u:open:"+name, ""), btn(T("users"), "m:users", "")},
	}
	// Classic Telegram URL button (always visible; rich tg-button may be stripped by client).
	rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
	return map[string]any{"inline_keyboard": rows}
}

// legacy alias

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


func helpHubHTML() string {
	ru := getLang() != "en"
	body := helpText()
	if ru {
		body += "\n" + `<tg-button-row align="left">` +
			`<tg-button type="callback_data" style="primary" data="m:status">📊 Статус</tg-button>` +
			`<tg-button type="callback_data" data="m:users">👥 Users</tg-button>` +
			`<tg-button type="callback_data" data="m:fleet">🌐 Флот</tg-button>` +
			`<tg-button type="callback_data" data="m:tools">🧰 Tools</tg-button>` +
			`</tg-button-row>`
	} else {
		body += "\n" + `<tg-button-row align="left">` +
			`<tg-button type="callback_data" style="primary" data="m:status">📊 Status</tg-button>` +
			`<tg-button type="callback_data" data="m:users">👥 Users</tg-button>` +
			`<tg-button type="callback_data" data="m:fleet">🌐 Fleet</tg-button>` +
			`<tg-button type="callback_data" data="m:tools">🧰 Tools</tg-button>` +
			`</tg-button-row>`
	}
	return body
}


func displayRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "", "core":
		return "primary"
	case "relay":
		return "secondary"
	default:
		return strings.ToLower(strings.TrimSpace(role))
	}
}
