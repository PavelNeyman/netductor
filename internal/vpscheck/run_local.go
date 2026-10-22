package vpscheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Options for RunLocal / RunViaSSH.
type Options struct {
	Profile    Profile
	PrimaryIP  string // optional latency target (secondary → primary)
	SkipInstall bool
	IperfSeconds int // default 5
}

// RunLocal executes checks on the current host (node or operator with tools).
func RunLocal(ctx context.Context, opt Options) (Report, error) {
	if opt.Profile == "" {
		opt.Profile = ProfileGeneric
	}
	if opt.IperfSeconds <= 0 {
		opt.IperfSeconds = 5
	}
	r := Report{At: time.Now().UTC(), Profile: opt.Profile, Host: hostname()}
	if !opt.SkipInstall {
		msg, err := EnsureToolsCtx(ctx)
		st := StatusOK
		detail := msg
		if err != nil {
			st = StatusWarn
			detail = err.Error()
		}
		r.Results = append(r.Results, Result{ID: "tools", TitleEN: "Install tools", TitleRU: "Пакеты", Status: st, Detail: detail})
	}

	// Public IP + region
	ip, region, detail, st := checkIPRegion(ctx)
	r.PublicIP, r.Region = ip, region
	r.Results = append(r.Results, Result{ID: "ip_region", TitleEN: "IP region", TitleRU: "Регион IP", Status: st, Detail: detail})

	// IP quality (DNSBL, proxy flags, egress match)
	r.Results = append(r.Results, checkIPQuality(ctx, ip)...)

	// Bench: CPU / mem / disk
	r.Results = append(r.Results, checkSysbench(ctx))
	r.Results = append(r.Results, checkBenchExtra(ctx)...)
	r.Results = append(r.Results, checkDisk(ctx))

	// Network base
	r.Results = append(r.Results, checkDNS(ctx))
	r.Results = append(r.Results, checkLatency(ctx, "1.1.1.1:443", "cloudflare"))
	r.Results = append(r.Results, checkLatency(ctx, "8.8.8.8:443", "google_dns"))
	r.Results = append(r.Results, checkLatency(ctx, "9.9.9.9:443", "quad9"))
	if opt.PrimaryIP != "" {
		r.Results = append(r.Results, checkLatency(ctx, net.JoinHostPort(opt.PrimaryIP, "443"), "primary_443"))
		r.Results = append(r.Results, checkLatency(ctx, net.JoinHostPort(opt.PrimaryIP, "52222"), "primary_ssh"))
	}

	// Geoblock reachability matrix
	r.Results = append(r.Results, checkGeoblock(ctx, opt.Profile)...)

	// DPI suite (secondary-focused)
	r.Results = append(r.Results, checkDPI(ctx, opt.Profile)...)

	// Profile-specific connectivity + iperf
	switch opt.Profile {
	case ProfileSecondary:
		r.Results = append(r.Results, checkRUHTTP(ctx)...)
		r.Results = append(r.Results, checkIperfRU(ctx, opt.IperfSeconds))
		r.Results = append(r.Results, checkTCPPorts(ctx, []string{"ya.ru:443", "vk.com:443", "api.vk.me:443"}))
	case ProfilePrimary:
		r.Results = append(r.Results, checkAbroadHTTP(ctx)...)
		r.Results = append(r.Results, checkIperfPublic(ctx, opt.IperfSeconds))
		r.Results = append(r.Results, checkTCPPorts(ctx, []string{"1.1.1.1:443", "github.com:443", "cloudflare.com:443"}))
	default:
		r.Results = append(r.Results, checkAbroadHTTP(ctx)...)
		r.Results = append(r.Results, checkRUHTTP(ctx)...)
		r.Results = append(r.Results, checkIperfPublic(ctx, opt.IperfSeconds))
	}

	r.Results = append(r.Results, checkHTTPCode(ctx, "https://api.ipify.org", "ipify"))
	r.Results = append(r.Results, checkHTTPCode(ctx, "https://ifconfig.me/ip", "ifconfig_me"))

	r.tally()
	return r, nil
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func checkIPRegion(ctx context.Context) (ip, region, detail string, st string) {
	client := &http.Client{Timeout: 12 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://ip-api.com/json/?fields=status,message,country,countryCode,regionName,city,query,isp,org,as,mobile,proxy,hosting", nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err.Error(), StatusWarn
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if fmt.Sprint(m["status"]) != "success" {
		return "", "", string(b), StatusWarn
	}
	ip = fmt.Sprint(m["query"])
	region = fmt.Sprintf("%s/%s %s", m["countryCode"], m["regionName"], m["city"])
	detail = fmt.Sprintf("%s isp=%v org=%v as=%v proxy=%v hosting=%v", region, m["isp"], m["org"], m["as"], m["proxy"], m["hosting"])
	st = StatusOK
	if m["proxy"] == true || m["hosting"] == true {
		// hosting=true is normal for VPS
		st = StatusInfo
		if m["proxy"] == true {
			st = StatusWarn
			detail += " (proxy flag)"
		}
	}
	return ip, region, detail, st
}

func checkSysbench(ctx context.Context) Result {
	start := time.Now()
	r := Result{ID: "sysbench_cpu", TitleEN: "CPU sysbench", TitleRU: "CPU sysbench"}
	if _, err := exec.LookPath("sysbench"); err != nil {
		r.Status = StatusSkip
		r.Detail = "sysbench not installed"
		return r
	}
	cmd := exec.CommandContext(ctx, "sysbench", "cpu", "run", "--threads=1")
	out, err := cmd.CombinedOutput()
	r.DurationMs = time.Since(start).Milliseconds()
	s := string(out)
	// parse events per second
	eps := ""
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "events per second") {
			eps = strings.TrimSpace(line)
			break
		}
	}
	if err != nil {
		r.Status = StatusWarn
		r.Detail = truncate(s, 300)
		return r
	}
	r.Status = StatusOK
	if eps != "" {
		r.Detail = eps
	} else {
		r.Detail = truncate(s, 200)
	}
	return r
}

