package vpn

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

var reMismatchIP = regexp.MustCompile(`from ([0-9.]+):`)

// MismatchStats is flow-mismatch summary from local sing-box journal.
type MismatchStats struct {
	WindowMin int            `json:"window_min"`
	Total     int            `json:"total"`
	ByIP      map[string]int `json:"by_ip"`
	Top       []MismatchIP   `json:"top"`
	Note      string         `json:"note,omitempty"`
}

// MismatchIP one source address.
type MismatchIP struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

// CollectMismatch parses journalctl for sing-box "flow mismatch" lines.
func CollectMismatch(windowMin int) MismatchStats {
	if windowMin <= 0 {
		windowMin = 30
	}
	st := MismatchStats{WindowMin: windowMin, ByIP: map[string]int{}}
	since := fmt.Sprintf("%d min ago", windowMin)
	out, err := exec.Command("journalctl", "-u", "sing-box", "--since", since, "--no-pager", "-o", "cat").CombinedOutput()
	if err != nil {
		out2, err2 := exec.Command("journalctl", "--since", since, "--no-pager", "-o", "cat").CombinedOutput()
		if err2 != nil {
			st.Note = "journal unavailable"
			return st
		}
		out = out2
	}
	nl := string([]byte{10})
	for _, line := range strings.Split(string(out), nl) {
		if !strings.Contains(line, "flow mismatch") {
			continue
		}
		st.Total++
		if strings.Contains(line, "vless-reality") {
			st.Note = "core inbound (vless-reality)"
		} else if strings.Contains(line, "relay-in") {
			st.Note = "secondary inbound"
		}
		m := reMismatchIP.FindStringSubmatch(line)
		if len(m) > 1 {
			st.ByIP[m[1]]++
		}
	}
	type kv struct {
		ip string
		n  int
	}
	var list []kv
	for ip, n := range st.ByIP {
		list = append(list, kv{ip, n})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	for i, x := range list {
		if i >= 8 {
			break
		}
		st.Top = append(st.Top, MismatchIP{IP: x.ip, Count: x.n})
	}
	return st
}

// FormatMismatchText plain summary for CLI/TG.
func FormatMismatchText(st MismatchStats) string {
	nl := string([]byte{10})
	if st.Total == 0 {
		return fmt.Sprintf("flow mismatch (last %dm): 0", st.WindowMin)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("flow mismatch (last %dm): %d", st.WindowMin, st.Total))
	for _, t := range st.Top {
		b.WriteString(fmt.Sprintf("%s  %s × %d", nl, t.IP, t.Count))
	}
	if st.Note != "" {
		b.WriteString(nl + "  (" + st.Note + ")")
	}
	return b.String()
}

// ParseMismatchCount counts mismatch lines (tests).
func ParseMismatchCount(log string) int {
	n := 0
	for _, line := range strings.Split(log, string([]byte{10})) {
		if strings.Contains(line, "flow mismatch") {
			n++
		}
	}
	return n
}
