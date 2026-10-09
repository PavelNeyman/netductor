package metrics

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/update"
)

var Services = []string{"sing-box", "blocky", "netductor-api", "netductor-telegram-bot"}

func Dir() string { return paths.MetricsDir() }

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func Collect() map[string]any {
	out := map[string]any{
		"ts":       time.Now().Unix(),
		"hostname": hostname(),
	}
	out["loadavg"] = loadavg()
	out["mem"] = meminfo()
	out["disk"] = diskRoot()
	out["net"] = netCounters()
	out["cpu_pct"] = cpuPct()
	svcs := map[string]string{}
	for _, s := range Services {
		svcs[s] = serviceActive(s)
	}
	out["services"] = svcs
	out["containers"] = dockerPS()
	fw := firewall.AlertIfUnhealthy()
	out["firewall"] = map[string]any{
		"backend": string(fw.Backend), "active": fw.Active, "ok": fw.OK, "role": fw.Role, "warnings": fw.Warnings,
	}
	tags, _ := update.ListLocalTags()
	latest := ""
	if len(tags) > 0 {
		latest = tags[0]
	}
	out["local_release_latest"] = latest
	out["local_release_count"] = len(tags)
	st := update.ReadReleaseBuildStatus()
	out["local_release_build"] = map[string]any{
		"state": st.State, "tag": st.Tag, "detail": st.Detail, "at": st.At,
	}
	if latest != "" {
		if fi, err := os.Stat(update.LocalTagDir(latest)); err == nil {
			out["local_release_mtime"] = fi.ModTime().UTC().Format(time.RFC3339)
			out["local_release_age_sec"] = int(time.Since(fi.ModTime()).Seconds())
		}
	}
	return out
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func loadavg() map[string]float64 {
	m := map[string]float64{}
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return m
	}
	p := strings.Fields(string(b))
	if len(p) >= 3 {
		m["1"], _ = strconv.ParseFloat(p[0], 64)
		m["5"], _ = strconv.ParseFloat(p[1], 64)
		m["15"], _ = strconv.ParseFloat(p[2], 64)
	}
	return m
}

func meminfo() map[string]int64 {
	out := map[string]int64{}
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return out
	}
	defer f.Close()
	kv := map[string]int64{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) >= 2 {
			n, _ := strconv.ParseInt(fs[1], 10, 64)
			kv[strings.TrimSuffix(fs[0], ":")] = n * 1024
		}
	}
	out["total"] = kv["MemTotal"]
	out["available"] = kv["MemAvailable"]
	if out["available"] == 0 {
		out["available"] = kv["MemFree"]
	}
	out["used"] = out["total"] - out["available"]
	return out
}

func diskRoot() map[string]any {
	out := map[string]any{}
	o, err := exec.Command("df", "-B1", "/").Output()
	if err != nil {
		return out
	}
	lines := strings.Split(strings.TrimSpace(string(o)), "\n")
	if len(lines) < 2 {
		return out
	}
	fs := strings.Fields(lines[1])
	if len(fs) >= 5 {
		total, _ := strconv.ParseInt(fs[1], 10, 64)
		used, _ := strconv.ParseInt(fs[2], 10, 64)
		avail, _ := strconv.ParseInt(fs[3], 10, 64)
		out["total"] = total
		out["used"] = used
		out["available"] = avail
		pct := strings.TrimSuffix(fs[4], "%")
		out["use_pct"], _ = strconv.Atoi(pct)
	}
	return out
}

func netCounters() map[string]int64 {
	out := map[string]int64{}
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return out
	}
	var rx, tx int64
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		if strings.Contains(line, "lo:") {
			continue
		}
		p := strings.Fields(strings.Replace(line, ":", " ", 1))
		if len(p) >= 10 {
			r, _ := strconv.ParseInt(p[1], 10, 64)
			t, _ := strconv.ParseInt(p[9], 10, 64)
			rx += r
			tx += t
		}
	}
	out["rx_bytes"] = rx
	out["tx_bytes"] = tx
	return out
}

func cpuPct() float64 {
	// simple 100ms sample
	r1, i1 := readCPU()
	time.Sleep(100 * time.Millisecond)
	r2, i2 := readCPU()
	dr, di := r2-r1, i2-i1
	if dr <= 0 {
		return 0
	}
	return float64(dr-di) / float64(dr) * 100
}

func readCPU() (total, idle int64) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}
	line := strings.SplitN(string(b), "\n", 2)[0]
	fs := strings.Fields(line)
	if len(fs) < 5 || fs[0] != "cpu" {
		return 0, 0
	}
	var sum int64
	for i := 1; i < len(fs); i++ {
		n, _ := strconv.ParseInt(fs[i], 10, 64)
		sum += n
		if i == 4 {
			idle = n
		}
	}
	return sum, idle
}

func serviceActive(name string) string {
	out, err := exec.Command("systemctl", "is-active", name).Output()
	st := strings.TrimSpace(string(out))
	if err != nil && st == "" {
		return "inactive"
	}
	return st
}

func dockerPS() []map[string]string {
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Status}}").Output()
	if err != nil {
		return []map[string]string{}
	}
	var rows []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		p := strings.SplitN(line, "\t", 2)
		if len(p) == 2 {
			rows = append(rows, map[string]string{"name": p[0], "status": p[1]})
		}
	}
	if rows == nil {
		rows = []map[string]string{}
	}
	return rows
}

func History(limit int) []map[string]any {
	if limit <= 0 {
		limit = 180
	}
	path := filepath.Join(Dir(), "history.jsonl")
	b, err := os.ReadFile(path)
	if err != nil {
		return []map[string]any{}
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	out := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func LatestProbes() []any {
	rows := History(1)
	if len(rows) == 0 {
		return []any{}
	}
	if p, ok := rows[len(rows)-1]["probes"].([]any); ok {
		return p
	}
	return []any{}
}

func ProbeUptime(limit int) map[string]any {
	if limit <= 0 {
		limit = 1440
	}
	rows := History(limit)
	stats := map[string][2]int{} // ok, total
	for _, row := range rows {
		probes, _ := row["probes"].([]any)
		for _, raw := range probes {
			p, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			name, _ := p["name"].(string)
			if name == "" {
				name = "?"
			}
			st := stats[name]
			st[1]++
			if okv, _ := p["ok"].(bool); okv {
				st[0]++
			}
			stats[name] = st
		}
	}
	out := map[string]any{}
	for name, st := range stats {
		total := st[1]
		if total == 0 {
			total = 1
		}
		out[name] = map[string]any{
			"ok":         st[0],
			"total":      st[1],
			"uptime_pct": float64(st[0]) / float64(total) * 100,
		}
	}
	return out
}
