package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
)

// handleEdgeTemplateCB — TG UI for edge template vpn.dns / mode / fallback.
// m:edgetpl | m:edgetpl:show:<id> | m:edgetpl:pick | m:edgetpl:dns:<id>:<vpn|wan|off> |
// m:edgetpl:mode:<id>:<tun|off> | m:edgetpl:fb:<id>:<wan|block> | m:edgetpl:soft:<id>:<0|1> | m:edgetpl:en:<id>:<0|1>
func handleEdgeTemplateCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:edgetpl" && !strings.HasPrefix(data, "m:edgetpl:") {
		return false
	}
	edge.EnsureDefaultTemplate()
	id := "default"

	if data == "m:edgetpl:pick" {
		reply(token, chat, msgID, formatEdgeTplPickHTML(), edgeTplPickKeyboard())
		return true
	}
	if data == "m:edgetpl" || data == "m:edgetpl:show" || strings.HasPrefix(data, "m:edgetpl:show:") {
		if strings.HasPrefix(data, "m:edgetpl:show:") {
			id = strings.TrimPrefix(data, "m:edgetpl:show:")
		}
		reply(token, chat, msgID, formatEdgeTplVPNHTML(id), edgeTplVPNKeyboard(id))
		return true
	}
	rest := strings.TrimPrefix(data, "m:edgetpl:")
	parts := strings.Split(rest, ":")
	if len(parts) < 2 {
		reply(token, chat, msgID, formatEdgeTplVPNHTML("default"), edgeTplVPNKeyboard("default"))
		return true
	}
	act := parts[0]
	id = parts[1]
	if id == "" {
		id = "default"
	}
	kvs := map[string]any{}
	switch act {
	case "dns":
		if len(parts) >= 3 {
			kvs["dns"] = parts[2]
		}
	case "mode":
		if len(parts) >= 3 {
			kvs["mode"] = parts[2]
		}
	case "fb":
		if len(parts) >= 3 {
			kvs["fallback"] = parts[2]
		}
	case "soft":
		if len(parts) >= 3 {
			kvs["soft_fallback"] = parts[2] == "1"
		}
	case "en":
		if len(parts) >= 3 {
			kvs["enabled"] = parts[2] == "1"
		}
	case "show":
		reply(token, chat, msgID, formatEdgeTplVPNHTML(id), edgeTplVPNKeyboard(id))
		return true
	default:
		reply(token, chat, msgID, formatEdgeTplVPNHTML(id), edgeTplVPNKeyboard(id))
		return true
	}
	if len(kvs) > 0 {
		if _, err := edge.SetTemplateVPN(id, kvs); err != nil {
			args := []string{"edge", "template-set-vpn", id}
			for k, v := range kvs {
				args = append(args, fmt.Sprintf("%s=%v", k, v))
			}
			out, err2 := exec.Command(netductorBin(), args...).CombinedOutput()
			if err2 != nil {
				reply(token, chat, msgID, "❌ "+esc(string(out)), edgeTplVPNKeyboard(id))
				return true
			}
		}
	}
	reply(token, chat, msgID, formatEdgeTplVPNHTML(id), edgeTplVPNKeyboard(id))
	return true
}

func formatEdgeTplPickHTML() string {
	nl := "\n"
	var b strings.Builder
	b.WriteString("🧩 <b>" + esc(T("edge_tpl_vpn")) + "</b>" + nl)
	b.WriteString("<i>" + esc(T("edge_tpl_pick_hint")) + "</i>" + nl)
	list := edge.ListTemplates()
	if len(list) == 0 {
		b.WriteString("<p>—</p>")
		return b.String()
	}
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>#</th><th>id</th></tr>" + nl)
	for i, tmpl := range list {
		tid, _ := tmpl["id"].(string)
		if tid == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td></tr>"+nl, i+1, esc(tid)))
	}
	b.WriteString("</table>" + nl)
	b.WriteString(`<tg-button-row align="left">`)
	n := 0
	for _, tmpl := range list {
		tid, _ := tmpl["id"].(string)
		if tid == "" {
			continue
		}
		n++
		if n > 8 {
			break
		}
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgetpl:show:%s">%s</tg-button>`, tid, esc(tid)))
	}
	b.WriteString(`</tg-button-row>`)
	return b.String()
}

func edgeTplPickKeyboard() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("«", "m:cat:routers", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}

