package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/policy"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

var (
	appPresetDraft presetDraft
	appPresetOK    bool
)

func catalogServiceIDs() []string {
	cat, err := policy.EnsureCatalog()
	if err != nil || cat == nil {
		return nil
	}
	var ids []string
	for _, s := range cat.Services {
		if s.Disabled || s.Kind == policy.KindEgress || s.ID == "internet" {
			continue
		}
		ids = append(ids, s.ID)
		if len(ids) >= 12 {
			break
		}
	}
	return ids
}

func buildAppPresetDraft() (string, error) {
	d := appPresetDraft
	if !appPresetOK {
		d = presetDraft{AllowInternet: true, ServicesMode: "list"}
		appPresetDraft = d
		appPresetOK = true
	}
	svcs := catalogServiceIDs()
	var lines []string
	lines = append(lines, fmt.Sprintf("inet=%v mode=%s", d.AllowInternet, d.ServicesMode))
	set := map[string]bool{}
	for _, s := range d.Services {
		set[s] = true
	}
	var btns [][2]string
	for _, id := range svcs {
		mark := "·"
		if set[id] {
			mark = "✓"
		}
		lines = append(lines, mark+" "+id)
		btns = append(btns, [2]string{"ps:" + id, mark + id})
	}
	return appMarshal(appState{
		S: "prdraft", T: appTitle("Preset draft", "Черновик пресета"),
		Lines: strings.Join(lines, "\n"),
		Acts:  "pd_net,pd_mode,pd_save",
		Btns:  btns,
		H:     appTitle("toggle services · save → name in chat", "сервисы · save → имя в чат"),
	})
}

func buildAppUserPolicy(name string) (string, error) {
	p, err := vpn.GetUserPolicy(name)
	if err != nil {
		return appMarshal(appState{S: "upol", T: name, B: err.Error()})
	}
	svcs := catalogServiceIDs()
	set := map[string]bool{}
	for _, s := range p.Services {
		set[s] = true
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("inet=%v mode=%s", p.AllowInternet, p.ServicesMode))
	var btns [][2]string
	for _, id := range svcs {
		mark := "·"
		if set[id] {
			mark = "✓"
		}
		lines = append(lines, mark+" "+id)
		btns = append(btns, [2]string{"us:" + id, mark + id})
	}
	appMu.Lock()
	appFocus = name
	appMu.Unlock()
	return appMarshal(appState{
		S: "upol", T: name,
		Lines: strings.Join(lines, "\n"),
		Acts:  "up_inet,up_mode",
		Btns:  btns,
		H:     appTitle("user policy", "политика пользователя"),
	})
}

func buildAppTemplateCard(id string) (string, error) {
	// status via ND or template get
	out := runND("edge", "template", "show", id)
	return appMarshal(appState{
		S: "edgetpl", T: id,
		Lines: widgetTruncate(out, 280),
		Acts:  "et_on,et_off,et_dns_vpn,et_dns_wan,et_dns_off,et_mode_tun,et_mode_off,et_fb_wan,et_fb_block,et_soft_on,et_soft_off",
		H:     id,
	})
}

func buildAppSvcPaths() (string, error) {
	out := runND("svc-paths", "status")
	return appMarshal(appState{
		S: "svcpaths", T: "Svc paths",
		Lines: widgetTruncate(out, 400),
		Acts:  "spa_apply",
	})
}

