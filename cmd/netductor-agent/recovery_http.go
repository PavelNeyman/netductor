package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const recoveryHTML = `<!DOCTYPE html>
<html lang="ru"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>netductor recovery</title>
<style>
body{font-family:system-ui,sans-serif;max-width:28rem;margin:2rem auto;padding:0 1rem}
input,button{width:100%;padding:.6rem;margin:.4rem 0;font-size:1rem;box-sizing:border-box}
button{background:#1a7;color:#fff;border:0;border-radius:6px}
.err{color:#a00}.ok{color:#070}
</style></head><body>
<h1>netductor recovery</h1>
<p>Введите код с primary (CLI/TG). Настройки Wi‑Fi/сети <b>не</b> меняются — только привязка к серверу.</p>
<form method="POST" action="/netductor-recovery">
<label>Primary URL<br/><input name="server" placeholder="https://x.x.x.x:8789" value="{{SERVER}}"/></label>
<label>Recovery / bootstrap code<br/><input name="code" autocomplete="off" required/></label>
<button type="submit">Отправить</button>
</form>
<p class="msg">{{MSG}}</p>
</body></html>`

func startRecoveryHTTP(cfg *config) {
	if os.Getenv("NETDUCTOR_RECOVERY_HTTP") == "0" {
		return
	}
	addr := os.Getenv("NETDUCTOR_RECOVERY_ADDR")
	if addr == "" {
		addr = pickPrivateListenAddr("7879")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/netductor-recovery", func(w http.ResponseWriter, r *http.Request) {
		if !allowRecoveryClient(r) {
			http.Error(w, "forbidden: LAN only", http.StatusForbidden)
			return
		}
		msg := ""
		srv := cfg.Server
		if pin := strings.TrimSpace(os.Getenv("NETDUCTOR_SERVER_PIN")); pin != "" {
			srv = pin
		} else if v := readConfigFlag("SERVER_PIN"); v != "" {
			srv = v
		}
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			code := strings.TrimSpace(r.Form.Get("code"))
			server := strings.TrimSpace(r.Form.Get("server"))
			if pin := strings.TrimSpace(os.Getenv("NETDUCTOR_SERVER_PIN")); pin != "" {
				server = pin
			} else if v := readConfigFlag("SERVER_PIN"); v != "" {
				server = v
			}
			if server == "" {
				server = cfg.Server
			}
			if code == "" || len(code) < 16 {
				msg = `<span class="err">Нужен код (recovery или bootstrap)</span>`
			} else if !validPrimaryURL(server) {
				msg = `<span class="err">Некорректный Primary URL (только http/https)</span>`
			} else {
				server = strings.TrimRight(server, "/")
				if err := writeAgentConfig(server, code, cfg.DeviceID); err != nil {
					msg = `<span class="err">` + htmlEsc(err.Error()) + `</span>`
				} else {
					cfg.Server = server
					cfg.Token = code
					msg = `<span class="ok">Сохранено. Агент сделает enroll → pending на primary. Оператор должен Approve.</span>`
				}
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		body := recoveryHTML
		body = strings.Replace(body, "{{SERVER}}", htmlEsc(srv), 1)
		body = strings.Replace(body, "{{MSG}}", msg, 1)
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/netductor-recovery", http.StatusFound)
	})
	go func() {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			// Never auto-bind 0.0.0.0 unless operator explicitly asked.
			if os.Getenv("NETDUCTOR_RECOVERY_ALLOW_ANY") == "1" && os.Getenv("NETDUCTOR_RECOVERY_ADDR") == "" {
				addr = "127.0.0.1:7879"
				ln, err = net.Listen("tcp", addr)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "recovery http: %v (no private LAN bind; set NETDUCTOR_RECOVERY_ADDR or RECOVERY_ALLOW_ANY=1)\n", err)
				return
			}
		}
		fmt.Fprintf(os.Stderr, "recovery http on %s /netductor-recovery (LAN clients only unless RECOVERY_ALLOW_ANY=1; set RECOVERY_HTTP=0 to disable)\n", addr)
		s := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
		_ = s.Serve(ln)
	}()
}

func pickPrivateListenAddr(port string) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ":" + port
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.To4() == nil || !ip.IsPrivate() {
				continue
			}
			return net.JoinHostPort(ip.String(), port)
		}
	}
	// No private iface: loopback only (not 0.0.0.0).
	return "127.0.0.1:" + port
}

func allowRecoveryClient(r *http.Request) bool {
	if os.Getenv("NETDUCTOR_RECOVERY_ALLOW_ANY") == "1" {
		return true
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func htmlEsc(s string) string {
	r := strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func validPrimaryURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if strings.Contains(u.Host, "@") {
		return false
	}
	return true
}

func writeAgentConfig(server, token, deviceID string) error {
	dir := agentDir()
	_ = os.MkdirAll(dir, 0o700)
	path := dir + "/config"
	cur := map[string]string{}
	if b, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if i := strings.IndexByte(line, '='); i > 0 {
				cur[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
			}
		}
	}
	cur["SERVER"] = server
	cur["TOKEN"] = token
	if deviceID != "" {
		cur["DEVICE_ID"] = deviceID
	}
	cur["CONTROL_ONLY"] = "1"
	if pin := strings.TrimSpace(os.Getenv("NETDUCTOR_SERVER_PIN")); pin != "" {
		cur["SERVER_PIN"] = pin
	}
	var b strings.Builder
	b.WriteString("# written by netductor-recovery\n")
	for _, k := range []string{"SERVER", "TOKEN", "DEVICE_ID", "INTERVAL", "CONTROL_ONLY", "SERVER_PIN", "NVR_DIR", "NVR_MAX_MB"} {
		if v := cur[k]; v != "" {
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(v)
			b.WriteByte('\n')
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}
