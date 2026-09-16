package notify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var (
	alertMu   sync.Mutex
	lastSent  = map[string]time.Time{}
	cooldown  = 15 * time.Minute
)

func alertStatePath() string {
	return filepath.Join(paths.StateDir(), "alerts_sent.json")
}

func loadSent() map[string]int64 {
	b, err := os.ReadFile(alertStatePath())
	if err != nil {
		return map[string]int64{}
	}
	var m map[string]int64
	_ = json.Unmarshal(b, &m)
	if m == nil {
		return map[string]int64{}
	}
	return m
}

func saveSent(m map[string]int64) {
	_ = os.MkdirAll(filepath.Dir(alertStatePath()), 0o700)
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(alertStatePath(), b, 0o600)
}

// AlertOnce sends Telegram at most once per key within cooldown.
func AlertOnce(key, msg string) {
	alertMu.Lock()
	defer alertMu.Unlock()
	now := time.Now()
	if t, ok := lastSent[key]; ok && now.Sub(t) < cooldown {
		return
	}
	disk := loadSent()
	if ts, ok := disk[key]; ok && now.Unix()-ts < int64(cooldown.Seconds()) {
		return
	}
	tgErr := Telegram(msg)
	if smtpConfigured() {
		_ = Email("netductor: "+key, stripTags(msg))
	}
	if tgErr != nil {
		return
	}
	lastSent[key] = now
	disk[key] = now.Unix()
	saveSent(disk)
}

func ClearAlert(key string) {
	alertMu.Lock()
	defer alertMu.Unlock()
	delete(lastSent, key)
	disk := loadSent()
	delete(disk, key)
	saveSent(disk)
}
