package channels

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

// UserPath is best-effort C11 row (sing-box local stats if enabled).
type UserPath struct {
	User      string `json:"user,omitempty"`
	UUID      string `json:"uuid,omitempty"`
	UpBytes   int64  `json:"up_bytes,omitempty"`
	DownBytes int64  `json:"down_bytes,omitempty"`
	Source    string `json:"source,omitempty"` // primary | secondary:<id>
	Note      string `json:"note,omitempty"`
}

// UsersReport is C11 aggregate.
type UsersReport struct {
	Available bool       `json:"available"`
	Source    string     `json:"source,omitempty"`
	Users     []UserPath `json:"users,omitempty"`
	Note      string     `json:"note,omitempty"`
	TS        time.Time  `json:"ts"`
}

// CollectUsers tries localhost sing-box clash/stats API and secondary heartbeat snapshots (T7).
func CollectUsers() UsersReport {
	rep := UsersReport{TS: time.Now().UTC()}
	var users []UserPath
	sources := []string{}

	if local, src, ok := scrapeLocalUsers(); ok {
		users = append(users, local...)
		sources = append(sources, src)
		rep.Available = true
	}

	// Secondary agents may report UserTraffic from their local clash_api (client face).
	for _, d := range secondary.List() {
		if len(d.UserTraffic) == 0 {
			continue
		}
		if !secondary.Online(d, 3*time.Minute) {
			continue
		}
		src := "secondary:" + d.ID
		if d.Name != "" {
			src = "secondary:" + d.Name
		}
		sources = append(sources, src)
		rep.Available = true
		for _, ut := range d.UserTraffic {
			users = append(users, UserPath{
				User: ut.User, UUID: ut.UUID,
				UpBytes: ut.Up, DownBytes: ut.Down,
				Source: src,
			})
		}
	}

	rep.Users = users
	if len(sources) > 0 {
		rep.Source = strings.Join(sources, ",")
	}
	if !rep.Available {
		rep.Note = "unsupported: sing-box local stats not enabled (C11 best-effort); enable via Apply clash_api then vpn apply / secondary config pull"
		return rep
	}
	if len(rep.Users) == 0 {
		rep.Note = "stats available; no active per-user rows in snapshot"
	}
	return rep
}

func scrapeLocalUsers() (users []UserPath, source string, ok bool) {
	endpoints := []string{
		"http://127.0.0.1:9090/connections",
	}
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_SINGBOX_STATS")); v != "" {
		endpoints = append([]string{strings.TrimRight(v, "/") + "/connections"}, endpoints...)
	}
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_CLASH_API")); v != "" && v != "off" && v != "0" {
		addr := v
		if !strings.Contains(addr, "://") {
			if !strings.Contains(addr, ":") {
				addr = "127.0.0.1:" + addr
			}
			addr = "http://" + addr
		}
		endpoints = append([]string{strings.TrimRight(addr, "/") + "/connections"}, endpoints...)
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
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, url, true
		}
		source = url
		ok = true
		if conns, okc := raw["connections"].([]any); okc {
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
					byUser[user] = &UserPath{User: user, UpBytes: up, DownBytes: down, Source: "primary"}
				}
			}
			for _, p := range byUser {
				users = append(users, *p)
			}
		}
		return users, source, true
	}
	return nil, "", false
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
		src := u.Source
		if src != "" {
			src = " [" + src + "]"
		}
		b.WriteString(fmt.Sprintf("  %s up=%d down=%d%s\n", u.User, u.UpBytes, u.DownBytes, src))
	}
	return b.String()
}