func checkDisk(ctx context.Context) Result {
	start := time.Now()
	r := Result{ID: "disk_write", TitleEN: "Disk write sample", TitleRU: "Запись диска"}
	dir, err := os.MkdirTemp("", "nd-vpscheck-*")
	if err != nil {
		r.Status = StatusWarn
		r.Detail = "tmpdir: " + err.Error()
		return r
	}
	defer os.RemoveAll(dir)
	_ = os.Chmod(dir, 0o700)
	f := filepath.Join(dir, "write.bin")
	// 64MB sequential (smaller; avoids exhausting tmpfs RAM)
	cmd := exec.CommandContext(ctx, "dd", "if=/dev/zero", "of="+f, "bs=1M", "count=64", "conv=fdatasync")
	out, err := cmd.CombinedOutput()
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = truncate(string(out)+" "+err.Error(), 300)
		return r
	}
	r.Status = StatusOK
	r.Detail = truncate(string(out), 200)
	return r
}

func checkDNS(ctx context.Context) Result {
	start := time.Now()
	r := Result{ID: "dns", TitleEN: "DNS resolve", TitleRU: "DNS"}
	resolver := net.DefaultResolver
	_, err := resolver.LookupHost(ctx, "example.com")
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusFail
		r.Detail = err.Error()
		return r
	}
	r.Status = StatusOK
	r.Detail = fmt.Sprintf("example.com ok %dms", r.DurationMs)
	return r
}

func checkLatency(ctx context.Context, addr, id string) Result {
	start := time.Now()
	r := Result{ID: "latency_" + id, TitleEN: "TCP " + addr, TitleRU: "TCP " + addr}
	d := net.Dialer{Timeout: 5 * time.Second}
	c, err := d.DialContext(ctx, "tcp", addr)
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = err.Error()
		return r
	}
	_ = c.Close()
	r.Status = StatusOK
	r.Detail = fmt.Sprintf("%dms", r.DurationMs)
	return r
}

