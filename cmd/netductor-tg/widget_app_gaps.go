package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/policy"
	"github.com/PavelNeyman/netductor/internal/sites"
)

// Remaining classic parity: presets, mtls, secondary enroll, edge lifecycle, loc rooms, policy toggles, fmt confirm.

func buildAppPresets() (string, error) {
	list, _ := policy.LoadCustomPresets()
	var lines []string
	var ids []string
	lines = append(lines, "full / media / none (built-in)")
	for i, p := range list {
		if i >= 10 {
			break
		}
		lines = append(lines, fmt.Sprintf("%d %s %s", len(ids)+1, p.ID, p.Title))
		ids = append(ids, p.ID)
	}
	appMu.Lock()
	appSessUnlocked().ListIDs = ids
	appMu.Unlock()
	return appMarshal(appState{
		S: "presets", T: appTitle("Presets", "Пресеты"),
		Lines: strings.Join(lines, "\n"),
		IDs:   strings.Join(ids, ","),
		Acts:  "pr_new,pr_draft",
		H:     appTitle("# custom · new via chat name", "# свои · new = имя в чат"),
	})
}

func buildAppMTLS() (string, error) {
	out := runND("mtls", "list")
	if strings.TrimSpace(out) == "" {
		out = runND("mtls", "status")
	}
	return appMarshal(appState{
		S: "mtls", T: "mTLS",
		Lines: appLines(out),
		Acts:  "mt_list,mt_rot,mt_roll",
	})
}

func buildAppSecondaryOps() (string, error) {
	out := strings.TrimSpace(runND("secondary", "list"))
	return appMarshal(appState{
		S: "secops", T: appTitle("Secondary ops", "Secondary"),
		Lines: appLines(out),
		Acts:  "sec_exp,sec_ol,sec_enr,sec_sync",
		H:     appTitle("export · oneline · enroll · sync", "export · oneline · enroll · sync"),
	})
}

func buildAppEdgeLife() (string, error) {
	devs := edge.ListDevices()
	var lines, ids []string
	for i, d := range devs {
		if d.DeviceID == "" || i >= 10 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d %s %s", len(ids)+1, d.DeviceID, d.Status))
		ids = append(ids, d.DeviceID)
	}
	appMu.Lock()
	appSessUnlocked().ListIDs = ids
	appMu.Unlock()
	return appMarshal(appState{
		S: "edgelife", T: appTitle("Edge devices", "Edge устройства"),
		Lines: strings.Join(lines, "\n"),
		IDs:   strings.Join(ids, ","),
		Acts:  "er_reg,er_bind,er_apply,er_site,er_rec",
		H:     appTitle("register/bind/apply via chat", "register/bind/apply через чат"),
	})
}

func buildAppLoc() (string, error) {
	list, err := sites.List()
	if err != nil {
		return appMarshal(appState{S: "loc", T: "Locations", B: err.Error()})
	}
	var lines, ids []string
	for i, s := range list {
		if i >= 12 {
			break
		}
		lines = append(lines, fmt.Sprintf("%d %s", len(ids)+1, s.ID))
		ids = append(ids, s.ID)
	}
	appMu.Lock()
	appSessUnlocked().ListIDs = ids
	appMu.Unlock()
	return appMarshal(appState{
		S: "loc", T: appTitle("Locations", "Локации"),
		Lines: strings.Join(lines, "\n"),
		IDs:   strings.Join(ids, ","),
		Acts:  "loc_home,loc_off",
		H:     appTitle("# site rooms · add home/office", "# комнаты · добавить дом/офис"),
	})
}

func buildAppEdgePolicyCard(id string) (string, error) {
	p, err := edge.GetDevicePolicy(id)
	if err != nil {
		return appMarshal(appState{S: "edgepol", T: id, B: err.Error()})
	}
	return appMarshal(appState{
		S: "edgepol", T: id,
		Rows: [][2]string{
			{"inet", fmt.Sprintf("%v", p.AllowInternet)},
			{"mode", p.ServicesMode},
			{"svc", fmt.Sprintf("%d", len(p.Services))},
		},
		Acts: "ep_inet,ep_mode",
		H:    id,
	})
}

