package notify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Alert batching policy:
//   - same key: overwrite (one message per key)
//   - solitary alert: flush after soloDelay (fast path — no 25s wait)
//   - several different keys in a short window: coalesce up to batchWindow, then one TG message
//   - queue ≥ maxBatch: flush immediately
//   - FlushAlerts(true): always send now (stack apply, tests)

const (
	soloDelay     = 2 * time.Second
	batchWindow   = 8 * time.Second
	maxBatch      = 8
	hubRepinMin   = 90 * time.Second
	tickerPeriod  = 2 * time.Second
)

var (
	batchMu       sync.Mutex
	pending       = map[string]string{} // key → html body
	flushOnce     sync.Once
	lastFlush     time.Time
	lastHubPin    time.Time
	firstPending  time.Time // when current batch started filling
	soloTimer     *time.Timer
)

func hubStatePath() string {
	return filepath.Join(paths.StateDir(), "tg", "hub_msg.json")
}

// HubMsg is the last operator hub (main menu) message we may delete/recreate.
type HubMsg struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int   `json:"message_id"`
	Updated   int64 `json:"updated"`
}

// SaveHubMsg records the latest menu message (called from netductor-tg).
func SaveHubMsg(chatID int64, messageID int) {
	if chatID == 0 || messageID == 0 {
		return
	}
	_ = os.MkdirAll(filepath.Dir(hubStatePath()), 0o700)
	b, _ := json.Marshal(HubMsg{ChatID: chatID, MessageID: messageID, Updated: time.Now().Unix()})
	_ = os.WriteFile(hubStatePath(), append(b, '\n'), 0o600)
}

// LoadHubMsg returns last hub or zero.
func LoadHubMsg() HubMsg {
	b, err := os.ReadFile(hubStatePath())
	if err != nil {
		return HubMsg{}
	}
	var h HubMsg
	_ = json.Unmarshal(b, &h)
	return h
}

// ClearHubMsg drops stored singleton hub id (after user delete / force / topic recreate).
func ClearHubMsg() {
	_ = os.Remove(hubStatePath())
}

func startFlusher() {
	flushOnce.Do(func() {
		go func() {
			t := time.NewTicker(tickerPeriod)
			defer t.Stop()
			for range t.C {
				_ = FlushAlerts(false)
			}
		}()
	})
}

// EnqueueAlert queues a unique alert by key (overwrites same key text).
func EnqueueAlert(key, msg string) {
	key = strings.TrimSpace(key)
	if key == "" || strings.TrimSpace(msg) == "" {
		return
	}
	batchMu.Lock()
	if len(pending) == 0 {
		firstPending = time.Now()
	}
	pending[key] = msg
	n := len(pending)
	batchMu.Unlock()
	startFlusher()

	if n >= maxBatch {
		_ = FlushAlerts(false)
		return
	}
	// Solitary: schedule near-immediate flush (cancelled if more arrive).
	if n == 1 {
		batchMu.Lock()
		if soloTimer != nil {
			soloTimer.Stop()
		}
		soloTimer = time.AfterFunc(soloDelay, func() {
			_ = FlushAlerts(false)
		})
		batchMu.Unlock()
	}
}

