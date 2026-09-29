package notify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Default private-chat topics the bot ensures when Threaded Mode is on.
var defaultTopics = []struct {
	Key  string
	Name string
}{
	{"alerts", "🚨 Alerts"},
	{"warnings", "⚠️ Warnings"},
	{"service", "🛠 Service"},
	{"updates", "🔄 Updates"},
}

type topicsFile struct {
	ChatID int64             `json:"chat_id"`
	Topics map[string]int    `json:"topics"` // key → message_thread_id
}

var topicMu sync.Mutex

func topicsPath() string {
	return filepath.Join(paths.StateDir(), "tg", "topics.json")
}

func loadTopics() topicsFile {
	var t topicsFile
	t.Topics = map[string]int{}
	b, err := os.ReadFile(topicsPath())
	if err != nil {
		return t
	}
	_ = json.Unmarshal(b, &t)
	if t.Topics == nil {
		t.Topics = map[string]int{}
	}
	return t
}

func saveTopics(t topicsFile) error {
	_ = os.MkdirAll(filepath.Dir(topicsPath()), 0o700)
	raw, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(topicsPath(), append(raw, '\n'), 0o600)
}

// ThreadForAlertKey maps alert key prefix → topic key.
func ThreadForAlertKey(key string) string {
	k := strings.ToLower(key)
	switch {
	case strings.Contains(k, "update") || strings.Contains(k, "release"):
		return "updates"
	case strings.Contains(k, "warn") || strings.Contains(k, "mismatch") || strings.Contains(k, "quota"):
		return "warnings"
	case strings.Contains(k, "backup") || strings.Contains(k, "mtls") || strings.Contains(k, "svc") || strings.Contains(k, "path"):
		return "service"
	default:
		return "alerts"
	}
}

// ThreadID returns message_thread_id for topic key, or 0.
func ThreadID(topicKey string) int {
	// explicit secret overrides all (legacy single-thread)
	if th := alertsThreadID(); th != "" {
		var n int
		fmt.Sscanf(th, "%d", &n)
		if n > 0 {
			return n
		}
	}
	topicMu.Lock()
	defer topicMu.Unlock()
	t := loadTopics()
	return t.Topics[topicKey]
}

// EnsureTopics creates missing private-chat topics via Bot API and stores ids.
// Safe to call repeatedly. Requires BotFather Threaded Mode.
func EnsureTopics(botToken string, chatID int64) error {
	if botToken == "" || chatID == 0 {
		return fmt.Errorf("token/chat required")
	}
	topicMu.Lock()
	defer topicMu.Unlock()
	t := loadTopics()
	t.ChatID = chatID
	if t.Topics == nil {
		t.Topics = map[string]int{}
	}
	for _, d := range defaultTopics {
		if t.Topics[d.Key] > 0 {
			continue
		}
		id, err := createForumTopic(botToken, chatID, d.Name)
		if err != nil {
			// Threaded Mode off or API error — skip remaining
			return fmt.Errorf("create topic %s: %w (enable Threaded Mode in BotFather?)", d.Key, err)
		}
		t.Topics[d.Key] = id
	}
	return saveTopics(t)
}

func createForumTopic(botToken string, chatID int64, name string) (int, error) {
	u := fmt.Sprintf("https://api.telegram.org/bot%s/createForumTopic", botToken)
	body := fmt.Sprintf(`{"chat_id":%d,"name":%q}`, chatID, name)
	resp, err := http.Post(u, "application/json", strings.NewReader(body))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Result      struct {
			MessageThreadID int `json:"message_thread_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wr); err != nil {
		return 0, err
	}
	if !wr.OK || wr.Result.MessageThreadID == 0 {
		return 0, fmt.Errorf("%s", wr.Description)
	}
	return wr.Result.MessageThreadID, nil
}

// ResolveThreadIDForMessage picks thread for outbound alert by key.
func ResolveThreadIDForMessage(alertKey string) int {
	return ThreadID(ThreadForAlertKey(alertKey))
}