func handleAppGapsAct(token string, q *widgetQuery, act, focus string) bool {
	switch act {
	case "pr_new":
		appPresetOK = true
		appPresetDraft = presetDraft{AllowInternet: true, ServicesMode: "list"}
		cd, _ := buildAppPresetDraft()
		appPushState(token, q, cd)
		return true
	case "mt_list":
		out := runND("mtls", "list")
		cd, _ := appMarshal(appState{S: "mtls", T: "mTLS", Lines: appLines(out), Acts: "mt_list,mt_rot,mt_roll"})
		appPushState(token, q, cd)
		return true
	case "mt_rot":
		if q.Chat == nil {
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_mtls_rotate",
			appTitle("Send node id to rotate cert", "Пришлите id ноды для rotate"))
		cd, _ := appMarshal(appState{S: "mtls", T: "mTLS", B: appTitle("waiting node id…", "ждём id…"), Acts: "mt_list,mt_rot,mt_roll"})
		appPushState(token, q, cd)
		return true
	case "mt_roll":
		out := runND("mtls", "rollover-status")
		cd, _ := appMarshal(appState{S: "mtls", T: "mTLS", Lines: appLines(out), Acts: "mt_list,mt_rot,mt_roll"})
		appPushState(token, q, cd)
		return true
	case "sec_exp":
		out := runND("secondary", "export", "-o", "/tmp/nd-secondary-bundle.json", "--sni", "ya.ru")
		b, err := os.ReadFile("/tmp/nd-secondary-bundle.json")
		msg := out
		if err == nil {
			msg = string(b)
		}
		cd, _ := appMarshal(appState{S: "secops", T: "Secondary", Lines: appLines(msg), Acts: "sec_exp,sec_ol,sec_enr,sec_sync"})
		appPushState(token, q, cd)
		return true
	case "sec_ol":
		msg := formatRelayOneline()
		cd, _ := appMarshal(appState{S: "secops", T: "Oneline", Lines: widgetTruncate(stripTags(msg), 500), Acts: "sec_exp,sec_ol,sec_enr,sec_sync"})
		appPushState(token, q, cd)
		return true
	case "sec_enr":
		if q.Chat == nil {
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_relay_host", T("enroll_ip"))
		cd, _ := appMarshal(appState{S: "secops", T: "Enroll", B: appTitle("send IP…", "пришлите IP…"), Acts: "sec_exp,sec_ol,sec_enr,sec_sync"})
		appPushState(token, q, cd)
		return true
	case "sec_sync":
		out := runND("secondary", "sync")
		cd, _ := appMarshal(appState{S: "secops", T: "Secondary", Lines: appLines(out), Acts: "sec_exp,sec_ol,sec_enr,sec_sync"})
		appPushState(token, q, cd)
		return true
	case "er_reg":
		if q.Chat == nil {
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_edge_register", T("edge_register_prompt"))
		cd, _ := appMarshal(appState{S: "edgelife", T: "Edge", B: appTitle("waiting register…", "ждём register…"), Acts: "er_reg,er_bind,er_apply,er_site,er_rec"})
		appPushState(token, q, cd)
		return true
	case "er_bind":
		if q.Chat == nil {
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_edge_bind_dev", T("bind_prompt"))
		cd, _ := appMarshal(appState{S: "edgelife", T: "Edge", B: appTitle("waiting bind…", "ждём bind…"), Acts: "er_reg,er_bind,er_apply,er_site,er_rec"})
		appPushState(token, q, cd)
		return true
	case "er_apply":
		if q.Chat == nil {
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_edge_apply", T("apply_prompt"))
		cd, _ := appMarshal(appState{S: "edgelife", T: "Edge", B: appTitle("waiting apply…", "ждём apply…"), Acts: "er_reg,er_bind,er_apply,er_site,er_rec"})
		appPushState(token, q, cd)
		return true
	case "er_site":
		if q.Chat == nil {
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_edge_set_site", T("edge_set_site_prompt"))
		cd, _ := appMarshal(appState{S: "edgelife", T: "Edge", B: appTitle("waiting site…", "ждём site…"), Acts: "er_reg,er_bind,er_apply,er_site,er_rec"})
		appPushState(token, q, cd)
		return true
	case "er_rec":
		if focus == "" {
			out := runND("edge", "recovery", "list")
			cd, _ := appMarshal(appState{S: "edgelife", T: "Recovery", Lines: appLines(out), Acts: "er_reg,er_bind,er_apply,er_site,er_rec"})
			appPushState(token, q, cd)
			return true
		}
		out := runND("edge", "recovery", focus)
		cd, _ := appMarshal(appState{S: "edgelife", T: focus, Lines: appLines(out), Acts: "er_reg,er_bind,er_apply,er_site,er_rec"})
		appPushState(token, q, cd)
		return true
	case "loc_home", "loc_off":
		kind := "home"
		if act == "loc_off" {
			kind = "office"
		}
		_, err := sites.Upsert(sites.Site{ID: kind, Name: kind})
		msg := "ok " + kind
		if err != nil {
			msg = err.Error()
		}
		cd, _ := buildAppLoc()
		_ = msg
		appPushState(token, q, cd)
		return true
	case "ep_inet":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		p, err := edge.GetDevicePolicy(focus)
		if err != nil {
			cd, _ := appMarshal(appState{S: "edgepol", T: focus, B: err.Error()})
			appPushState(token, q, cd)
			return true
		}
		p = applyPolicyToggle(p, "inet:x")
		_ = edge.SetDevicePolicy(focus, p, "tg")
		cd, _ := buildAppEdgePolicyCard(focus)
		appPushState(token, q, cd)
		return true
	case "ep_mode":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		p, err := edge.GetDevicePolicy(focus)
		if err != nil {
			cd, _ := appMarshal(appState{S: "edgepol", T: focus, B: err.Error()})
			appPushState(token, q, cd)
			return true
		}
		p = applyPolicyToggle(p, "mode")
		_ = edge.SetDevicePolicy(focus, p, "tg")
		cd, _ := buildAppEdgePolicyCard(focus)
		appPushState(token, q, cd)
		return true
	case "es_fmt":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		appSetConfirmTarget(focus)
		cd, _ := appMarshal(appState{
			S: "edgestor", T: appTitle("Format+restore?", "Format+restore?"),
			B: focus, Acts: "y_fmt",
			H: appTitle("destructive: format USB then restore", "опасно: format USB затем restore"),
		})
		appPushState(token, q, cd)
		return true
	case "y_fmt":
		target := appGetConfirmTarget()
		if target == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		appSetConfirmTarget("")
		_ = edge.EnqueueCmd(target, "storage_init_usb", "format=yes")
		cid := edge.EnqueueCmd(target, "storage_restore_remote", "")
		cd, _ := appMarshal(appState{S: "edgestor", T: target, B: "fmt+restore " + cid, Acts: "es_bak,es_rst,es_list,es_mute,es_fmt"})
		appPushState(token, q, cd)
		return true
	case "et_fb":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_, err := edge.SetTemplateVPN(focus, map[string]any{"fallback": "direct"})
		msg := "fallback=direct"
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "edgetpl", T: focus, B: msg, Acts: "et_on,et_off,et_dns,et_mode,et_fb"})
		appPushState(token, q, cd)
		return true
	case "et_soft":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_, err := edge.SetTemplateVPN(focus, map[string]any{"soft": true})
		msg := "soft=on"
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "edgetpl", T: focus, B: msg, Acts: "et_on,et_off,et_dns,et_mode,et_fb,et_soft"})
		appPushState(token, q, cd)
		return true
	}
	return false
}
