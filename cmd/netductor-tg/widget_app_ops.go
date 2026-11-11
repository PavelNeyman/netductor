package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	gitstore "github.com/PavelNeyman/netductor/internal/git"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

// handleAppOpsAct covers remaining classic ops not in the core SPA switch.
func handleAppOpsAct(token string, q *widgetQuery, act, focus string) bool {
	switch act {
	// --- session presets ---
	case "sess1", "sess4", "sess24", "sess72":
		if q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nochat"}`)
			return true
		}
		h := map[string]string{"sess1": "1", "sess4": "4", "sess24": "24", "sess72": "72"}[act]
		tok := runVPN("session", h)
		fields := strings.Fields(tok)
		copyVal := tok
		if len(fields) > 0 {
			copyVal = fields[0]
		}
		kb := map[string]any{
			"inline_keyboard": [][]map[string]any{
				{btnCopy(T("copy_token"), copyVal)},
			},
		}
		sendEphemeralHTML(token, q.Chat.ID, formatSessionHTML(tok), kb, 120)
		cd, _ := appMarshal(appState{
			S: "session", T: appTitle("Session", "Сессия"),
			B: appTitle("token sent (auto-delete ~2m)", "токен отправлен (удалится ~2м)"),
			Acts: "sess1,sess4,sess24,sess72,sessx",
		})
		appPushState(token, q, cd)
		return true
	case "sessx":
		if q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nochat"}`)
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_session_hours", T("session_prompt"))
		cd, _ := appMarshal(appState{
			S: "session", T: appTitle("Session", "Сессия"),
			B: appTitle("send hours in chat…", "пришлите часы в чат…"),
			Acts: "sess1,sess4,sess24,sess72,sessx",
		})
		appPushState(token, q, cd)
		return true

	// --- sessions revoke ---
	case "srev":
		out := runND("session", "revoke-all")
		if strings.TrimSpace(out) == "" {
			out = runVPN("session", "revoke-all")
		}
		cd, _ := appMarshal(appState{
			S: "sessions", T: appTitle("Sessions", "Сессии"),
			Lines: appLines(out),
			Acts:  "srev",
		})
		appPushState(token, q, cd)
		return true

	// --- canary toggle on user ---
	case "cy":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		on, err := vpn.ToggleCanary(focus)
		msg := "canary off"
		if on {
			msg = "canary on"
		}
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{
			S: "ucard", T: focus, B: msg,
			Acts: "en,di,lk,sb,q50,q200,q0,qn,rn,qr,kd,ku,sp,cy,rv",
		})
		appPushState(token, q, cd)
		return true

	// --- canary hub toggle by list # already via o: ---

	// --- sshkeys protect ---
	case "sk_on":
		out := runND("ssh-keys", "protect", "on")
		cd, _ := appMarshal(appState{S: "sshkeys", T: "SSH keys", Lines: appLines(out), Acts: "sk_on,sk_off,sk_scan"})
		appPushState(token, q, cd)
		return true
	case "sk_off":
		out := runND("ssh-keys", "protect", "off")
		cd, _ := appMarshal(appState{S: "sshkeys", T: "SSH keys", Lines: appLines(out), Acts: "sk_on,sk_off,sk_scan"})
		appPushState(token, q, cd)
		return true
	case "sk_scan":
		out := runND("ssh-keys", "scan")
		cd, _ := appMarshal(appState{S: "sshkeys", T: "SSH keys", Lines: appLines(out), Acts: "sk_on,sk_off,sk_scan"})
		appPushState(token, q, cd)
		return true

	// --- certs / firewall / aptguard ---
	case "cert_ren":
		out := runND("certs", "renew")
		if strings.TrimSpace(out) == "" {
			out = runND("le", "renew")
		}
		cd, _ := appMarshal(appState{S: "certs", T: appTitle("Certs", "Сертификаты"), Lines: appLines(out), Acts: "cert_ren"})
		appPushState(token, q, cd)
		return true
	case "fw_apply":
		out := runND("firewall", "apply")
		cd, _ := appMarshal(appState{S: "fw", T: appTitle("Firewall", "Firewall"), Lines: appLines(out), Acts: "fw_apply"})
		appPushState(token, q, cd)
		return true
	case "ag_st":
		out := runND("aptguard", "status")
		cd, _ := appMarshal(appState{S: "aptguard", T: "Aptguard", Lines: appLines(out), Acts: "ag_st,ag_prev,ag_scan"})
		appPushState(token, q, cd)
		return true
	case "ag_prev":
		out := runND("aptguard", "preview")
		cd, _ := appMarshal(appState{S: "aptguard", T: "Aptguard", Lines: appLines(out), Acts: "ag_st,ag_prev,ag_scan"})
		appPushState(token, q, cd)
		return true
	case "ag_scan":
		out := runND("aptguard", "scan")
		cd, _ := appMarshal(appState{S: "aptguard", T: "Aptguard", Lines: appLines(out), Acts: "ag_st,ag_prev,ag_scan"})
		appPushState(token, q, cd)
		return true

	// --- node rename ---
	case "nrn":
		if focus == "" || q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_node_rename:"+focus,
			appTitle("Send new node name for ", "Новое имя ноды для ")+"<code>"+esc(focus)+"</code>:")
		cd, _ := appMarshal(appState{
			S: "ncard", T: focus,
			B: appTitle("waiting for name in chat…", "ждём имя в чате…"),
			Acts: "up,rb,jn,rs,nrn",
		})
		appPushState(token, q, cd)
		return true

	// --- metrics only ---
	case "mx":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := enqueueNodeCmd(focus, "metrics")
		cd, _ := appMarshal(appState{S: "ncard", T: focus, Lines: appLines(out), Acts: "up,rb,jn,rs,nrn,mx"})
		appPushState(token, q, cd)
		return true

	// --- git bulk ---
	case "gbuild":
		res, okN, failN, skipN := gitstore.BuildAll(true)
		msg := fmt.Sprintf("build-all ok=%d fail=%d skip=%d\n", okN, failN, skipN)
		for _, r := range res {
			st := "ok"
			if !r.OK { st = "fail" }
			if r.Skipped { st = "skip" }
			msg += r.Name + ": " + st + "\n"
			if len(msg) > 400 {
				break
			}
		}
		cd, _ := appMarshal(appState{S: "git", T: "Git", Lines: appLines(msg), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd"})
		appPushState(token, q, cd)
		return true
	case "gmirror":
		list, err := gitstore.LoadProjects()
		var b strings.Builder
		if err != nil {
			b.WriteString(err.Error())
		} else {
			for _, pr := range list {
				out, e := gitstore.MirrorPush(pr.Name)
				if e != nil {
					b.WriteString(pr.Name + ": " + e.Error() + "\n")
				} else {
					b.WriteString(pr.Name + ": ok\n")
				}
				_ = out
				if b.Len() > 400 {
					break
				}
			}
		}
		cd, _ := appMarshal(appState{S: "git", T: "Git", Lines: appLines(b.String()), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd"})
		appPushState(token, q, cd)
		return true
	case "gjobs":
		jobs, err := gitstore.ListShipJobs()
		var lines []string
		var ids []string
		if err != nil {
			lines = []string{err.Error()}
		} else {
			for i, j := range jobs {
				if i >= 10 {
					break
				}
				lines = append(lines, fmt.Sprintf("%d %s %s", i+1, j.Project, j.State))
				ids = append(ids, j.ID)
			}
		}
		appMu.Lock()
		appSessUnlocked().ListIDs = ids
		appMu.Unlock()
		cd, _ := appMarshal(appState{
			S: "gjobs", T: appTitle("Ship jobs", "Ship jobs"),
			Lines: strings.Join(lines, "\n"),
			IDs:   strings.Join(ids, ","),
			Acts:  "gjobs",
			H:     appTitle("# pause/resume/rm via acts on card", "# pause/resume/rm"),
		})
		appPushState(token, q, cd)
		return true
	case "gpause":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_ = gitstore.MarkShipWaiting(focus, "paused via TG")
		cd, _ := appMarshal(appState{S: "gjobs", T: focus, B: "paused", Acts: "gresume,grm"})
		appPushState(token, q, cd)
		return true
	case "gresume":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_, logOut, err := gitstore.ResumeShip(focus)
		msg := logOut
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "gjobs", T: focus, Lines: appLines(msg), Acts: "gpause,gresume,grm"})
		appPushState(token, q, cd)
		return true
	case "grm":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_ = gitstore.RemoveShipJob(focus)
		cd, _ := appMarshal(appState{S: "gjobs", T: "Ship jobs", B: "removed " + focus, Acts: "gjobs"})
		appPushState(token, q, cd)
		return true
	case "gadd":
		if q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nochat"}`)
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_git_add",
			appTitle("Send: <code>name https://github.com/org/repo</code>", "Пришлите: <code>name https://github.com/org/repo</code>"))
		cd, _ := appMarshal(appState{
			S: "git", T: "Git",
			B: appTitle("waiting for name + URL…", "ждём name + URL…"),
			Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd",
		})
		appPushState(token, q, cd)
		return true
	case "gmig":
		if q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nochat"}`)
			return true
		}
		beginTextInput(token, q.Chat.ID, "wait_git_migrate",
			appTitle("Send GitHub <code>owner</code> to migrate repos", "Пришлите GitHub <code>owner</code> для migrate"))
		cd, _ := appMarshal(appState{
			S: "git", T: "Git",
			B: appTitle("waiting for owner…", "ждём owner…"),
			Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd,gmig",
		})
		appPushState(token, q, cd)
		return true

	// --- LuCI durations (global, no per-device focus needed for simple path) ---
	case "el_1", "el_4", "el_24", "el_72":
		dur := map[string]string{"el_1": "1h", "el_4": "4h", "el_24": "24h", "el_72": "72h"}[act]
		out := runND("edge", "luci", "on", dur)
		if focus != "" {
			out = runND("edge", "luci", "on", focus, dur)
		}
		cd, _ := appMarshal(appState{S: "edgeluci", T: "Edge LuCI", Lines: appLines(out), Acts: "el_1,el_4,el_24,el_72,el_off"})
		appPushState(token, q, cd)
		return true
	case "el_ext":
		out := runND("edge", "luci", "extend", "24h")
		if focus != "" {
			out = runND("edge", "luci", "extend", focus, "24h")
		}
		cd, _ := appMarshal(appState{S: "edgeluci", T: "Edge LuCI", Lines: appLines(out), Acts: "el_1,el_4,el_24,el_72,el_ext,el_off"})
		appPushState(token, q, cd)
		return true

	// --- edge storage / policy / template status ---
	case "es_st":
		out := runND("edge", "storage", "status")
		cd, _ := appMarshal(appState{S: "edgestor", T: "Edge storage", Lines: appLines(out)})
		appPushState(token, q, cd)
		return true
	case "ep_st":
		out := runND("edge", "policy", "status")
		cd, _ := appMarshal(appState{S: "edgepol", T: "Edge policy", Lines: appLines(out)})
		appPushState(token, q, cd)
		return true
	case "et_st":
		out := runND("edge", "template", "list")
		cd, _ := appMarshal(appState{S: "edgetpl", T: "Edge templates", Lines: appLines(out)})
		appPushState(token, q, cd)
		return true

	// --- guest facts ---
	case "eg_facts":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := runND("edge", "guest", "facts", focus)
		cd, _ := appMarshal(appState{S: "edgeguest", T: focus, Lines: appLines(out), Acts: "eg_gr,eg_rv,eg_facts"})
		appPushState(token, q, cd)
		return true
	}
	return false
}

