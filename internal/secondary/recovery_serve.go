package secondary

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const recoveryPort = "8790"

// EnsureRecoveryToken creates durable recovery token for DR pull from secondary.
func EnsureRecoveryToken() (string, error) {
	path := "/etc/netductor/secrets/recovery_token"
	if b, err := os.ReadFile(path); err == nil {
		t := strings.TrimSpace(string(b))
		if len(t) >= 16 {
			return t, nil
		}
	}
	buf := make([]byte, 24)
	_, _ = rand.Read(buf)
	t := hex.EncodeToString(buf)
	_ = os.MkdirAll("/etc/netductor/secrets", 0o700)
	if err := os.WriteFile(path, []byte(t+"\n"), 0o600); err != nil {
		return "", err
	}
	return t, nil
}

// StartRecoveryServer serves encrypted backups for bare-metal recover (primary wiped).
//
// Security model (0.8.76+):
//   - Auth: Bearer recovery_token (long random in secrets).
//   - Payload: encrypted .ndenc only (+ COMPONENTS.txt). Decryption key is NOT served
//     unless NETDUCTOR_RECOVERY_SERVE_KEY=1 (discouraged). Operator supplies
//     NETDUCTOR_BACKUP_KEY / --key from offline store.
//   - Bind: NETDUCTOR_RECOVERY_BIND (default 0.0.0.0 — needed so a clean primary can pull).
//   - Optional NETDUCTOR_RECOVERY_ALLOW_CIDR (comma CIDRs) — restrict source IPs.
//   - Failed auth: progressive lockout per IP.
//   - Optional NETDUCTOR_RECOVERY_UFW=1 → ufw allow 8790/tcp.
func StartRecoveryServer() {
	tok, err := EnsureRecoveryToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "recovery-serve: token: %v\n", err)
		return
	}
	bind := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_BIND"))
	if bind == "" {
		bind = "0.0.0.0"
	}
	if os.Getenv("NETDUCTOR_RECOVERY_UFW") == "1" {
		_ = exec.Command("ufw", "allow", recoveryPort+"/tcp", "comment", "netductor-recovery").Run()
	}

	var nets []*net.IPNet
	if allow := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_ALLOW_CIDR")); allow != "" {
		for _, p := range strings.Split(allow, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if !strings.Contains(p, "/") {
				p += "/32"
			}
			_, n, err := net.ParseCIDR(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "recovery-serve: bad CIDR %q: %v\n", p, err)
				continue
			}
			nets = append(nets, n)
		}
	}
	serveKey := os.Getenv("NETDUCTOR_RECOVERY_SERVE_KEY") == "1"

	var mu sync.Mutex
	type ipState struct {
		lastOK     time.Time
		fails      int
		lockedUntil time.Time
	}
	state := map[string]*ipState{}

	clientIP := func(r *http.Request) string {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return strings.Trim(r.RemoteAddr, "[]")
		}
		return ip
	}
	ipAllowed := func(ipStr string) bool {
		if len(nets) == 0 {
			return true
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false
		}
		for _, n := range nets {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}

	mux := http.NewServeMux()
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if !ipAllowed(ip) {
				http.Error(w, "forbidden", 403)
				return
			}
			mu.Lock()
			st := state[ip]
			if st == nil {
				st = &ipState{}
				state[ip] = st
			}
			if time.Now().Before(st.lockedUntil) {
				mu.Unlock()
				http.Error(w, "locked", 429)
				return
			}
			// mild throttle successful-path spam
			if !st.lastOK.IsZero() && time.Since(st.lastOK) < 500*time.Millisecond {
				mu.Unlock()
				http.Error(w, "rate", 429)
				return
			}
			got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			if got == "" || got != tok {
				st.fails++
				if st.fails >= 5 {
					st.lockedUntil = time.Now().Add(15 * time.Minute)
					st.fails = 0
				}
				mu.Unlock()
				http.Error(w, "unauthorized", 401)
				return
			}
			st.fails = 0
			st.lastOK = time.Now()
			mu.Unlock()
			next(w, r)
		}
	}

	mux.HandleFunc("/recovery/latest", auth(func(w http.ResponseWriter, r *http.Request) {
		dir := "/var/lib/netductor/backups/peers/core"
		ents, err := os.ReadDir(dir)
		if err != nil {
			http.Error(w, "no backups", 404)
			return
		}
		var latest string
		var latestT int64
		for _, e := range ents {
			n := e.Name()
			if !strings.HasSuffix(n, ".ndenc") && !strings.HasSuffix(n, ".tar.gz") {
				continue
			}
			info, _ := e.Info()
			if info != nil && info.ModTime().Unix() >= latestT {
				latestT = info.ModTime().Unix()
				latest = filepath.Join(dir, n)
			}
		}
		if latest == "" {
			http.Error(w, "no backups", 404)
			return
		}
		w.Header().Set("X-Netductor-Backup-Name", filepath.Base(latest))
		http.ServeFile(w, r, latest)
	}))

	// Key over the wire is opt-in only (discouraged). Prefer offline NETDUCTOR_BACKUP_KEY.
	if serveKey {
		mux.HandleFunc("/recovery/key", auth(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range []string{
				"/var/lib/netductor/backups/peers/core/BACKUP_KEY.txt",
				"/etc/netductor/secrets/backup_key",
			} {
				if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
					w.Header().Set("Content-Type", "text/plain")
					_, _ = w.Write(b)
					return
				}
			}
			http.Error(w, "no key", 404)
		}))
	}

	mux.HandleFunc("/recovery/components", auth(func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile("/var/lib/netductor/backups/peers/core/COMPONENTS.txt")
		if err != nil {
			http.Error(w, "no components", 404)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write(b)
	}))
	mux.HandleFunc("/recovery/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok\n"))
	})

	addr := bind + ":" + recoveryPort
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	extra := "key=off"
	if serveKey {
		extra = "key=ON(discouraged)"
	}
	if len(nets) > 0 {
		extra += fmt.Sprintf(" cidr=%d", len(nets))
	}
	fmt.Fprintf(os.Stderr, "recovery-serve: %s Bearer token; %s\n", addr, extra)
	_ = srv.ListenAndServe()
}
