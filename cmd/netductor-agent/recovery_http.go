package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/format"
)


// recoveryUnlockPath is a local secret (root-only file) required to rebind an already-configured agent.
func recoveryUnlockPath() string {
	return agentDir() + "/recovery_unlock"
}

// ensureRecoveryUnlock creates unlock secret once (for operator to read via SSH).
func ensureRecoveryUnlock() string {
	path := recoveryUnlockPath()
	if b, err := os.ReadFile(path); err == nil {
		s := strings.TrimSpace(string(b))
		if len(s) >= 16 {
			return s
		}
	}
	_ = os.MkdirAll(agentDir(), 0o700)
	var buf [24]byte
	_, _ = rand.Read(buf[:])
	s := fmt.Sprintf("%x", buf[:])
	_ = os.WriteFile(path, []byte(s+"\n"), 0o600)
	return s
}

func recoveryUnlockOK(submitted string) bool {
	submitted = strings.TrimSpace(submitted)
	if submitted == "" {
		return false
	}
	b, err := os.ReadFile(recoveryUnlockPath())
	if err != nil {
		return false
	}
	want := strings.TrimSpace(string(b))
	if len(submitted) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(submitted), []byte(want)) == 1
}


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
<p>Введите код с primary (CLI/TG). Страница по <b>HTTPS</b> (self-signed). Wi‑Fi/сеть <b>не</b> меняются — только привязка к серверу.</p>
<form method="POST" action="/netductor-recovery">
<label>Primary URL<br/><input name="server" placeholder="https://x.x.x.x:8789" value="{{SERVER}}"/></label>
<label>Recovery / bootstrap code<br/><input name="code" autocomplete="off" required/></label>
<label>Local unlock (если агент уже привязан; файл recovery_unlock на роутере)<br/><input name="unlock" autocomplete="off"/></label>
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
			} else if strings.TrimSpace(cfg.Token) != "" || strings.TrimSpace(cfg.Server) != "" {
				// Already bound: require local unlock (LAN-only is not authentication).
				unlock := strings.TrimSpace(r.Form.Get("unlock"))
				_ = ensureRecoveryUnlock()
				if !recoveryUnlockOK(unlock) {
					msg = `<span class="err">Нужен local unlock (cat …/recovery_unlock на роутере). Без него перепривязка запрещена.</span>`
				} else if !validPrimaryURL(server) {
					msg = `<span class="err">Некорректный Primary URL (нужен https://…:8789; lab: NETDUCTOR_RECOVERY_ALLOW_HTTP=1)</span>`
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
			} else if !validPrimaryURL(server) {
				msg = `<span class="err">Некорректный Primary URL (нужен https://…:8789; lab: NETDUCTOR_RECOVERY_ALLOW_HTTP=1)</span>`
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
		fmt.Fprintf(os.Stderr, "recovery https on %s /netductor-recovery (self-signed; LAN only unless RECOVERY_ALLOW_ANY=1)\n", addr)
		s := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
		cert, err := recoveryTLSCertificate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "recovery tls cert: %v — falling back to HTTP (set NETDUCTOR_RECOVERY_HTTP=1 to force)\n", err)
			if os.Getenv("NETDUCTOR_RECOVERY_HTTP") == "1" {
				_ = s.Serve(ln)
			}
			return
		}
		_ = s.Serve(tls.NewListener(ln, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}))
	}()
}

// recoveryTLSCertificate loads PEM from env paths or generates ephemeral self-signed for LAN recovery UI.
func recoveryTLSCertificate() (tls.Certificate, error) {
	certFile := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_TLS_CERT"))
	keyFile := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_TLS_KEY"))
	if certFile != "" && keyFile != "" {
		return tls.LoadX509KeyPair(certFile, keyFile)
	}
	// Ephemeral self-signed (valid 7d) — browser warning expected; protects recovery code on LAN.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "netductor-recovery"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(7 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	// Attach private interface IPs as SANs.
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil {
				tmpl.IPAddresses = append(tmpl.IPAddresses, ipn.IP)
			}
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return tls.X509KeyPair(certPEM, keyPEM)
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

func htmlEsc(s string) string { return format.HTMLEsc(s) }

func validPrimaryURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	// HTTPS required by default — recovery code must not travel as plaintext (review P1).
	// Opt-in: NETDUCTOR_RECOVERY_ALLOW_HTTP=1 for lab-only.
	allowHTTP := os.Getenv("NETDUCTOR_RECOVERY_ALLOW_HTTP") == "1"
	if u.Scheme == "https" {
		// ok
	} else if u.Scheme == "http" && allowHTTP {
		// lab only
	} else {
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