func buildAppCanary() (string, error) {
	list := vpn.LoadCanaryUsers()
	st10 := vpn.CollectMismatch(10)
	var lines []string
	var ids []string
	for i, u := range list {
		if i >= 12 {
			break
		}
		lines = append(lines, fmt.Sprintf("%d %s", i+1, u))
		ids = append(ids, u)
	}
	appMu.Lock()
	appSessUnlocked().ListIDs = ids
	appMu.Unlock()
	return appMarshal(appState{
		S: "canary", T: "Canary",
		Rows: [][2]string{
			{"m10", fmt.Sprintf("%d/%d", st10.CanaryTotal, st10.Total)},
		},
		Lines: strings.Join(lines, "\n"),
		IDs:   strings.Join(ids, ","),
		H:     appTitle("# toggle · or canary from user card", "# toggle · или с карточки user"),
	})
}

func buildAppSession() (string, error) {
	return appMarshal(appState{
		S: "session", T: appTitle("Session", "Сессия"),
		B: appTitle("issue operator session token", "выдать session token"),
		Acts: "sess1,sess4,sess24,sess72,sessx",
		H:    appTitle("presets or custom hours", "пресеты или свои часы"),
	})
}

func buildAppSSHKeys() (string, error) {
	out := runND("ssh-keys", "status")
	return appMarshal(appState{
		S: "sshkeys", T: appTitle("SSH keys", "SSH ключи"),
		Lines: appLines(out),
		Acts:  "sk_on,sk_off,sk_scan",
		H:     appTitle("protect on/off · scan", "защита on/off · scan"),
	})
}

