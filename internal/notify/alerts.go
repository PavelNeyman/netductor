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
	// Channel / mismatch: shorter re-notify while incident is ongoing (still not spam every tick).
	if strings.HasPrefix(key, "channel:") || strings.HasPrefix(key, "mismatch:") {
		cd = 5 * time.Minute
	}
	if strings.HasPrefix(key, "git:mac-build") {
		cd = 2 * time.Minute
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

// AlertRefresh clears cooldown for key then AlertOnce (ops status text updates).
func AlertRefresh(key, msg string) bool {
	alertMu.Lock()
	delete(lastSent, key)
	delete(clearedAt, key)
	disk := loadSent()
	delete(disk, key)
	saveSent(disk)
	alertMu.Unlock()
	return AlertOnce(key, msg)
}

// ClearAlert marks key healthy. If we previously sent a negative alert for this key,
// enqueue a single ✅ recovery message (ok:<key>) so TG is not only "everything is bad".
func ClearAlert(key string) {
	key = strings.TrimSpace(key)
	if key == "" || strings.HasPrefix(key, "ok:") {
		return
	}
	alertMu.Lock()
	_, wasMem := lastSent[key]
	disk := loadSent()
	_, wasDisk := disk[key]
	was := wasMem || wasDisk
	delete(lastSent, key)
	clearedAt[key] = time.Now()
	delete(disk, key)
	saveSent(disk)
	alertMu.Unlock()

	if !was {
		return
	}
	msg := recoveryMessage(key)
	if msg == "" {
		return
	}
	// Separate key so batch overwrite does not fight the original; no AlertOnce cooldown.
	EnqueueAlert("ok:"+key, msg)
}

func recoveryMessage(key string) string {
	if key == "" || strings.HasPrefix(key, "ok:") {
		return ""
	}
	switch {
	case key == "svcpath:sp":
		return "✅ Service path <b>SP</b> recovered (nd-svc-sp up)"
	case key == "svcpath:ps":
		return "✅ Service path <b>PS</b> recovered (nd-svc-ps up)"
	case key == "sni:down":
		return "✅ VLESS port reachable again"
	case key == "channel:reality-sec":
		return "✅ Reality invalid (from secondary) back to normal"
	case key == "channel:reality-total":
		return "✅ Reality invalid total back to normal"
	case key == "mismatch:core":
		return "✅ Flow mismatch spike cleared"
	case key == "backup:offsite":
		return "✅ Offsite backup path OK again"
	case key == "backup:verify":
		return "✅ Backup verify OK again"
	case key == "firewall:not-ok":
		return "✅ Firewall healthy again"
	case key == "git:mac-build-pending":
		return "✅ Mac build queue empty"
	case strings.HasPrefix(key, "probe:"):
		return "✅ Probe <b>" + escAlert(strings.TrimPrefix(key, "probe:")) + "</b> OK"
	case strings.HasPrefix(key, "svc:"):
		return "✅ Service <b>" + escAlert(strings.TrimPrefix(key, "svc:")) + "</b> active"
	case strings.HasPrefix(key, "secondary:") && strings.HasSuffix(key, ":uplink"):
		id := strings.TrimSuffix(strings.TrimPrefix(key, "secondary:"), ":uplink")
		return "✅ Secondary uplink recovered: <code>" + escAlert(id) + "</code>"
	case strings.HasPrefix(key, "secondary:") && strings.HasSuffix(key, ":sb"):
		id := strings.TrimSuffix(strings.TrimPrefix(key, "secondary:"), ":sb")
		return "✅ Secondary sing-box active: <code>" + escAlert(id) + "</code>"
	case strings.HasPrefix(key, "secondary:"):
		id := strings.TrimPrefix(key, "secondary:")
		return "✅ Secondary online: <code>" + escAlert(id) + "</code>"
	case strings.HasPrefix(key, "channel:"):
		return "✅ Channel recovered: <code>" + escAlert(key) + "</code>"
	case strings.HasPrefix(key, "path:e2e:"):
		return "✅ Path e2e OK: <code>" + escAlert(strings.TrimPrefix(key, "path:e2e:")) + "</code>"
	case strings.HasPrefix(key, "git:mac-build:"):
		return "✅ Mac build job done: <code>" + escAlert(strings.TrimPrefix(key, "git:mac-build:")) + "</code>"
	case strings.HasPrefix(key, "git:build-fail:"):
		return "✅ Project build recovered: <code>" + escAlert(strings.TrimPrefix(key, "git:build-fail:")) + "</code>"
	case strings.HasPrefix(key, "addon-update-fail"):
		return "✅ Addon update path OK: <code>" + escAlert(key) + "</code>"
	case strings.HasSuffix(key, ":log"):
		// log-attach companion keys — no separate recovery spam
		return ""
	default:
		return "✅ Recovered: <code>" + escAlert(key) + "</code>"
	}
}

func escAlert(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// SendTestAlert queues a one-off message and flushes immediately (bypasses AlertOnce cooldown).
func SendTestAlert(msg string) error {
	key := fmt.Sprintf("test:manual:%d", time.Now().UnixNano())
	EnqueueAlert(key, msg)
	return FlushAlerts(true)
}
