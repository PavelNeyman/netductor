package channels

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// UserPath is best-effort C11 row (sing-box local stats if enabled).
type UserPath struct {
	User     string  `json:"user,omitempty"`
	UUID     string  `json:"uuid,omitempty"`
	UpBytes  int64   `json:"up_bytes,omitempty"`
	DownBytes int64  `json:"down_bytes,omitempty"`
	Note     string  `json:"note,omitempty"`
}

// UsersReport is C11 aggregate.
type UsersReport struct {
	Available bool       `json:"available"`
	Source    string     `json:"source,omitempty"`
	Users     []UserPath `json:"users,omitempty"`
	Note      string     `json:"note,omitempty"`
	TS        time.Time  `json:"ts"`
}

// CollectUsers tries localhost sing-box clash/stats API (T7).
// Never binds or opens a public stats port — read-only client to 127.0.0.1.
func CollectUsers() UsersReport {
	rep := UsersReport{TS: time.Now().UTC()}
	endpoints := []string{
		"http://127.0.0.1:9090/connections",
		"http://127.0.0.1:9090/traffic",
	}
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_SINGBOX_STATS")); v != "" {
		endpoints = append([]string{strings.TrimRight(v, "/") + "/connections"}, endpoints...)
	}
	client := &http.Client{Timeout: 800 * time.Millisecond}
	for _, url := range endpoints {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if resp.StatusCode != 200 {
			continue
		}
		// Best-effort parse: clash connections JSON has "connections" array with upload/download.
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			// traffic endpoint may be stream — treat as available but empty snapshot
			rep.Available = true
			rep.Source = url
			rep.Note = "stats endpoint reachable; snapshot parse limited"
			return rep
		}
		rep.Available = true
		rep.Source = url
		if conns, ok := raw["connections"].([]any); ok {
			byUser := map[string]*UserPath{}
			for _, c := range conns {
				m, _ := c.(map[string]any)
				if m == nil {
					continue
				}
				meta, _ := m["metadata"].(map[string]any)
				user := ""
				if meta != nil {
					if u, ok := meta["user"].(string); ok {
						user = u
					}
				}
				if user == "" {
					user = "unknown"
				}
				up, _ := asInt64(m["upload"])
				down, _ := asInt64(m["download"])
				if p := byUser[user]; p != nil {
					p.UpBytes += up
					p.DownBytes += down
				} else {
					byUser[user] = &UserPath{User: user, UpBytes: up, DownBytes: down}
				}
			}
			for _, p := range byUser {
				rep.Users = append(rep.Users, *p)
			}
		}
		if len(rep.Users) == 0 {
			rep.Note = "stats API up; no per-user rows in snapshot"
		}
		return rep
	}
	rep.Available = false
	rep.Note = "unsupported: sing-box local stats not enabled (C11 best-effort)"
	return rep
}

func asInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int64:
		return t, true
	case json.Number:
		i, err := t.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}

// FormatUsersText for CLI.
func FormatUsersText(r UsersReport) string {
	if !r.Available {
		return "channel users: " + r.Note + "\n"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("channel users source=%s\n", r.Source))
	if r.Note != "" {
		b.WriteString("  note: " + r.Note + "\n")
	}
	if len(r.Users) == 0 {
		b.WriteString("  (no rows)\n")
		return b.String()
	}
	for _, u := range r.Users {
		b.WriteString(fmt.Sprintf("  %s up=%d down=%d\n", u.User, u.UpBytes, u.DownBytes))
	}
	return b.String()
}
