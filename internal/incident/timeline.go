package incident

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

type Event struct {
	TS   int64  `json:"ts"`
	Kind string `json:"kind"` // alert | recovery | note
	Key  string `json:"key"`
	Text string `json:"text"`
}

var tlMu sync.Mutex

func timelinePath() string {
	return filepath.Join(paths.StateDir(), "incident_timeline.json")
}

func AppendEvent(kind, key, text string) {
	kind = strings.TrimSpace(kind)
	key = strings.TrimSpace(key)
	text = strings.TrimSpace(text)
	if kind == "" || text == "" {
		return
	}
	tlMu.Lock()
	defer tlMu.Unlock()
	evs := loadEvents()
	evs = append(evs, Event{TS: time.Now().Unix(), Kind: kind, Key: key, Text: stripTags(text)})
	if len(evs) > 200 {
		evs = evs[len(evs)-200:]
	}
	saveEvents(evs)
}

func ListEvents(limit int) []Event {
	if limit < 1 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}
	tlMu.Lock()
	defer tlMu.Unlock()
	evs := loadEvents()
	if len(evs) > limit {
		evs = evs[len(evs)-limit:]
	}
	out := make([]Event, len(evs))
	for i := range evs {
		out[len(evs)-1-i] = evs[i]
	}
	return out
}

func loadEvents() []Event {
	b, err := os.ReadFile(timelinePath())
	if err != nil {
		return nil
	}
	var evs []Event
	_ = json.Unmarshal(b, &evs)
	return evs
}

func saveEvents(evs []Event) {
	_ = os.MkdirAll(filepath.Dir(timelinePath()), 0o700)
	b, _ := json.MarshalIndent(evs, "", "  ")
	_ = os.WriteFile(timelinePath(), b, 0o600)
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
