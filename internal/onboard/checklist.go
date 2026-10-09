package onboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

type Item struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
	Hint  string `json:"hint,omitempty"`
}

type State struct {
	Items     []Item    `json:"items"`
	UpdatedAt time.Time `json:"updated_at"`
}

func defaultItems() []Item {
	return []Item{
		{ID: "user", Title: "Create first VPN user", Hint: "netductor vpn add …"},
		{ID: "sub", Title: "Issue subscription / SR config", Hint: "TG or vpn sr-config"},
		{ID: "egress", Title: "Client egress via tunnel", Hint: "curl ifconfig.me matches node"},
		{ID: "lampac", Title: "Internal service (lampac VIP)", Hint: "http://10.88.0.10:9118"},
		{ID: "secondary", Title: "Secondary online + path e2e", Hint: "netductor channel status"},
		{ID: "policy", Title: "Apply access policy / preset", Hint: "policy presets"},
	}
}

func path() string {
	return filepath.Join(paths.StateDir(), "onboard.json")
}

func Load() State {
	b, err := os.ReadFile(path())
	if err != nil {
		return State{Items: defaultItems()}
	}
	var s State
	if json.Unmarshal(b, &s) != nil || len(s.Items) == 0 {
		return State{Items: defaultItems()}
	}
	return s
}

func Save(s State) error {
	s.UpdatedAt = time.Now().UTC()
	_ = os.MkdirAll(filepath.Dir(path()), 0o700)
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(path(), append(b, '\n'), 0o600)
}

func Mark(id string, done bool) (State, error) {
	s := Load()
	for i := range s.Items {
		if s.Items[i].ID == id {
			s.Items[i].Done = done
		}
	}
	return s, Save(s)
}