func buildAppAptguard() (string, error) {
	out := runND("aptguard", "status")
	return appMarshal(appState{
		S: "aptguard", T: "Aptguard",
		Lines: appLines(out),
		Acts:  "ag_st,ag_prev,ag_scan",
	})
}

func buildAppCerts() (string, error) {
	out := runND("certs", "status")
	if strings.TrimSpace(out) == "" {
		out = runND("le", "status")
	}
	return appMarshal(appState{
		S: "certs", T: appTitle("Certs", "Сертификаты"),
		Lines: appLines(out),
		Acts:  "cert_ren",
	})
}

func buildAppFW() (string, error) {
	out := runND("firewall", "status")
	return appMarshal(appState{
		S: "fw", T: "Firewall",
		Lines: appLines(out),
		Acts:  "fw_apply",
	})
}

func buildAppEdgeLuciFull() (string, error) {
	out := strings.TrimSpace(runND("edge", "luci", "status"))
	return appMarshal(appState{
		S: "edgeluci", T: "Edge LuCI",
		Lines: appLines(out),
		Acts:  "el_1,el_4,el_24,el_72,el_ext,el_off",
		H:     appTitle("duration presets", "пресеты длительности"),
	})
}


// --- residual SPA: storage card, templates, ship log, admin, guest grant text ---

