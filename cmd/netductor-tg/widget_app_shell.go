package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/PavelNeyman/netductor/internal/fleet"
)

// Multi-shell SPA: HTML is immutable after sendWidget, so a different layout
// (e.g. status with canvas sparkline) requires a new message + delete of the old.
// Kind "hub" = default navigation; "spark" = status metrics with canvas.

const (
	shellHub   = "hub"
	shellSpark = "spark"
)

var (
	lastAppShell   = map[int64]string{} // chat → shell kind
	lastAppShellMu sync.Mutex
)

func shellKindForScreen(screen string) string {
	switch screen {
	case "status":
		return shellSpark
	default:
		return shellHub
	}
}

func appHTMLForShell(kind string) string {
	if kind == shellSpark {
		return appWidgetHTMLSpark()
	}
	return appWidgetHTMLHub()
}

// appPushStateSmart edits in place when the shell matches; otherwise swaps the message.
func appPushStateSmart(token string, q *widgetQuery, screen, cd string) {
	if q == nil {
		return
	}
	kind := shellKindForScreen(screen)
	answerWidgetQuery(token, q.ID, cd)
	if q.Chat == nil {
		return
	}
	chat := q.Chat.ID
	lastAppShellMu.Lock()
	cur := lastAppShell[chat]
	lastAppShellMu.Unlock()

	lastAppMu.Lock()
	mid := lastAppMsgID[chat]
	if mid == 0 && q.MessageID != 0 {
		mid = q.MessageID
	}
	lastAppMu.Unlock()

	// Same shell → edit chat_data only (smooth).
	if cur == kind || (cur == "" && kind == shellHub && mid != 0 && q.MessageID != 0) {
		if mid != 0 {
			_, err := apiPostWidget(token, "editMessageWidgetData", map[string]any{
				"chat_id": chat, "message_id": mid, "chat_data": cd,
			})
			if err == nil {
				lastAppShellMu.Lock()
				lastAppShell[chat] = kind
				lastAppShellMu.Unlock()
				lastAppMu.Lock()
				lastAppMsgID[chat] = mid
				lastAppMu.Unlock()
				return
			}
			fmt.Fprintln(os.Stderr, "editMessageWidgetData:", err)
		}
	}

	// Shell change or edit failed → send new widget, delete previous.
	html := appHTMLForShell(kind)
	raw, err := apiPostWidget(token, "sendWidget", map[string]any{
		"chat_id": chat, "html": html, "chat_data": cd, "allow_queries": true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sendWidget swap:", err)
		// fallback: try edit on current message
		if q.MessageID != 0 {
			_, _ = apiPostWidget(token, "editMessageWidgetData", map[string]any{
				"chat_id": chat, "message_id": q.MessageID, "chat_data": cd,
			})
		}
		return
	}
	var wr struct {
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	newID := 0
	if json.Unmarshal(raw, &wr) == nil {
		newID = wr.Result.MessageID
	}
	if newID > 0 {
		lastAppMu.Lock()
		lastAppMsgID[chat] = newID
		lastAppMu.Unlock()
		lastAppShellMu.Lock()
		lastAppShell[chat] = kind
		lastAppShellMu.Unlock()
	}
	// delete old message for smooth single-card feel
	old := mid
	if old == 0 {
		old = q.MessageID
	}
	if old > 0 && old != newID {
		_ = deleteMessage(token, chat, old)
	}
}

// parseScreenFromChatData reads "s" field for shell selection.
func parseScreenFromChatData(cd string) string {
	var m struct {
		S string `json:"s"`
	}
	if json.Unmarshal([]byte(cd), &m) == nil && m.S != "" {
		return m.S
	}
	return "home"
}

func forgetAppShell(chat int64) {
	lastAppShellMu.Lock()
	delete(lastAppShell, chat)
	lastAppShellMu.Unlock()
}

// split secondary list text into lines for display (id ver on)
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
		ids = append(ids, s.ID)
		if ids[len(ids)-1] == "" {
			ids[len(ids)-1] = name
		}
	}
	return lines, ids
}

func formatGitHubLines() (rows [][2]string, lines string) {
	out := appLines(runND("git", "status"))
	if out == "" {
		out = appLines(runND("git", "sync", "status"))
	}
	// Prefer key rows when formatCmdOut produced key: value
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
