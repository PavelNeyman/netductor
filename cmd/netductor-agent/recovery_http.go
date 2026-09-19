package main

import (
	"fmt"
	"net"
	"net/http"
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
<label>Primary URL<br/><input name="server" placeholder="http://x.x.x.x:8787" value="{{SERVER}}"/></label>
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
		addr = ":7879"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/netductor-recovery", func(w http.ResponseWriter, r *http.Request) {
		// best-effort: refuse obvious WAN-only if X-Forwarded (still rely on firewall)
		msg := ""
		srv := cfg.Server
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			code := strings.TrimSpace(r.Form.Get("code"))
			server := strings.TrimSpace(r.Form.Get("server"))
			if server == "" {
				server = cfg.Server
			}
			if code == "" || len(code) < 16 {
				msg = `<span class="err">Нужен код (recovery или bootstrap)</span>`
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
			fmt.Fprintf(os.Stderr, "recovery http: %v\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "recovery http on %s /netductor-recovery (LAN only — firewall WAN)\n", addr)
		s := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
		_ = s.Serve(ln)
	}()
}

func htmlEsc(s string) string {
	r := strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func writeAgentConfig(server, token, deviceID string) error {
	dir := agentDir()
	_ = os.MkdirAll(dir, 0o700)
	path := dir + "/config"
	// preserve other keys
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
	cur["CONTROL_ONLY"] = "1" // recovery must not re-apply site template
	var b strings.Builder
	b.WriteString("# written by netductor-recovery\n")
	for _, k := range []string{"SERVER", "TOKEN", "DEVICE_ID", "INTERVAL", "CONTROL_ONLY", "NVR_DIR", "NVR_MAX_MB"} {
		if v := cur[k]; v != "" {
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(v)
			b.WriteByte('\n')
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}
