package probes

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func ssListening(port int, udp bool) bool {
	flag := "-tlnH"
	if udp {
		flag = "-ulnH"
	}
	out, err := exec.Command("ss", flag, "sport", "=", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func probeTCP(host string, port int, timeout time.Duration) map[string]any {
	t0 := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	ms := float64(time.Since(t0).Milliseconds())
	if err == nil {
		_ = conn.Close()
		return map[string]any{"ok": true, "ms": ms}
	}
	if ssListening(port, false) {
		return map[string]any{"ok": true, "ms": ms, "via": "ss"}
	}
	return map[string]any{"ok": false, "ms": ms, "error": err.Error()}
}

func probeUDP(host string, port int, timeout time.Duration) map[string]any {
	t0 := time.Now()
	if ssListening(port, true) {
		return map[string]any{"ok": true, "ms": float64(time.Since(t0).Milliseconds()), "via": "ss"}
	}
	c, err := net.DialTimeout("udp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return map[string]any{"ok": false, "ms": float64(time.Since(t0).Milliseconds()), "error": err.Error()}
	}
	_ = c.SetDeadline(time.Now().Add(timeout))
	_, _ = c.Write([]byte{0})
	_ = c.Close()
	return map[string]any{"ok": false, "ms": float64(time.Since(t0).Milliseconds()), "error": "no udp listener"}
}

func probeHTTP(url string, timeout time.Duration, insecure bool) map[string]any {
	t0 := time.Now()
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if insecure || strings.HasPrefix(url, "https://127.0.0.1") || strings.HasPrefix(url, "https://localhost") {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // local admin self-signed
	}
	client := &http.Client{Timeout: timeout, Transport: tr}
	resp, err := client.Get(url)
	ms := float64(time.Since(t0).Milliseconds())
	if err != nil {
		return map[string]any{"ok": false, "ms": ms, "error": err.Error()}
	}
	defer resp.Body.Close()
	ok := resp.StatusCode >= 200 && resp.StatusCode < 400
	return map[string]any{"ok": ok, "ms": ms, "code": resp.StatusCode}
}

func Run(cfg map[string]any) []map[string]any {
	raw, _ := cfg["probes"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		p, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := p["name"].(string)
		if name == "" {
			name = "probe"
		}
		timeout := 3.0
		switch v := p["timeout"].(type) {
		case float64:
			timeout = v
		case int:
			timeout = float64(v)
		}
		to := time.Duration(timeout * float64(time.Second))
		typ, _ := p["type"].(string)
		typ = strings.ToLower(typ)
		if typ == "" {
			typ = "tcp"
		}
		var res map[string]any
		switch typ {
		case "http":
			u, _ := p["url"].(string)
			ins, _ := p["insecure"].(bool)
			if !ins && (strings.HasPrefix(u, "https://127.0.0.1") || strings.HasPrefix(u, "https://localhost")) {
				ins = true
			}
			res = probeHTTP(u, to, ins)
		case "udp":
			host, _ := p["host"].(string)
			if host == "" {
				host = "127.0.0.1"
			}
			port := numPort(p["port"])
			res = probeUDP(host, port, to)
		default:
			host, _ := p["host"].(string)
			if host == "" {
				host = "127.0.0.1"
			}
			port := numPort(p["port"])
			res = probeTCP(host, port, to)
		}
		res["name"] = name
		res["type"] = typ
		out = append(out, res)
	}
	return out
}

func numPort(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	default:
		return 0
	}
}
