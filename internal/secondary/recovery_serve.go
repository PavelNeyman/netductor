package secondary

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

// StartRecoveryServer serves encrypted backups for bare-metal recover (no primary API needed).
// Auth: Bearer recovery_token.
// Bind: NETDUCTOR_RECOVERY_BIND (default 0.0.0.0). Set 127.0.0.1 to lock down.
// Optional: NETDUCTOR_RECOVERY_UFW=1 → ufw allow 8790/tcp
// Optional: NETDUCTOR_RECOVERY_ALLOW_CIDR=1.2.3.4/32 (comma-separated; empty = any)
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
	allow := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_ALLOW_CIDR"))
	var allowList []string
	if allow != "" {
		for _, p := range strings.Split(allow, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				allowList = append(allowList, p)
			}
		}
	}
	var mu sync.Mutex
	last := map[string]time.Time{}

	mux := http.NewServeMux()
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if i := strings.LastIndex(ip, ":"); i > 0 {
				ip = ip[:i]
			}
			ip = strings.Trim(ip, "[]")
			if len(allowList) > 0 {
				ok := false
				for _, a := range allowList {
					if a == ip || strings.HasPrefix(ip, strings.TrimSuffix(a, "/32")) {
						ok = true
						break
					}
				}
				if !ok {
					http.Error(w, "forbidden", 403)
					return
				}
			}
			mu.Lock()
			if t, hit := last[ip]; hit && time.Since(t) < 2*time.Second {
				mu.Unlock()
				http.Error(w, "rate", 429)
				return
			}
			last[ip] = time.Now()
			mu.Unlock()

			got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			if got == "" || got != tok {
				http.Error(w, "unauthorized", 401)
				return
			}
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
	fmt.Fprintf(os.Stderr, "recovery-serve: %s (Bearer recovery_token)\n", addr)
	_ = srv.ListenAndServe()
}
