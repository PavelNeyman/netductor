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

// Fixed bootstrap set — only these topics are auto-created.
var defaultTopics = []struct {
	Key  string
	Name string
}{
	{"alerts", "🚨 Alerts"},
	{"warnings", "⚠️ Warnings"},
	{"service", "🛠 Service"},
	{"updates", "🔄 Updates"},
}

// TopicKeys is the ordered list of roles.
var TopicKeys = []string{"alerts", "warnings", "service", "updates"}

type topicsFile struct {
	ChatID int64          `json:"chat_id"`
	Topics map[string]int `json:"topics"` // key → message_thread_id
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

// ThreadForAlertKey maps alert key → topic role.
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
	topicMu.Lock()
	defer topicMu.Unlock()
	return loadTopics().Topics[topicKey]
}

// EnsureTopics creates the fixed bootstrap topics if missing (Threaded Mode required).
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
			return fmt.Errorf("create topic %s: %w (enable Threaded Mode in BotFather?)", d.Key, err)
		}
		t.Topics[d.Key] = id
		// First real message removes TG "send a message to start this topic" placeholder.
		_ = seedTopicMessage(botToken, chatID, id, d.Key)
	}
	return saveTopics(t)
}

func seedTopicMessage(botToken string, chatID int64, threadID int, key string) error {
	if threadID <= 0 {
		return nil
	}
	text := "📌 " + key + " — netductor"
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	body := fmt.Sprintf(`{"chat_id":%d,"message_thread_id":%d,"text":%q,"disable_notification":true}`, chatID, threadID, text)
	resp, err := http.Post(u, "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wr)
	if !wr.OK {
		return fmt.Errorf("%s", wr.Description)
	}
	return nil
}


// SeedAllTopics posts a quiet marker into each known topic (clears empty-topic UI).
func SeedAllTopics(botToken string, chatID int64) {
	topicMu.Lock()
	t := loadTopics()
	topicMu.Unlock()
	for _, k := range TopicKeys {
		if id := t.Topics[k]; id > 0 {
			_ = seedTopicMessage(botToken, chatID, id, k)
		}
	}
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

// AssignTopic re-binds a role (optional override after bootstrap).
func AssignTopic(chatID int64, key string, threadID int) error {
	key = strings.ToLower(strings.TrimSpace(key))
	ok := false
	for _, k := range TopicKeys {
		if k == key {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("unknown topic key %q (use: %s)", key, strings.Join(TopicKeys, ", "))
	}
	if threadID <= 0 {
		return fmt.Errorf("message_thread_id required")
	}
	topicMu.Lock()
	defer topicMu.Unlock()
	t := loadTopics()
	t.ChatID = chatID
	if t.Topics == nil {
		t.Topics = map[string]int{}
	}
	t.Topics[key] = threadID
	return saveTopics(t)
}

// ListTopics returns role → thread_id.
func ListTopics() map[string]int {
	topicMu.Lock()
	defer topicMu.Unlock()
	t := loadTopics()
	out := map[string]int{}
	for _, k := range TopicKeys {
		out[k] = t.Topics[k]
	}
	return out
}

// TopicsStatusHTML for TG menu.
func TopicsStatusHTML(ru bool) string {
	m := ListTopics()
	var b strings.Builder
	if ru {
		b.WriteString("📁 <b>Топики алертов</b>\n")
		b.WriteString("<i>Bootstrap: Alerts / Warnings / Service / Updates (Threaded Mode в BotFather).\nПереназначить: <code>/topic alerts</code> изнутри темы.</i>\n\n")
	} else {
		b.WriteString("📁 <b>Alert topics</b>\n")
		b.WriteString("<i>Bootstrap: Alerts / Warnings / Service / Updates (Threaded Mode in BotFather).\nRe-bind: <code>/topic alerts</code> from inside a topic.</i>\n\n")
	}
	b.WriteString("<table bordered striped compact>\n<tr><th>role</th><th>thread</th></tr>\n")
	for _, k := range TopicKeys {
		id := m[k]
		cell := "—"
		if id > 0 {
			cell = fmt.Sprintf("%d", id)
		}
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td><code>%s</code></td></tr>\n", k, cell))
	}
	b.WriteString("</table>\n")
	return b.String()
}

// ResolveThreadIDForMessage picks thread for outbound alert by key.
func ResolveThreadIDForMessage(alertKey string) int {
	return ThreadID(ThreadForAlertKey(alertKey))
}