// FlushAlerts sends pending alerts as one message. force ignores timing windows.
func FlushAlerts(force bool) error {
	batchMu.Lock()
	n := len(pending)
	if n == 0 {
		batchMu.Unlock()
		return nil
	}
	age := time.Since(firstPending)
	// Adaptive gate:
	//  force → always
	//  n >= maxBatch → always
	//  n == 1 → after soloDelay
	//  n > 1 → after batchWindow (coalesce storm of different keys)
	if !force {
		if n >= maxBatch {
			// ok
		} else if n == 1 && age < soloDelay {
			batchMu.Unlock()
			return nil
		} else if n > 1 && age < batchWindow {
			batchMu.Unlock()
			return nil
		}
	}

	keys := make([]string, 0, n)
	for k := range pending {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > maxBatch {
		keys = keys[:maxBatch]
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, pending[k])
		delete(pending, k)
	}
	if len(pending) == 0 {
		firstPending = time.Time{}
	} else {
		firstPending = time.Now()
	}
	lastFlush = time.Now()
	doHub := time.Since(lastHubPin) >= hubRepinMin
	if doHub {
		lastHubPin = time.Now()
	}
	if soloTimer != nil {
		soloTimer.Stop()
		soloTimer = nil
	}
	batchMu.Unlock()

	var b strings.Builder
	if len(parts) == 1 {
		b.WriteString(parts[0])
	} else {
		b.WriteString(fmt.Sprintf("🔔 <b>%d alerts</b>\n\n", len(parts)))
		for i, p := range parts {
			b.WriteString(fmt.Sprintf("<b>%d.</b> %s\n\n", i+1, p))
		}
	}
	if err := sendTelegramHTML(b.String()); err != nil {
		batchMu.Lock()
		for i, k := range keys {
			if i < len(parts) {
				pending[k] = parts[i]
			}
		}
		if firstPending.IsZero() {
			firstPending = time.Now()
		}
		batchMu.Unlock()
		return err
	}
	if doHub && !AlertsChannelConfigured() {
		_ = repinHub()
	}
	return nil
}

func sendTelegramHTML(msg string) error {
	tok := secret("telegram_bot_token")
	chat := secret("telegram_admin_id")
	channelMode := false
	if ac := AlertsChatID(); ac != "" {
		chat = ac
		channelMode = true
	}
	if tok == "" || chat == "" {
		return fmt.Errorf("telegram secrets not configured")
	}
	if !channelMode {
		u := fmt.Sprintf("https://api.telegram.org/bot%s/sendRichMessage", tok)
		body := fmt.Sprintf(`{"chat_id":%s,"rich_message":{"html":%q}}`, chat, msg)
		resp, err := http.Post(u, "application/json", strings.NewReader(body))
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode < 300 {
				return nil
			}
		}
	}
	vals := url.Values{"chat_id": {chat}, "text": {msg}, "parse_mode": {"HTML"}}
	if err := postTG(tok, "sendMessage", vals); err != nil {
		return fmt.Errorf("sendMessage chat=%s: %w", chat, err)
	}
	return nil
}

func chatIDPositive(chat string) bool {
	var n int64
	_, err := fmt.Sscanf(strings.TrimSpace(chat), "%d", &n)
	return err == nil && n > 0
}

// repinHub deletes previous hub (if any) and sends a compact menu at chat bottom.
func repinHub() error {
	tok := secret("telegram_bot_token")
	chat := secret("telegram_admin_id")
	if tok == "" || chat == "" {
		return fmt.Errorf("telegram secrets not configured")
	}
	h := LoadHubMsg()
	if h.MessageID > 0 {
		_ = postTG(tok, "deleteMessage", url.Values{
			"chat_id":    {fmt.Sprintf("%d", h.ChatID)},
			"message_id": {fmt.Sprintf("%d", h.MessageID)},
		})
	}
	kb := `{"inline_keyboard":[[{"text":"Users","callback_data":"m:users"},{"text":"Fleet","callback_data":"m:cat:nodes"}],[{"text":"Tools","callback_data":"m:tools"},{"text":"Status","callback_data":"m:status"}],[{"text":"📋 Menu","callback_data":"m:menu"}]]}`
	html := hubMenuHTML()
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", tok)
	payload := fmt.Sprintf(`{"chat_id":%s,"text":%q,"parse_mode":"HTML","reply_markup":%s}`, chat, html, kb)
	resp, err := http.Post(u, "application/json", strings.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var wr struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int                `json:"message_id"`
			Chat      struct{ ID int64 } `json:"chat"`
		} `json:"result"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wr)
	if wr.OK && wr.Result.MessageID > 0 {
		cid := wr.Result.Chat.ID
		if cid == 0 {
			fmt.Sscanf(chat, "%d", &cid)
		}
		SaveHubMsg(cid, wr.Result.MessageID)
	}
	return nil
}

func hubMenuHTML() string {
	return "📋 <b>Menu</b> / <b>Меню</b>"
}
