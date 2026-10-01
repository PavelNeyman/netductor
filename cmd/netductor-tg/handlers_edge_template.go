package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
)

// handleEdgeTemplateCB — TG UI for edge template vpn.dns / mode / fallback.
// m:edgetpl | m:edgetpl:show:<id> | m:edgetpl:dns:<id>:<vpn|wan|off> |
// m:edgetpl:mode:<id>:<tun|off> | m:edgetpl:fb:<id>:<wan|block> | m:edgetpl:soft:<id>:<0|1>
func handleEdgeTemplateCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:edgetpl" && !strings.HasPrefix(data, "m:edgetpl:") {
		return false
	}
	edge.EnsureDefaultTemplate()
	id := "default"
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
			// fallback CLI if package path differs in older builds
			_ = err
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
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
	b.WriteString("<tr><td>enabled</td><td>" + esc(fmt.Sprint(boolish("enabled", true))) + "</td></tr>" + nl)
	b.WriteString("<tr><td>dns</td><td><b>" + esc(get("dns", "vpn")) + "</b></td></tr>" + nl)
	b.WriteString("<tr><td>mode</td><td>" + esc(get("mode", "tun")) + "</td></tr>" + nl)
	b.WriteString("<tr><td>fallback</td><td>" + esc(get("fallback", "wan")) + "</td></tr>" + nl)
	b.WriteString("<tr><td>soft_fallback</td><td>" + esc(fmt.Sprint(boolish("soft_fallback", true))) + "</td></tr>" + nl)
	b.WriteString("</table>" + nl)
	// rich buttons in body
	b.WriteString(`<tg-button-row align="left">`)
	for _, d := range []string{"vpn", "wan", "off"} {
		st := ""
		if get("dns", "vpn") == d {
			st = ` style="primary"`
		}
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:dns:%s:%s">dns:%s</tg-button>`, st, id, d, d))
	}
	b.WriteString(`</tg-button-row>`)
	b.WriteString(`<tg-button-row align="left">`)
	for _, m := range []string{"tun", "off"} {
		st := ""
		if get("mode", "tun") == m {
			st = ` style="primary"`
		}
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:mode:%s:%s">mode:%s</tg-button>`, st, id, m, m))
	}
	b.WriteString(`</tg-button-row>`)
	b.WriteString(`<tg-button-row align="left">`)
	for _, f := range []string{"wan", "block"} {
		st := ""
		if get("fallback", "wan") == f {
			st = ` style="primary"`
		}
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:fb:%s:%s">fb:%s</tg-button>`, st, id, f, f))
	}
	softOn := boolish("soft_fallback", true)
	st0, st1 := "", ` style="primary"`
	if !softOn {
		st0, st1 = ` style="primary"`, ""
	}
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:soft:%s:1">soft:on</tg-button>`, st1, id))
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data"%s data="m:edgetpl:soft:%s:0">soft:off</tg-button>`, st0, id))
	b.WriteString(`</tg-button-row>`)
	return b.String()
}

func edgeTplVPNKeyboard(id string) map[string]any {
	if id == "" {
		id = "default"
	}
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("🔄", "m:edgetpl:show:"+id, "primary"), btn("«", "m:cat:routers", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
}
