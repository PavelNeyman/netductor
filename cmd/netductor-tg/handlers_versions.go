package main

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/stack"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

// versionSelect holds operator multi-select for apply (in-memory, primary session).
var (
	verSelMu sync.Mutex
	verSel   = map[string]bool{"primary": true, "secondary": false}
)

func verSelGet() (pri, sec bool) {
	verSelMu.Lock()
	defer verSelMu.Unlock()
	return verSel["primary"], verSel["secondary"]
}

func verSelToggle(key string) {
	verSelMu.Lock()
	defer verSelMu.Unlock()
	verSel[key] = !verSel[key]
}

func verSelSetAll(on bool) {
	verSelMu.Lock()
	defer verSelMu.Unlock()
	verSel["primary"] = on
	verSel["secondary"] = on
}

func mark(on bool) string {
	if on {
		return "✅"
	}
	return "⬜"
}

func formatVersionsHTML(force bool) string {
	ru := getLang() != "en"
	nl := string([]byte{10})
	d := fleet.Build()
	st := stack.Collect()
	pri, sec := verSelGet()

	var b strings.Builder
	if ru {
		b.WriteString("🔄 <b>Версии</b>" + nl)
		b.WriteString("<i>легенда: 🟢 актуально · 🆕 новее · ✅/⬜ выбор для Apply</i>" + nl + nl)
	} else {
		b.WriteString("🔄 <b>Versions</b>" + nl)
		b.WriteString("<i>legend: 🟢 current · 🆕 newer · ✅/⬜ select for Apply</i>" + nl + nl)
	}

	latest := ""
	if force {
		if t, err := ndupdate.LatestReleaseTagForce(); err == nil {
			latest = t
		}
	} else {
		latest = d.LatestRelease
		if latest == "" {
			if t, err := ndupdate.LatestReleaseTag(); err == nil {
				latest = t
			}
		}
	}

	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>#</th><th>node</th><th>now</th><th>latest</th><th>sel</th></tr>" + nl)
	local := d.PrimaryVersion
	if local == "" {
		local = ndver.Running()
	}
	stIcon := "🟢"
	if latest != "" && ndupdate.Newer(latest, local) {
		stIcon = "🆕"
	}
	b.WriteString(fmt.Sprintf("<tr><td>1</td><td>primary</td><td><code>%s</code> %s</td><td><code>%s</code></td><td>%s</td></tr>"+nl,
		esc(local), stIcon, esc(orDash(latest)), mark(pri)))

	secVer := "—"
	secOnline := false
	for _, n := range d.Secondaries {
		if n.Version != "" {
			secVer = n.Version
		}
		if n.Online {
			secOnline = true
		}
	}
	if secVer == "—" {
		for _, dev := range secondary.List() {
			if dev.Version != "" {
				secVer = dev.Version
			}
			if secondary.Online(dev, 2*time.Minute) {
				secOnline = true
			}
		}
	}
	st2 := "🟢"
	if latest != "" && secVer != "—" && ndupdate.Newer(latest, secVer) {
		st2 = "🆕"
	}
	if !secOnline && secVer != "—" {
		st2 = "🔴"
	}
	b.WriteString(fmt.Sprintf("<tr><td>2</td><td>secondary</td><td><code>%s</code> %s</td><td><code>%s</code></td><td>%s</td></tr>"+nl,
		esc(secVer), st2, esc(orDash(latest)), mark(sec)))
	b.WriteString("</table>" + nl + nl)

	// stack summary one line
	b.WriteString("<table bordered striped compact>" + nl)
	if ru {
		b.WriteString("<tr><th>стек</th><th>значение</th></tr>" + nl)
	} else {
		b.WriteString("<tr><th>stack</th><th>value</th></tr>" + nl)
	}
	b.WriteString(fmt.Sprintf("<tr><td>release</td><td><code>%s</code></td></tr>"+nl, esc(st.Release)))
	b.WriteString(fmt.Sprintf("<tr><td>prev</td><td><code>%s</code></td></tr>"+nl, esc(orDash(st.Prev))))
	tok := ndupdate.GetTokenStatus()
	tokS := "❌"
	if tok.Configured {
		tokS = "✅"
	}
	b.WriteString(fmt.Sprintf("<tr><td>github token</td><td>%s</td></tr>"+nl, tokS))
	b.WriteString("</table>" + nl + nl)
	b.WriteString(formatUpdateJobsSnippet(ru, nl))
	if ru {
		b.WriteString("<i>Apply — выбранные. Primary: stack schedule. Secondary: очередь. Desired ≠ running до health.</i>" + nl)
	} else {
		b.WriteString("<i>Apply — selected. Primary: stack schedule. Secondary: queue. Desired ≠ running until health.</i>" + nl)
	}

	// body buttons
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" data="m:ver:toggle:primary">1 primary</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" data="m:ver:toggle:secondary">2 secondary</tg-button>`)
	b.WriteString(`</tg-button-row>`)
	b.WriteString(`<tg-button-row align="left">`)
	if ru {
		b.WriteString(`<tg-button type="callback_data" data="m:ver:all">✅ все</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" data="m:ver:none">⬜ сброс</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="primary" data="m:ver:apply">⬇️ Apply</tg-button>`)
	} else {
		b.WriteString(`<tg-button type="callback_data" data="m:ver:all">✅ all</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" data="m:ver:none">⬜ none</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="primary" data="m:ver:apply">⬇️ Apply</tg-button>`)
	}
	b.WriteString(`</tg-button-row>`)
	b.WriteString(`<tg-button-row align="left">`)
	if ru {
		b.WriteString(`<tg-button type="callback_data" data="m:ver:refresh">🔄 обновить с GitHub</tg-button>`)
	} else {
		b.WriteString(`<tg-button type="callback_data" data="m:ver:refresh">🔄 refresh GitHub</tg-button>`)
	}
	b.WriteString(`<tg-button type="callback_data" data="m:stack">🧱 Stack</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" data="m:stack:rollback">↩️ Rollback</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" data="m:ver:jobs">📋 jobs</tg-button>`)
	b.WriteString(`</tg-button-row>`)
	return b.String()
}

