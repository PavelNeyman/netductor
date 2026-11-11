package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/fleet"
)

// Shell kind labels (dual info slot metadata only — not SPA multi-HTML).
// Historical note: SPA hub↔spark message swap was removed; revive only if zlib
// forces a second HTML document (see docs/PLAN-TG-WIDGET-UI-V2.md).
const (
	shellHub   = "hub"
	shellSpark = "spark" // dual info: metrics series present / preferred
)

// appPushStateSingle: one SPA message, one HTML. Edit chat_data in place;
// on edit failure resend once and drop the old message id.
func appPushStateSingle(token string, q *widgetQuery, cd string) {
	if q == nil {
		return
	}
	answerWidgetQuery(token, q.ID, cd)
	if q.Chat == nil {
		return
	}
	chat := q.Chat.ID
	lastAppMu.Lock()
	mid := lastAppMsgID[chat]
	if mid == 0 && q.MessageID != 0 {
		mid = q.MessageID
	}
	lastAppMu.Unlock()

	if mid != 0 {
		if err := editWidgetData(token, chat, mid, cd); err == nil {
			setLastAppMsg(chat, mid)
			return
		}
		fmt.Fprintln(os.Stderr, "editMessageWidgetData:", mid, "→ resend")
	}

	html := appWidgetHTML()
	newID, err := sendWidgetMessage(token, chat, html, cd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sendWidget SPA:", err)
		return
	}
	if mid > 0 && mid != newID {
		_ = deleteMessage(token, chat, mid)
	}
	setLastAppMsg(chat, newID)
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
