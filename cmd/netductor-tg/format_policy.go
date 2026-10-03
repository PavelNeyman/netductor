package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/policy"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func formatUserPolicyHTML(name string) string {
	p, err := vpn.GetUserPolicy(name)
	if err != nil {
		return "❌ " + esc(err.Error())
	}
	return formatPolicyHTML("user", name, p)
}

func formatEdgePolicyHTML(id string) string {
	p, err := edge.GetDevicePolicy(id)
	if err != nil {
		return "❌ " + esc(err.Error())
	}
	return formatPolicyHTML("edge", id, p)
}

func formatPolicyHTML(kind, id string, p policy.AccessPolicy) string {
	nl := string([]byte{10})
	cat, _ := policy.EnsureCatalog()
	p.Normalize()
	var b strings.Builder

	titleKey := "pol_title_user"
	if kind == "edge" {
		titleKey = "pol_title_edge"
	}
	b.WriteString("🔐 <b>" + T(titleKey) + "</b> <code>" + esc(id) + "</code>" + nl)

	inetIcon := "🟢"
	if !p.AllowInternet {
		inetIcon = "🔴"
	}
	mode := p.ServicesMode
	if mode == "" {
		mode = "list"
	}

	// Same card pattern as users/dns: compact table + in-message buttons
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>" + T("pol_th_field") + "</th><th>" + T("pol_th_value") + "</th></tr>" + nl)
	b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s <code>%v</code></td></tr>"+nl, T("pol_field_internet"), inetIcon, p.AllowInternet))
	b.WriteString(fmt.Sprintf("<tr><td>%s</td><td><code>%s</code></td></tr>"+nl, T("pol_field_mode"), esc(mode)))
	svc := strings.Join(p.Services, ", ")
	if svc == "" {
		svc = "—"
	}
	b.WriteString(fmt.Sprintf("<tr><td>%s</td><td><code>%s</code></td></tr>"+nl, T("pol_field_services"), esc(svc)))
	b.WriteString("</table>" + nl)

	prefix := "u:pol:" + id + ":"
	if kind == "edge" {
		prefix = "e:pol:" + id + ":"
	}

	inetLabel := T("pol_inet_on")
	inetData := "inet:0"
	if !p.AllowInternet {
		inetLabel = T("pol_inet_off")
		inetData = "inet:1"
	}
	b.WriteString(`<tg-button-row align="left">`)
	style := "danger"
	if p.AllowInternet {
		style = "success"
	}
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="%s" data="%s%s">%s</tg-button>`, style, prefix, inetData, inetLabel))
	modeLabel := T("pol_mode_list")
	if mode == "all" {
		modeLabel = T("pol_mode_all")
	}
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" data="%smode">%s</tg-button>`, prefix, modeLabel))
	b.WriteString(`</tg-button-row>` + nl)

	allowed := map[string]struct{}{}
	for _, s := range p.Services {
		allowed[s] = struct{}{}
	}
	allMode := mode == "all"
	if cat != nil {
		var ids []string
		var labels []string
		for _, s := range cat.Services {
			if s.Disabled || s.Kind == policy.KindEgress || s.ID == "internet" {
				continue
			}
			ids = append(ids, s.ID)
			title := s.Title
			if title == "" {
				title = s.ID
			}
			on := allMode
			if _, ok := allowed[s.ID]; ok {
				on = true
			}
			mark := "☐"
			if on {
				mark = "☑"
			}
			labels = append(labels, mark+" "+title)
		}
		const per = 2
		for i := 0; i < len(ids); i += per {
			b.WriteString(`<tg-button-row align="left">`)
			end := i + per
			if end > len(ids) {
				end = len(ids)
			}
			for j := i; j < end; j++ {
				b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" data="%ssvc:%s">%s</tg-button>`, prefix, ids[j], esc(labels[j])))
			}
			b.WriteString(`</tg-button-row>` + nl)
		}
	}

	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="primary" data="%spreset:media">%s</tg-button>`, prefix, T("pol_preset_media")))
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="primary" data="%spreset:full">%s</tg-button>`, prefix, T("pol_preset_full")))
	b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" data="%spreset:none">%s</tg-button>`, prefix, T("pol_preset_none")))
	b.WriteString(`</tg-button-row>` + nl)
	return b.String()
}

func policyNavKeyboard(kind, id string) map[string]any {
	if kind == "edge" {
		return map[string]any{"inline_keyboard": [][]map[string]any{
			{btn(T("routers"), "m:routers", ""), btn(T("main_menu"), "m:menu", "primary")},
		}}
	}
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("user_card"), "u:open:"+id, ""), btn(T("users"), "m:users", "")},
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func applyPolicyToggle(p policy.AccessPolicy, action string) policy.AccessPolicy {
	p.Normalize()
	switch {
	case action == "inet:0":
		p.AllowInternet = false
	case action == "inet:1":
		p.AllowInternet = true
	case strings.HasPrefix(action, "inet:"):
		p.AllowInternet = !p.AllowInternet
	case action == "mode":
		if p.ServicesMode == "all" {
			p.ServicesMode = "list"
		} else {
			p.ServicesMode = "all"
		}
	case strings.HasPrefix(action, "svc:"):
		sid := strings.TrimPrefix(action, "svc:")
		if p.ServicesMode == "all" {
			cat, _ := policy.EnsureCatalog()
			var all []string
			if cat != nil {
				for _, s := range cat.Services {
					if s.Disabled || s.Kind == policy.KindEgress || s.ID == "internet" {
						continue
					}
					if s.ID != sid {
						all = append(all, s.ID)
					}
				}
			}
			p.ServicesMode = "list"
			p.Services = all
		} else {
			found := false
			out := make([]string, 0, len(p.Services))
			for _, s := range p.Services {
				if s == sid {
					found = true
					continue
				}
				out = append(out, s)
			}
			if !found {
				out = append(out, sid)
			}
			p.Services = out
		}
	case action == "preset:media":
		p = policy.ApplyPreset(policy.PresetMedia, p)
	case action == "preset:full":
		p = policy.ApplyPreset(policy.PresetFull, p)
	case action == "preset:none":
		p = policy.ApplyPreset(policy.PresetNone, p)
	}
	p.Normalize()
	return p
}