func formatUpdateJobsSnippet(ru bool, nl string) string {
	var b strings.Builder
	b.WriteString("<b>Job registry</b>" + nl)
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>comp</th><th>state</th><th>target</th><th>err</th></tr>" + nl)
	anyJ := false
	for _, c := range []ndupdate.ComponentID{ndupdate.ComponentNode, ndupdate.ComponentTG, ndupdate.ComponentAgent} {
		j, ok := ndupdate.DefaultHostRegistry.Get(c, "local")
		if !ok {
			continue
		}
		anyJ = true
		_, disp := ndupdate.UserFacingVersionLabel(j.TargetVersion, j.State == ndupdate.StateSucceeded, j.TargetVersion)
		errMsg := j.LastError
		if len(errMsg) > 40 {
			errMsg = errMsg[:40] + "…"
		}
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td></tr>"+nl,
			esc(string(c)), esc(string(j.State)), esc(disp), esc(errMsg)))
	}
	if !anyJ {
		if ru {
			b.WriteString(`<tr><td colspan="4"><i>нет job</i></td></tr>` + nl)
		} else {
			b.WriteString(`<tr><td colspan="4"><i>no jobs</i></td></tr>` + nl)
		}
	}
	b.WriteString("</table>" + nl + nl)
	return b.String()
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func versionsKeyboard() map[string]any {
	return navKeyboard("m:tools", parentTools())
}

func handleVersionsCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	switch {
	case data == "m:ver:refresh":
		reply(token, chat, msgID, formatVersionsHTML(true), versionsKeyboard())
	case data == "m:versions" || data == "m:updates":
		reply(token, chat, msgID, formatVersionsHTML(false), versionsKeyboard())
	case data == "m:ver:toggle:primary":
		verSelToggle("primary")
		reply(token, chat, msgID, formatVersionsHTML(false), versionsKeyboard())
	case data == "m:ver:toggle:secondary":
		verSelToggle("secondary")
		reply(token, chat, msgID, formatVersionsHTML(false), versionsKeyboard())
	case data == "m:ver:all":
		verSelSetAll(true)
		reply(token, chat, msgID, formatVersionsHTML(false), versionsKeyboard())
	case data == "m:ver:none":
		verSelSetAll(false)
		reply(token, chat, msgID, formatVersionsHTML(false), versionsKeyboard())
	case data == "m:ver:jobs":
		reply(token, chat, msgID, formatVersionsHTML(false), versionsKeyboard())
	case data == "m:ver:apply":
		pri, sec := verSelGet()
		if !pri && !sec {
			msg := "❌ nothing selected"
			if ru {
				msg = "❌ ничего не выбрано"
			}
			reply(token, chat, msgID, msg, versionsKeyboard())
			return
		}
		if j, ok := ndupdate.DefaultHostRegistry.Get(ndupdate.ComponentNode, "local"); ok {
			switch j.State {
			case ndupdate.StateSucceeded, ndupdate.StateFailed, ndupdate.StateRolledBack, ndupdate.StateCancelled, ndupdate.StateIdle:
			default:
				msg := "❌ node job active: " + string(j.State)
				if ru {
					msg = "❌ job node активен: " + string(j.State)
				}
				reply(token, chat, msgID, msg, versionsKeyboard())
				return
			}
		}
		tag, err := ndupdate.LatestReleaseTag()
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), versionsKeyboard())
			return
		}
		msg := "⬇️ Confirm apply <code>" + esc(tag) + "</code>?"
		if ru {
			msg = "⬇️ Подтвердить apply <code>" + esc(tag) + "</code>?"
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("✅ "+tag, "m:ver:apply:go:"+tag, "primary")},
			{btn("⬅️", "m:versions", ""), btn(T("main_menu"), "m:menu", "")},
		}}
		reply(token, chat, msgID, msg, kb)
	case strings.HasPrefix(data, "m:ver:apply:go:"):
		tag := strings.TrimPrefix(data, "m:ver:apply:go:")
		if _, err := ndupdate.ValidReleaseTag(tag); err != nil {
			reply(token, chat, msgID, "❌ bad tag", versionsKeyboard())
			return
		}
		pri, sec := verSelGet()
		if !pri && !sec {
			reply(token, chat, msgID, "❌ nothing selected", versionsKeyboard())
			return
		}
		var lines []string
		if pri {
			if err := exec.Command("/usr/local/bin/netductor", "stack", "schedule", tag).Run(); err != nil {
				lines = append(lines, "primary: ❌ "+err.Error())
			} else {
				lines = append(lines, "primary: ⏳ scheduled "+tag)
			}
		}
		if sec {
			n := 0
			for _, d := range secondary.List() {
				if secondary.Online(d, 2*time.Minute) {
					if err := secondary.EnqueueCmd(d.ID, "upgrade:"+tag); err == nil {
						n++
					}
				}
			}
			lines = append(lines, fmt.Sprintf("secondary: queued=%d %s", n, tag))
		}
		body := "⬇️ <b>Apply</b>" + "\n" + "<pre>" + esc(strings.Join(lines, "\n")) + "</pre>"
		if ru {
			body += "\n<i>Не жать Apply повторно. Running — после health.</i>"
		} else {
			body += "\n<i>Do not re-tap Apply. Running only after health.</i>"
		}
		reply(token, chat, msgID, body, versionsKeyboard())
	default:
		if strings.HasPrefix(data, "m:updates:") {
			handleUpdatesCB(token, chat, msgID, data)
			return
		}
		reply(token, chat, msgID, formatVersionsHTML(false), versionsKeyboard())
	}
}
