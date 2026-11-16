package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/fleet"
)

// Shell kinds for SPA multi-HTML (hub lean / spark + canvas) and dual info metadata.
// See docs/PLAN-TG-WIDGET-UI-V2.md — same shell → edit; different → send + delete.
const (
	shellHub   = "hub"
	shellSpark = "spark"
)

// appPushStateSingle: multi-shell SPA.
// Same shell → editMessageWidgetData; different shell or edit fail → sendWidget + delete old.
func appPushStateSingle(token string, q *widgetQuery, cd string) {
	if q == nil {
		return
	}
	if c, e := clipWidgetChatData(cd); e == nil {
		cd = c
	}
	if q.Chat == nil {
		// No chat to edit: still ack with state so client paints from answer.
		answerWidgetQuery(token, q.ID, cd)
		return
	}
	appBindChat(q.Chat.ID)
	chat := q.Chat.ID
	unlock := lockChatPush(chat)
	defer unlock()

	screen := parseScreenFromChatData(cd)
	want := appShellForScreen(screen)

	lastAppMu.Lock()
	mid := lastAppMsgID[chat]
	if mid == 0 && q.MessageID != 0 {
		mid = q.MessageID
	}
	have := lastAppShell[chat]
	lastAppMu.Unlock()

	// Apply message first, then answer — avoids client showing new state while msg is old.
	tryEdit := mid != 0 && (have == want || have == "")
	if tryEdit {
		if err := editWidgetData(token, chat, mid, cd); err == nil {
			setLastAppMsg(chat, mid)
			setLastAppShell(chat, want)
			answerWidgetQuery(token, q.ID, cd)
			return
		}
		fmt.Fprintln(os.Stderr, "editMessageWidgetData:", mid, "→ resend shell", have, "→", want)
	} else if mid != 0 && have != want {
		fmt.Fprintln(os.Stderr, "SPA shell change", have, "→", want, "msg", mid)
	}

	html := appWidgetHTMLFor(want)
	newID, err := sendWidgetMessage(token, chat, html, cd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sendWidget SPA:", err)
		answerWidgetQuery(token, q.ID, `{"err":"push"}`)
		return
	}
	if mid > 0 && mid != newID {
		_ = deleteMessage(token, chat, mid)
	}
	setLastAppMsg(chat, newID)
	setLastAppShell(chat, want)
	answerWidgetQuery(token, q.ID, cd)
}

func parseScreenFromChatData(cd string) string {
	var m struct {
		S string `json:"s"`
	}
	if json.Unmarshal([]byte(cd), &m) == nil && m.S != "" {
		return m.S
	}
	return "home"
}

func formatSecondaryLines() (lines []string, ids []string) {
	d := fleet.Build()
	for i, s := range d.Secondaries {
		if i >= 12 {
			break
		}
		name := s.Name
		if name == "" {
			name = s.ID
		}
		on := "off"
		if s.Online {
			on = "on"
		}
		lines = append(lines, fmt.Sprintf("%d %s %s %s", len(ids)+1, name, s.Version, on))
		id := s.ID
		if id == "" {
			id = name
		}
		ids = append(ids, id)
	}
	return lines, ids
}

func formatGitHubLines() (rows [][2]string, lines string) {
	out := appLines(runND("git", "status"))
	if out == "" {
		out = appLines(runND("git", "sync", "status"))
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i := strings.IndexByte(line, ':'); i > 0 && i < 24 && len(rows) < 8 {
			rows = append(rows, [2]string{strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])})
			continue
		}
	}
	if len(rows) == 0 {
		lines = out
	}
	return rows, lines
}
