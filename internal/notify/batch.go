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

// Alert batching: collect unique keys, flush as one Telegram message,
// then optionally re-pin a compact hub so the interactive menu stays at the bottom.
//
// Guards against storm:
//   - one flush per flushInterval
//   - max maxBatch items per message
//   - hub re-pin at most once per hubRepinMin

const (
	flushInterval = 25 * time.Second
	maxBatch      = 8
	hubRepinMin   = 90 * time.Second
)

var (
	batchMu    sync.Mutex
	pending    = map[string]string{} // key → html body
	flushOnce  sync.Once
	lastFlush  time.Time
	lastHubPin time.Time
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
			t := time.NewTicker(flushInterval)
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
	pending[key] = msg
	n := len(pending)
	batchMu.Unlock()
	startFlusher()
	// flush early if queue is full
	if n >= maxBatch {
		_ = FlushAlerts(false)
	}
}

// FlushAlerts sends pending alerts as one message. force ignores min interval.
func FlushAlerts(force bool) error {
	batchMu.Lock()
	if !force && time.Since(lastFlush) < flushInterval && len(pending) < maxBatch {
		batchMu.Unlock()
		return nil
	}
	if len(pending) == 0 {
		batchMu.Unlock()
		return nil
	}
	// take snapshot
	keys := make([]string, 0, len(pending))
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
	lastFlush = time.Now()
	doHub := time.Since(lastHubPin) >= hubRepinMin
	if doHub {
		lastHubPin = time.Now()
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
		// put back on failure (best-effort, may duplicate later)
		batchMu.Lock()
		for i, k := range keys {
			if i < len(parts) {
				pending[k] = parts[i]
			}
		}
		batchMu.Unlock()
		return err
	}
	if doHub && !AlertsChannelConfigured() {
		// Only re-pin compact hub when alerts share the operator chat (topic mode).
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
	// Channels: sendMessage only. Admin DM: try rich, then sendMessage.
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
	// Compact hub — callbacks match netductor-tg handlers.
	kb := `{"inline_keyboard":[[{"text":"Users","callback_data":"m:users"},{"text":"Fleet","callback_data":"m:cat:nodes"}],[{"text":"Tools","callback_data":"m:tools"},{"text":"Status","callback_data":"m:status"}],[{"text":"📋 Menu","callback_data":"m:menu"}]]}`
	html := hubMenuHTML()
	// sendMessage returns message_id — parse and SaveHubMsg
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
	// Neutral; operator language is not always known from notify package.
	return "📋 <b>Menu</b> / <b>Меню</b>"
}