func buildAppEdgeStorage() (string, error) {
	devs := edge.ListDevices()
	var lines, ids []string
	for i, d := range devs {
		if d.DeviceID == "" || i >= 12 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d %s", len(ids)+1, d.DeviceID))
		ids = append(ids, d.DeviceID)
	}
	appMu.Lock()
	appSessUnlocked().ListIDs = ids
	appMu.Unlock()
	st := appState{S: "edgestor", T: "Edge storage", H: appTitle("# device actions", "# устройство")}
	if len(lines) == 0 {
		st.B = appTitle("no edge devices", "нет edge-устройств")
	} else {
		st.Lines = strings.Join(lines, "\n")
		st.IDs = strings.Join(ids, ",")
	}
	return appMarshal(st)
}

func buildAppEdgeTemplates() (string, error) {
	// Reuse classic pick HTML stripped to lines via list devices / template ids
	out := formatEdgeTplPickHTML()
	// Prefer structured list if available
	devs := edge.ListDevices()
	var lines, ids []string
	for i, d := range devs {
		if d.DeviceID == "" || i >= 12 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d %s", len(ids)+1, d.DeviceID))
		ids = append(ids, d.DeviceID)
	}
	appMu.Lock()
	appSessUnlocked().ListIDs = ids
	appMu.Unlock()
	st := appState{S: "edgetpl", T: "Edge templates", H: appTitle("# VPN template", "# VPN шаблон")}
	if len(lines) == 0 {
		st.Lines = widgetTruncate(stripTags(out), 400)
		if st.Lines == "" {
			st.B = appTitle("empty", "пусто")
		}
	} else {
		st.Lines = strings.Join(lines, "\n")
		st.IDs = strings.Join(ids, ",")
	}
	return appMarshal(st)
}

