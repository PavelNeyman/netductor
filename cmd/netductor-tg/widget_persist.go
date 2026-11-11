package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Persist dual pair + SPA message ids across process restart (AI-RULES lifecycle).

type widgetUIStore struct {
	Dual map[string]dualPair `json:"dual"`
	App  map[string]int      `json:"app"` // chat_id → last SPA/nav message id
}

var (
	widgetStoreOnce sync.Once
	widgetStoreMu   sync.Mutex
)

func widgetStorePath() string {
	return filepath.Join(paths.TelegramDir(), "widget_ui.json")
}

func loadWidgetUIStore() {
	widgetStoreOnce.Do(func() {
		b, err := os.ReadFile(widgetStorePath())
		if err != nil || len(b) == 0 {
			return
		}
		var st widgetUIStore
		if json.Unmarshal(b, &st) != nil {
			return
		}
		lastDualMu.Lock()
		for k, p := range st.Dual {
			id, err := strconv.ParseInt(k, 10, 64)
			if err != nil || (p.InfoID == 0 && p.NavID == 0) {
				continue
			}
			lastDual[id] = p
		}
		lastDualMu.Unlock()
		lastAppMu.Lock()
		for k, mid := range st.App {
			id, err := strconv.ParseInt(k, 10, 64)
			if err != nil || mid <= 0 {
				continue
			}
			lastAppMsgID[id] = mid
		}
		lastAppMu.Unlock()
	})
}

func saveWidgetUIStore() {
	loadWidgetUIStore() // ensure once ran
	widgetStoreMu.Lock()
	defer widgetStoreMu.Unlock()
	st := widgetUIStore{
		Dual: map[string]dualPair{},
		App:  map[string]int{},
	}
	lastDualMu.Lock()
	for id, p := range lastDual {
		if p.InfoID == 0 && p.NavID == 0 {
			continue
		}
		st.Dual[strconv.FormatInt(id, 10)] = p
	}
	lastDualMu.Unlock()
	lastAppMu.Lock()
	for id, mid := range lastAppMsgID {
		if mid > 0 {
			st.App[strconv.FormatInt(id, 10)] = mid
		}
	}
	lastAppMu.Unlock()

	dir := filepath.Dir(widgetStorePath())
	_ = os.MkdirAll(dir, 0o700)
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	tmp := widgetStorePath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "widget store write:", err)
		return
	}
	_ = os.Rename(tmp, widgetStorePath())
}
