package notify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Fixed bootstrap set — only these roles are managed as core.
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
	ChatID    int64          `json:"chat_id"`
	Topics    map[string]int `json:"topics"` // key → message_thread_id
	Names     map[string]string `json:"names,omitempty"` // key → display name
	UpdatedAt string         `json:"updated_at,omitempty"`
}

var topicMu sync.Mutex

func topicsPath() string {
	return filepath.Join(paths.StateDir(), "tg", "topics.json")
}

func topicsBackupPath() string {
	return filepath.Join(paths.StateDir(), "tg", "topics-backup.json")
}

func loadTopics() topicsFile {
	var t topicsFile
	t.Topics = map[string]int{}
	t.Names = map[string]string{}
	b, err := os.ReadFile(topicsPath())
	if err != nil {
		return t
	}
	_ = json.Unmarshal(b, &t)
	if t.Topics == nil {
		t.Topics = map[string]int{}
	}
	if t.Names == nil {
		t.Names = map[string]string{}
	}
	return t
}

func saveTopics(t topicsFile) error {
	_ = os.MkdirAll(filepath.Dir(topicsPath()), 0o700)
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	raw, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(topicsPath(), append(raw, '\n'), 0o600); err != nil {
		return err
	}
	// Snapshot backup whenever we have at least one valid bootstrap id
	alive := 0
	for _, k := range TopicKeys {
		if t.Topics[k] > 0 {
			alive++
		}
	}
	if alive > 0 {
		_ = os.WriteFile(topicsBackupPath(), append(raw, '\n'), 0o600)
	}
	return nil
}

func loadTopicsBackup() topicsFile {
	var t topicsFile
	t.Topics = map[string]int{}
	t.Names = map[string]string{}
	b, err := os.ReadFile(topicsBackupPath())
	if err != nil {
		return t
	}
	_ = json.Unmarshal(b, &t)
	if t.Topics == nil {
		t.Topics = map[string]int{}
	}
	if t.Names == nil {
		t.Names = map[string]string{}
	}
	return t
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

// topicAlive checks whether thread still accepts messages (cleared chat → dead ids).
func topicAlive(botToken string, chatID int64, threadID int) bool {
	if threadID <= 0 || botToken == "" || chatID == 0 {
		return false
	}
	// sendChatAction is lightweight; fails with TOPIC_ID_INVALID when topic is gone
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendChatAction", botToken)
	body := fmt.Sprintf(`{"chat_id":%d,"message_thread_id":%d,"action":"typing"}`, chatID, threadID)
	resp, err := http.Post(u, "application/json", strings.NewReader(body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wr)
	return wr.OK
}

func nameForKey(t topicsFile, key string) string {
	if t.Names != nil {
		if n := t.Names[key]; n != "" {
			return n
		}
	}
	for _, d := range defaultTopics {
		if d.Key == key {
			return d.Name
		}
	}
	return key
}

// EnsureTopics creates/reconciles bootstrap topics.
// - Verifies stored thread ids still work
// - If some missing but at least one bootstrap alive → recreate only dead ones
// - If all bootstrap dead → restore from topics-backup.json names, then create
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
	if t.Names == nil {
		t.Names = map[string]string{}
	}
	for _, d := range defaultTopics {
		if t.Names[d.Key] == "" {
			t.Names[d.Key] = d.Name
		}
	}

	// Verify existing ids
	aliveN := 0
	for _, k := range TopicKeys {
		id := t.Topics[k]
		if id > 0 && topicAlive(botToken, chatID, id) {
			aliveN++
			continue
		}
		if id > 0 {
			// dead id (e.g. user cleared chat history)
			t.Topics[k] = 0
		}
	}

	// Full wipe: restore names from backup if we have one
	if aliveN == 0 {
		bak := loadTopicsBackup()
		if bak.Names != nil {
			for _, k := range TopicKeys {
				if bak.Names[k] != "" {
					t.Names[k] = bak.Names[k]
				}
			}
		}
	}

	// Recreate missing bootstrap topics
	for _, d := range defaultTopics {
		if t.Topics[d.Key] > 0 {
			continue
		}
		name := nameForKey(t, d.Key)
		id, err := createForumTopic(botToken, chatID, name)
		if err != nil {
			return fmt.Errorf("create topic %s: %w (need a *group/supergroup* with Topics on — not a private chat; BotFather → Group Privacy / Topics)", d.Key, err)
		}
		t.Topics[d.Key] = id
		t.Names[d.Key] = name
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

// SeedAllTopics posts a quiet marker into each known topic.
func SeedAllTopics(botToken string, chatID int64) {
	if botToken == "" || chatID == 0 {
		return
	}
	topicMu.Lock()
	t := loadTopics()
	topicMu.Unlock()
	if t.Topics == nil {
		return
	}
	for _, k := range TopicKeys {
		if id := t.Topics[k]; id > 0 {
			_ = seedTopicMessage(botToken, chatID, id, k)
		}
	}
}

// ReconcileTopics is EnsureTopics + optional discovery of extra topics via /topic.
// Safe for periodic call (rate-limited by caller).
func ReconcileTopics(botToken string, chatID int64) error {
	return EnsureTopics(botToken, chatID)
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
			MessageThreadID int    `json:"message_thread_id"`
			Name            string `json:"name"`
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
	if t.Names == nil {
		t.Names = map[string]string{}
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
		b.WriteString("<i>Bootstrap: Alerts / Warnings / Service / Updates.\nПосле очистки чата id сбрасываются и топики создаются заново из бэкапа имён.</i>\n\n")
	} else {
		b.WriteString("📁 <b>Alert topics</b>\n")
		b.WriteString("<i>Bootstrap: Alerts / Warnings / Service / Updates.\nAfter clearing chat, dead ids are detected and topics recreated from name backup.</i>\n\n")
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
