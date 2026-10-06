package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// KnownChat is a channel/supergroup the bot has seen (my_chat_member / channel_post / forward).
type KnownChat struct {
	ID        int64  `json:"id"`
	Title     string `json:"title,omitempty"`
	Type      string `json:"type,omitempty"` // channel, supergroup, …
	UpdatedAt string `json:"updated_at,omitempty"`
}

var knownMu sync.Mutex

func knownChatsPath() string {
	return filepath.Join(paths.StateDir(), "tg", "known_chats.json")
}

func loadKnownChats() map[string]KnownChat {
	m := map[string]KnownChat{}
	b, err := os.ReadFile(knownChatsPath())
	if err != nil {
		return m
	}
	_ = json.Unmarshal(b, &m)
	if m == nil {
		m = map[string]KnownChat{}
	}
	return m
}

func saveKnownChats(m map[string]KnownChat) {
	_ = os.MkdirAll(filepath.Dir(knownChatsPath()), 0o700)
	raw, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(knownChatsPath(), append(raw, '\n'), 0o600)
}

// RememberChat records a chat id the bot observed.
func RememberChat(id int64, title, typ string) {
	if id == 0 {
		return
	}
	knownMu.Lock()
	defer knownMu.Unlock()
	m := loadKnownChats()
	key := strconv.FormatInt(id, 10)
	prev := m[key]
	if title == "" {
		title = prev.Title
	}
	if typ == "" {
		typ = prev.Type
	}
	m[key] = KnownChat{
		ID: id, Title: title, Type: typ,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	saveKnownChats(m)
}

// ListKnownChats newest-ish by UpdatedAt.
func ListKnownChats() []KnownChat {
	knownMu.Lock()
	defer knownMu.Unlock()
	m := loadKnownChats()
	out := make([]KnownChat, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

// ParseChatID accepts "-100…" or plain digits.
func ParseChatID(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty id")
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("chat id must be numeric")
	}
	return n, nil
}
