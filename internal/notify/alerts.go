package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var (
	alertMu    sync.Mutex
	lastSent   = map[string]time.Time{}
	clearedAt  = map[string]time.Time{}
	cooldown   = 15 * time.Minute
	rearmAfter = 5 * time.Minute // after ClearAlert, do not re-alert sooner
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

// SuppressUntil writes a marker so alerts are ignored until t (e.g. during sing-box restart).
func SuppressUntil(d time.Duration) {
	_ = os.MkdirAll("/run/netductor", 0o755)
	until := time.Now().Add(d).Unix()
	_ = os.WriteFile("/run/netductor/suppress_alerts_until", []byte(fmt.Sprintf("%d\n", until)), 0o644)
}

func alertsSuppressed() bool {
	b, err := os.ReadFile("/run/netductor/suppress_alerts_until")
	if err != nil {
		return false
	}
	var until int64
	_, _ = fmt.Sscanf(strings.TrimSpace(string(b)), "%d", &until)
	return until > time.Now().Unix()
}

// AlertOnce sends Telegram at most once per key within cooldown.
func AlertOnce(key, msg string) bool {
	alertMu.Lock()
	defer alertMu.Unlock()
	if alertsSuppressed() {
		return false
	}
	now := time.Now()
	cd := cooldown
	if strings.HasPrefix(key, "update:available:") {
		cd = 24 * time.Hour // one reminder per day per release tag
	}
	if t, ok := lastSent[key]; ok && now.Sub(t) < cd {
		return false
	}
	if t, ok := clearedAt[key]; ok && now.Sub(t) < rearmAfter {
		return false
	}
	disk := loadSent()
	if ts, ok := disk[key]; ok && now.Unix()-ts < int64(cd.Seconds()) {
		return false
	}
	EnqueueAlert(key, msg)
	if smtpConfigured() {
		_ = Email("netductor: "+key, stripTags(msg))
	}
	lastSent[key] = now
	disk[key] = now.Unix()
	saveSent(disk)
	return true
}

func ClearAlert(key string) {
	alertMu.Lock()
	defer alertMu.Unlock()
	delete(lastSent, key)
	clearedAt[key] = time.Now()
	disk := loadSent()
	delete(disk, key)
	saveSent(disk)
}

// SendTestAlert queues a one-off message and flushes immediately (bypasses AlertOnce cooldown).
func SendTestAlert(msg string) error {
	key := fmt.Sprintf("test:manual:%d", time.Now().UnixNano())
	EnqueueAlert(key, msg)
	return FlushAlerts(true)
}
