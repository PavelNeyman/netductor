package main

import (
	"fmt"
	"strings"

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
			Lines: widgetTruncate(out, 400),
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
		cd, _ := appMarshal(appState{S: "sshkeys", T: "SSH keys", Lines: widgetTruncate(out, 400), Acts: "sk_on,sk_off,sk_scan"})
		appPushState(token, q, cd)
		return true
	case "sk_off":
		out := runND("ssh-keys", "protect", "off")
		cd, _ := appMarshal(appState{S: "sshkeys", T: "SSH keys", Lines: widgetTruncate(out, 400), Acts: "sk_on,sk_off,sk_scan"})
		appPushState(token, q, cd)
		return true
	case "sk_scan":
		out := runND("ssh-keys", "scan")
		cd, _ := appMarshal(appState{S: "sshkeys", T: "SSH keys", Lines: widgetTruncate(out, 400), Acts: "sk_on,sk_off,sk_scan"})
		appPushState(token, q, cd)
		return true

	// --- certs / firewall / aptguard ---
	case "cert_ren":
		out := runND("certs", "renew")
		if strings.TrimSpace(out) == "" {
			out = runND("le", "renew")
		}
		cd, _ := appMarshal(appState{S: "certs", T: appTitle("Certs", "Сертификаты"), Lines: widgetTruncate(out, 450), Acts: "cert_ren"})
		appPushState(token, q, cd)
		return true
	case "fw_apply":
		out := runND("firewall", "apply")
		cd, _ := appMarshal(appState{S: "fw", T: appTitle("Firewall", "Firewall"), Lines: widgetTruncate(out, 450), Acts: "fw_apply"})
		appPushState(token, q, cd)
		return true
	case "ag_st":
		out := runND("aptguard", "status")
		cd, _ := appMarshal(appState{S: "aptguard", T: "Aptguard", Lines: widgetTruncate(out, 450), Acts: "ag_st,ag_prev,ag_scan"})
		appPushState(token, q, cd)
		return true
	case "ag_prev":
		out := runND("aptguard", "preview")
		cd, _ := appMarshal(appState{S: "aptguard", T: "Aptguard", Lines: widgetTruncate(out, 450), Acts: "ag_st,ag_prev,ag_scan"})
		appPushState(token, q, cd)
		return true
	case "ag_scan":
		out := runND("aptguard", "scan")
		cd, _ := appMarshal(appState{S: "aptguard", T: "Aptguard", Lines: widgetTruncate(out, 450), Acts: "ag_st,ag_prev,ag_scan"})
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
		cd, _ := appMarshal(appState{S: "ncard", T: focus, Lines: widgetTruncate(out, 450), Acts: "up,rb,jn,rs,nrn,mx"})
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
		cd, _ := appMarshal(appState{S: "git", T: "Git", Lines: widgetTruncate(msg, 450), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd"})
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
		cd, _ := appMarshal(appState{S: "git", T: "Git", Lines: widgetTruncate(b.String(), 450), Acts: "gsync,gship,gsall,gbuild,gmirror,gjobs,gadd"})
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
		appListIDs = ids
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
		cd, _ := appMarshal(appState{S: "gjobs", T: focus, Lines: widgetTruncate(msg, 400), Acts: "gpause,gresume,grm"})
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
		cd, _ := appMarshal(appState{S: "edgeluci", T: "Edge LuCI", Lines: widgetTruncate(out, 400), Acts: "el_1,el_4,el_24,el_72,el_off"})
		appPushState(token, q, cd)
		return true
	case "el_ext":
		out := runND("edge", "luci", "extend", "24h")
		if focus != "" {
			out = runND("edge", "luci", "extend", focus, "24h")
		}
		cd, _ := appMarshal(appState{S: "edgeluci", T: "Edge LuCI", Lines: widgetTruncate(out, 400), Acts: "el_1,el_4,el_24,el_72,el_ext,el_off"})
		appPushState(token, q, cd)
		return true

	// --- edge storage / policy / template status ---
	case "es_st":
		out := runND("edge", "storage", "status")
		cd, _ := appMarshal(appState{S: "edgestor", T: "Edge storage", Lines: widgetTruncate(out, 450)})
		appPushState(token, q, cd)
		return true
	case "ep_st":
		out := runND("edge", "policy", "status")
		cd, _ := appMarshal(appState{S: "edgepol", T: "Edge policy", Lines: widgetTruncate(out, 450)})
		appPushState(token, q, cd)
		return true
	case "et_st":
		out := runND("edge", "template", "list")
		cd, _ := appMarshal(appState{S: "edgetpl", T: "Edge templates", Lines: widgetTruncate(out, 450)})
		appPushState(token, q, cd)
		return true

	// --- guest facts ---
	case "eg_facts":
		if focus == "" {
			answerWidgetQuery(token, q.ID, `{"err":"nofocus"}`)
			return true
		}
		out := runND("edge", "guest", "facts", focus)
		cd, _ := appMarshal(appState{S: "edgeguest", T: focus, Lines: widgetTruncate(out, 450), Acts: "eg_gr,eg_rv,eg_facts"})
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
	appListIDs = ids
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
		Lines: widgetTruncate(out, 400),
		Acts:  "sk_on,sk_off,sk_scan",
		H:     appTitle("protect on/off · scan", "защита on/off · scan"),
	})
}

func buildAppAptguard() (string, error) {
	out := runND("aptguard", "status")
	return appMarshal(appState{
		S: "aptguard", T: "Aptguard",
		Lines: widgetTruncate(out, 400),
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
		Lines: widgetTruncate(out, 400),
		Acts:  "cert_ren",
	})
}

func buildAppFW() (string, error) {
	out := runND("firewall", "status")
	return appMarshal(appState{
		S: "fw", T: "Firewall",
		Lines: widgetTruncate(out, 400),
		Acts:  "fw_apply",
	})
}

func buildAppEdgeLuciFull() (string, error) {
	out := strings.TrimSpace(runND("edge", "luci", "status"))
	return appMarshal(appState{
		S: "edgeluci", T: "Edge LuCI",
		Lines: widgetTruncate(out, 350),
		Acts:  "el_1,el_4,el_24,el_72,el_ext,el_off",
		H:     appTitle("duration presets", "пресеты длительности"),
	})
}