func buildAppAdmin() (string, error) {
	body := T("admin_body")
	// strip simple tags for widget lines
	plain := stripTags(body)
	return appMarshal(appState{
		S: "admin", T: appTitle("Admin", "Админ"),
		Lines: widgetTruncate(plain, 500),
		Acts:  "sess1,sess4,sess24,sess72,sessx",
		H:     appTitle("session token presets below", "пресеты session token"),
	})
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == '<' {
			in = true
			continue
		}
		if r == '>' {
			in = false
			continue
		}
		if !in {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func handleAppResidualAct(token string, q *widgetQuery, act, focus string) bool {
	switch act {
	case "es_bak":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		a := edge.EnqueueCmd(focus, "config_backup", "")
		bb := edge.EnqueueCmd(focus, "storage_backup", "")
		msg := fmt.Sprintf("config=%s storage=%s", a, bb)
		cd, _ := appMarshal(appState{S: "edgestor", T: focus, Lines: msg, Acts: "es_bak,es_rst,es_list,es_mute"})
		appPushState(token, q, cd)
		return true
	case "es_rst":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		cid := edge.EnqueueCmd(focus, "storage_restore_remote", "")
		cd, _ := appMarshal(appState{S: "edgestor", T: focus, B: "restore " + cid, Acts: "es_bak,es_rst,es_list,es_mute"})
		appPushState(token, q, cd)
		return true
	case "es_list":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		msg := formatEdgeStorageListHTML(focus, getLang() != "en")
		cd, _ := appMarshal(appState{S: "edgestor", T: focus, Lines: widgetTruncate(stripTags(msg), 450), Acts: "es_bak,es_rst,es_list,es_mute"})
		appPushState(token, q, cd)
		return true
	case "es_mute":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := runND("edge", "storage", "mute", focus)
		if strings.TrimSpace(out) == "" {
			out = "mute queued/ok"
		}
		cd, _ := appMarshal(appState{S: "edgestor", T: focus, Lines: appLines(out), Acts: "es_bak,es_rst,es_list,es_mute"})
		appPushState(token, q, cd)
		return true
	case "et_on", "et_off":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		v := "0"
		if act == "et_on" {
			v = "1"
		}
		_, err := edge.SetTemplateVPN(focus, map[string]any{"enabled": v == "1"})
		msg := "template enabled=" + v
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "edgetpl", T: focus, B: msg, Acts: "et_on,et_off,et_dns,et_mode"})
		appPushState(token, q, cd)
		return true
	case "et_dns":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_, err := edge.SetTemplateVPN(focus, map[string]any{"dns": "blocky"})
		msg := "dns=blocky"
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "edgetpl", T: focus, B: msg, Acts: "et_on,et_off,et_dns,et_mode"})
		appPushState(token, q, cd)
		return true
	case "et_mode":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		_, err := edge.SetTemplateVPN(focus, map[string]any{"mode": "full"})
		msg := "mode=full"
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "edgetpl", T: focus, B: msg, Acts: "et_on,et_off,et_dns,et_mode"})
		appPushState(token, q, cd)
		return true
	case "glog":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		s, err := gitstore.ReadShipLog(focus, 2000)
		msg := s
		if err != nil {
			msg = err.Error()
		}
		cd, _ := appMarshal(appState{S: "gjobs", T: focus, Lines: appLines(msg), Acts: "gpause,gresume,grm,glog"})
		appPushState(token, q, cd)
		return true
	case "eg_code":
		if focus == "" || q.Chat == nil {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		appMu.Lock()
		appSessUnlocked().Focus = focus
		appMu.Unlock()
		// Prompt + keep device id in chatExtra by encoding "mid|device"
		mid := sendPlainHTML(token, q.Chat.ID, appTitle("Send: <code>CODE minutes</code> for ", "Пришлите: <code>КОД минуты</code> для ")+"<code>"+esc(focus)+"</code>")
		setState(q.Chat.ID, "wait_edgeguest_grant", fmt.Sprintf("%d|%s", mid, focus))
		cd, _ := appMarshal(appState{
			S: "edgeguest", T: focus,
			B: appTitle("waiting CODE minutes in chat…", "ждём КОД минуты в чате…"),
			Acts: "eg_gr,eg_rv,eg_facts,eg_code",
		})
		appPushState(token, q, cd)
		return true
	}
	return false
}