func handleAppFinalAct(token string, q *widgetQuery, act, focus string) bool {
	switch act {
	case "pd_net":
		appPresetDraft.AllowInternet = !appPresetDraft.AllowInternet
		appPresetOK = true
		cd, _ := buildAppPresetDraft()
		appPushState(token, q, cd)
		return true
	case "pd_mode":
		if appPresetDraft.ServicesMode == "all" {
			appPresetDraft.ServicesMode = "list"
		} else {
			appPresetDraft.ServicesMode = "all"
		}
		appPresetOK = true
		cd, _ := buildAppPresetDraft()
		appPushState(token, q, cd)
		return true
	case "pd_save":
		if q.Chat == nil {
			return true
		}
		// persist draft into chatExtra for handlePresetNameText
		saveDraft(q.Chat.ID, appPresetDraft)
		setState(q.Chat.ID, "wait_preset_name", chatExtra[q.Chat.ID])
		mid := sendPlainHTML(token, q.Chat.ID, appTitle("Send preset <b>id/name</b>", "Пришлите <b>id/имя</b> пресета"))
		if mid > 0 {
			// keep draft in extra; prompt id lost — name handler clears state
			saveDraft(q.Chat.ID, appPresetDraft)
			setState(q.Chat.ID, "wait_preset_name", chatExtra[q.Chat.ID])
		}
		cd, _ := appMarshal(appState{S: "prdraft", T: "Presets", B: appTitle("waiting name in chat…", "ждём имя в чате…"), Acts: "pd_net,pd_mode,pd_save"})
		appPushState(token, q, cd)
		return true
	case "pr_draft":
		appPresetOK = true
		appPresetDraft = presetDraft{AllowInternet: true, ServicesMode: "list"}
		cd, _ := buildAppPresetDraft()
		appPushState(token, q, cd)
		return true
	case "up_inet":
		if focus == "" {
			return true
		}
		p, err := vpn.GetUserPolicy(focus)
		if err != nil {
			cd, _ := appMarshal(appState{S: "upol", T: focus, B: err.Error()})
			appPushState(token, q, cd)
			return true
		}
		p = applyPolicyToggle(p, "inet:x")
		_ = vpn.SetUserPolicy(focus, p, "tg")
		cd, _ := buildAppUserPolicy(focus)
		appPushState(token, q, cd)
		return true
	case "up_mode":
		if focus == "" {
			return true
		}
		p, err := vpn.GetUserPolicy(focus)
		if err != nil {
			cd, _ := appMarshal(appState{S: "upol", T: focus, B: err.Error()})
			appPushState(token, q, cd)
			return true
		}
		p = applyPolicyToggle(p, "mode")
		_ = vpn.SetUserPolicy(focus, p, "tg")
		cd, _ := buildAppUserPolicy(focus)
		appPushState(token, q, cd)
		return true
	case "upol":
		if focus == "" {
			return true
		}
		cd, _ := buildAppUserPolicy(focus)
		appPushState(token, q, cd)
		return true
	case "spa_apply":
		out := runND("svc-paths", "apply")
		cd, _ := appMarshal(appState{S: "svcpaths", T: "Svc paths", Lines: widgetTruncate(out, 450), Acts: "spa_apply"})
		appPushState(token, q, cd)
		return true
	case "room_add":
		if focus == "" || q.Chat == nil {
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_room_add:"+focus,
			appTitle("Send room id (a-z0-9-)", "Пришлите id комнаты (a-z0-9-)"))
		cd, _ := appMarshal(appState{S: "loc", T: focus, B: appTitle("waiting room id…", "ждём id комнаты…"), Acts: "room_add"})
		appPushState(token, q, cd)
		return true
	case "et_dns_vpn", "et_dns_wan", "et_dns_off":
		if focus == "" {
			return true
		}
		v := strings.TrimPrefix(act, "et_dns_")
		_, err := edge.SetTemplateVPN(focus, map[string]any{"dns": v})
		msg := "dns=" + v
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "edgetpl", T: focus, B: msg, Acts: "et_on,et_off,et_dns_vpn,et_dns_wan,et_dns_off,et_mode_tun,et_mode_off,et_fb_wan,et_fb_block,et_soft_on,et_soft_off"})
		appPushState(token, q, cd)
		return true
	case "et_mode_tun", "et_mode_off":
		if focus == "" {
			return true
		}
		v := "tun"
		if act == "et_mode_off" {
			v = "off"
		}
		_, err := edge.SetTemplateVPN(focus, map[string]any{"mode": v})
		msg := "mode=" + v
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "edgetpl", T: focus, B: msg, Acts: "et_on,et_off,et_dns_vpn,et_dns_wan,et_dns_off,et_mode_tun,et_mode_off,et_fb_wan,et_fb_block,et_soft_on,et_soft_off"})
		appPushState(token, q, cd)
		return true
	case "et_fb_wan", "et_fb_block":
		if focus == "" {
			return true
		}
		v := strings.TrimPrefix(act, "et_fb_")
		_, err := edge.SetTemplateVPN(focus, map[string]any{"fallback": v})
		msg := "fallback=" + v
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "edgetpl", T: focus, B: msg, Acts: "et_on,et_off,et_dns_vpn,et_dns_wan,et_dns_off,et_mode_tun,et_mode_off,et_fb_wan,et_fb_block,et_soft_on,et_soft_off"})
		appPushState(token, q, cd)
		return true
	case "et_soft_on", "et_soft_off":
		if focus == "" {
			return true
		}
		on := act == "et_soft_on"
		_, err := edge.SetTemplateVPN(focus, map[string]any{"soft_fallback": on})
		msg := fmt.Sprintf("soft=%v", on)
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "edgetpl", T: focus, B: msg, Acts: "et_on,et_off,et_dns_vpn,et_dns_wan,et_dns_off,et_mode_tun,et_mode_off,et_fb_wan,et_fb_block,et_soft_on,et_soft_off"})
		appPushState(token, q, cd)
		return true
	}

	if strings.HasPrefix(act, "ps:") {
		sid := strings.TrimPrefix(act, "ps:")
		d := appPresetDraft
		found := false
		var next []string
		for _, s := range d.Services {
			if s == sid {
				found = true
				continue
			}
			next = append(next, s)
		}
		if !found {
			next = append(next, sid)
		}
		d.Services = next
		appPresetDraft = d
		appPresetOK = true
		cd, _ := buildAppPresetDraft()
		appPushState(token, q, cd)
		return true
	}
	if strings.HasPrefix(act, "us:") {
		sid := strings.TrimPrefix(act, "us:")
		if focus == "" {
			return true
		}
		p, err := vpn.GetUserPolicy(focus)
		if err != nil {
			cd, _ := appMarshal(appState{S: "upol", T: focus, B: err.Error()})
			appPushState(token, q, cd)
			return true
		}
		p = applyPolicyToggle(p, "svc:"+sid)
		_ = vpn.SetUserPolicy(focus, p, "tg")
		cd, _ := buildAppUserPolicy(focus)
		appPushState(token, q, cd)
		return true
	}
	return false
}
