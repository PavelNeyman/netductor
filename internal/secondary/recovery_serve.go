package secondary

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
// Auth: Bearer recovery_token. Bind: all interfaces :8790 (operator should firewall to trusted IPs).
func StartRecoveryServer() {
	tok, err := EnsureRecoveryToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "recovery-serve: token: %v\n", err)
		return
	}
	mux := http.NewServeMux()
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
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
	srv := &http.Server{Addr: ":" + recoveryPort, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	fmt.Fprintf(os.Stderr, "recovery-serve: :%s (Bearer recovery_token)\n", recoveryPort)
	_ = srv.ListenAndServe()
}
