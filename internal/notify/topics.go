package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Known topic roles (operator assigns manually after creating topics in Telegram).
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
// Legacy: telegram_alerts_thread_id secret forces one thread for everything.
func ThreadID(topicKey string) int {
	if th := alertsThreadID(); th != "" {
		var n int
		fmt.Sscanf(th, "%d", &n)
		if n > 0 {
			return n
		}
	}
	topicMu.Lock()
	defer topicMu.Unlock()
	return loadTopics().Topics[topicKey]
}

// AssignTopic binds a private-chat topic (thread id from a message) to a role.
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
		return fmt.Errorf("message_thread_id required — send the command from inside a topic")
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

// ClearTopic removes binding for key.
func ClearTopic(key string) error {
	key = strings.ToLower(strings.TrimSpace(key))
	topicMu.Lock()
	defer topicMu.Unlock()
	t := loadTopics()
	delete(t.Topics, key)
	return saveTopics(t)
}

// ListTopics returns role → thread_id (0 if unset).
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
		b.WriteString("<i>Создайте тему в чате с ботом (Threaded Mode в BotFather), откройте её и отправьте:\n<code>/topic alerts</code> (или warnings / service / updates)</i>\n\n")
	} else {
		b.WriteString("📁 <b>Alert topics</b>\n")
		b.WriteString("<i>Create a topic in the bot chat (Threaded Mode in BotFather), open it, send:\n<code>/topic alerts</code> (or warnings / service / updates)</i>\n\n")
	}
	b.WriteString("<table bordered striped compact>\n<tr><th>role</th><th>thread</th></tr>\n")
	keys := append([]string{}, TopicKeys...)
	sort.Strings(keys)
	for _, k := range TopicKeys {
		id := m[k]
		cell := "—"
		if id > 0 {
			cell = fmt.Sprintf("%d", id)
		}
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td><code>%s</code></td></tr>\n", k, cell))
	}
	b.WriteString("</table>\n")
	if th := alertsThreadID(); th != "" {
		if ru {
			b.WriteString("\n⚠️ Задан <code>telegram_alerts_thread_id</code> — все алерты в один тред (перекрывает таблицу).")
		} else {
			b.WriteString("\n⚠️ <code>telegram_alerts_thread_id</code> is set — all alerts use that one thread (overrides table).")
		}
	}
	return b.String()
}

// ResolveThreadIDForMessage picks thread for outbound alert by key.
func ResolveThreadIDForMessage(alertKey string) int {
	return ThreadID(ThreadForAlertKey(alertKey))
}