func checkTCPPorts(ctx context.Context, addrs []string) Result {
	start := time.Now()
	r := Result{ID: "tcp_bulk", TitleEN: "TCP bulk", TitleRU: "TCP пачка"}
	var ok, fail []string
	for _, a := range addrs {
		d := net.Dialer{Timeout: 4 * time.Second}
		c, err := d.DialContext(ctx, "tcp", a)
		if err != nil {
			fail = append(fail, a)
			continue
		}
		_ = c.Close()
		ok = append(ok, a)
	}
	r.DurationMs = time.Since(start).Milliseconds()
	if len(fail) == 0 {
		r.Status = StatusOK
		r.Detail = "ok " + strings.Join(ok, ",")
	} else if len(ok) == 0 {
		r.Status = StatusFail
		r.Detail = "all failed: " + strings.Join(fail, ",")
	} else {
		r.Status = StatusWarn
		r.Detail = "ok=" + strings.Join(ok, ",") + " fail=" + strings.Join(fail, ",")
	}
	return r
}

func checkHTTPCode(ctx context.Context, url, id string) Result {
	start := time.Now()
	r := Result{ID: "http_" + id, TitleEN: "HTTP " + id, TitleRU: "HTTP " + id}
	client := &http.Client{Timeout: 12 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = err.Error()
		return r
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		r.Status = StatusOK
	} else {
		r.Status = StatusWarn
	}
	r.Detail = fmt.Sprintf("%d %dms", resp.StatusCode, r.DurationMs)
	return r
}

func checkRUHTTP(ctx context.Context) []Result {
	return []Result{
		checkHTTPCode(ctx, "https://ya.ru/", "ya_ru"),
		checkHTTPCode(ctx, "https://vk.com/", "vk"),
	}
}

func checkAbroadHTTP(ctx context.Context) []Result {
	return []Result{
		checkHTTPCode(ctx, "https://www.cloudflare.com/cdn-cgi/trace", "cf_trace"),
		checkHTTPCode(ctx, "https://github.com/", "github"),
	}
}

// Known public iperf3 servers (may change; failures → warn not fail).
var iperfRU = []string{
	"iperf.msk1.wnet.su:5201",
	"iperf.spb1.wnet.su:5201",
	"speedtest.selectel.ru:5201",
}

var iperfPublic = []string{
	"iperf3.moji.online:5201",
	"bouygues.iperf.fr:5201",
	"iperf.he.net:5201",
}

func checkIperfRU(ctx context.Context, sec int) Result {
	return runIperf(ctx, "iperf_ru", "iperf3 RU", "iperf3 РФ", iperfRU, sec)
}

func checkIperfPublic(ctx context.Context, sec int) Result {
	return runIperf(ctx, "iperf_pub", "iperf3 public", "iperf3 public", iperfPublic, sec)
}

func runIperf(ctx context.Context, id, en, ru string, servers []string, sec int) Result {
	start := time.Now()
	r := Result{ID: id, TitleEN: en, TitleRU: ru}
	if _, err := exec.LookPath("iperf3"); err != nil {
		r.Status = StatusSkip
		r.Detail = "iperf3 not installed"
		return r
	}
	var last string
	for _, srv := range servers {
		host, port, _ := strings.Cut(srv, ":")
		if port == "" {
			port = "5201"
		}
		args := []string{"-c", host, "-p", port, "-t", strconv.Itoa(sec), "-J"}
		cmd := exec.CommandContext(ctx, "iperf3", args...)
		out, err := cmd.CombinedOutput()
		last = truncate(string(out), 400)
		if err != nil {
			continue
		}
		// parse bits_per_second from JSON if possible
		var j map[string]any
		if json.Unmarshal(out, &j) == nil {
			if end, ok := j["end"].(map[string]any); ok {
				if sum, ok := end["sum_received"].(map[string]any); ok {
					if bps, ok := sum["bits_per_second"].(float64); ok {
						r.Status = StatusOK
						r.Detail = fmt.Sprintf("%s → %.1f Mbit/s", srv, bps/1e6)
						r.DurationMs = time.Since(start).Milliseconds()
						return r
					}
				}
			}
		}
		r.Status = StatusOK
		r.Detail = srv + " ok"
		r.DurationMs = time.Since(start).Milliseconds()
		return r
	}
	r.Status = StatusWarn
	r.Detail = "no iperf server responded: " + last
	r.DurationMs = time.Since(start).Milliseconds()
	return r
}
