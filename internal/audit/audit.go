package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var mu sync.Mutex

type Event struct {
	TS     int64  `json:"ts"`
	Actor  string `json:"actor,omitempty"`
	Action string `json:"action"`
	Target string `json:"target,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func path() string {
	return filepath.Join(paths.StateDir(), "audit.jsonl")
}

func Log(actor, action, target, detail string) {
	mu.Lock()
	defer mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(path()), 0o700)
	ev := Event{TS: time.Now().Unix(), Actor: actor, Action: action, Target: target, Detail: detail}
	b, _ := json.Marshal(ev)
	f, err := os.OpenFile(path(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, 10))
}

// Tail returns the last n events (oldest first among the tail).
func Tail(n int) []Event {
	if n <= 0 {
		n = 50
	}
	mu.Lock()
	defer mu.Unlock()
	f, err := os.Open(path())
	if err != nil {
		return nil
	}
	defer f.Close()
	var all []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev Event
		if json.Unmarshal(line, &ev) != nil {
			continue
		}
		all = append(all, ev)
	}
	if len(all) > n {
		all = all[len(all)-n:]
	}
	return all
}