func formatEdgeTplVPNHTML(id string) string {
	if id == "" {
		id = "default"
	}
	tmpl, err := edge.GetTemplate(id)
	nl := "\n"
	var b strings.Builder
	b.WriteString("🧩 <b>" + esc(T("edge_tpl_vpn")) + "</b> · <code>" + esc(id) + "</code>" + nl)
	b.WriteString("<i>" + esc(T("edge_tpl_vpn_hint")) + "</i>" + nl)
	if err != nil {
		b.WriteString("<p>❌ " + esc(err.Error()) + "</p>")
		return b.String()
	}
	vpn, _ := tmpl["vpn"].(map[string]any)
	if vpn == nil {
		vpn = map[string]any{}
	}
	get := func(k string, def string) string {
		if v, ok := vpn[k]; ok && fmt.Sprint(v) != "" {
			return fmt.Sprint(v)
		}
		if k == "dns" {
			if v, ok := vpn["dns_mode"]; ok {
				return fmt.Sprint(v)
			}
		}
		return def
	}
	boolish := func(k string, def bool) bool {
		v, ok := vpn[k]
		if !ok {
			return def
		}
		switch x := v.(type) {
		case bool:
			return x
		case string:
			return x == "1" || strings.EqualFold(x, "true") || strings.EqualFold(x, "yes")
		default:
			return def
		}
	}

	curDNS := get("dns", "vpn")
	curMode := get("mode", "tun")
	curFB := get("fallback", "wan")
	curSoft := boolish("soft_fallback", true)
	curEn := boolish("enabled", true)

	// Current values table
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>" + esc(T("edge_tpl_col_param")) + "</th><th>" + esc(T("edge_tpl_col_value")) + "</th></tr>" + nl)
	b.WriteString("<tr><td>enabled</td><td><b>" + esc(boolLabel(curEn)) + "</b></td></tr>" + nl)
	b.WriteString("<tr><td>dns</td><td><b>" + esc(curDNS) + "</b></td></tr>" + nl)
	b.WriteString("<tr><td>mode</td><td><b>" + esc(curMode) + "</b></td></tr>" + nl)
	b.WriteString("<tr><td>fallback</td><td><b>" + esc(curFB) + "</b></td></tr>" + nl)
	b.WriteString("<tr><td>soft_fallback</td><td><b>" + esc(boolLabel(curSoft)) + "</b></td></tr>" + nl)
	b.WriteString("</table>" + nl)

	// Legend / descriptions
	b.WriteString("<blockquote>" + nl)
	b.WriteString("<b>enabled</b> — " + esc(T("edge_tpl_desc_enabled")) + nl)
	b.WriteString("· <code>on</code> — " + esc(T("edge_tpl_en_on")) + nl)
	b.WriteString("· <code>off</code> — " + esc(T("edge_tpl_en_off")) + nl + nl)
	b.WriteString("<b>dns</b> — " + esc(T("edge_tpl_desc_dns")) + nl)
	b.WriteString("· <code>vpn</code> — " + esc(T("edge_tpl_dns_vpn")) + nl)
	b.WriteString("· <code>wan</code> — " + esc(T("edge_tpl_dns_wan")) + nl)
	b.WriteString("· <code>off</code> — " + esc(T("edge_tpl_dns_off")) + nl + nl)
	b.WriteString("<b>mode</b> — " + esc(T("edge_tpl_desc_mode")) + nl)
	b.WriteString("· <code>tun</code> — " + esc(T("edge_tpl_mode_tun")) + nl)
	b.WriteString("· <code>off</code> — " + esc(T("edge_tpl_mode_off")) + nl + nl)
	b.WriteString("<b>fallback</b> — " + esc(T("edge_tpl_desc_fb")) + nl)
	b.WriteString("· <code>wan</code> — " + esc(T("edge_tpl_fb_wan")) + nl)
	b.WriteString("· <code>block</code> — " + esc(T("edge_tpl_fb_block")) + nl + nl)
	b.WriteString("<b>soft_fallback</b> — " + esc(T("edge_tpl_desc_soft")) + nl)
	b.WriteString("· <code>on</code> — " + esc(T("edge_tpl_soft_on")) + nl)
	b.WriteString("· <code>off</code> — " + esc(T("edge_tpl_soft_off")) + nl)
	b.WriteString("</blockquote>" + nl)

	// Buttons: enabled
	b.WriteString("<p><b>enabled</b></p>" + nl)
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:en:%s:1">on</tg-button>`, styleIf(curEn), id))
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:en:%s:0">off</tg-button>`, styleIf(!curEn), id))
	b.WriteString(`</tg-button-row>`)

	b.WriteString("<p><b>dns</b></p>" + nl)
	b.WriteString(`<tg-button-row align="left">`)
	for _, d := range []string{"vpn", "wan", "off"} {
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:dns:%s:%s">%s</tg-button>`, styleIf(curDNS == d), id, d, d))
	}
	b.WriteString(`</tg-button-row>`)

	b.WriteString("<p><b>mode</b></p>" + nl)
	b.WriteString(`<tg-button-row align="left">`)
	for _, m := range []string{"tun", "off"} {
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:mode:%s:%s">%s</tg-button>`, styleIf(curMode == m), id, m, m))
	}
	b.WriteString(`</tg-button-row>`)

	b.WriteString("<p><b>fallback</b></p>" + nl)
	b.WriteString(`<tg-button-row align="left">`)
	for _, f := range []string{"wan", "block"} {
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:fb:%s:%s">%s</tg-button>`, styleIf(curFB == f), id, f, f))
	}
	b.WriteString(`</tg-button-row>`)

	b.WriteString("<p><b>soft_fallback</b></p>" + nl)
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:soft:%s:1">on</tg-button>`, styleIf(curSoft), id))
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:soft:%s:0">off</tg-button>`, styleIf(!curSoft), id))
	b.WriteString(`</tg-button-row>`)

	return b.String()
}

func styleIf(on bool) string {
	if on {
		return ` style="primary"`
	}
	return ""
}

func boolLabel(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func edgeTplVPNKeyboard(id string) map[string]any {
	if id == "" {
		id = "default"
	}
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("🔄", "m:edgetpl:show:"+id, "primary"), btn("📋", "m:edgetpl:pick", "link"), btn("«", "m:cat:routers", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}
