package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	bRecMu    sync.Mutex
	bRecSrv   *http.Server
	bRecTok   string
	bRecUntil time.Time
)

func buildRecoveryDir() string {
	return "/var/lib/netductor-build/backups/peers/core"
}

func ensureBuildRecoveryToken() string {
	path := "/etc/netductor/secrets/recovery_token"
	if b, err := os.ReadFile(path); err == nil {
		t := strings.TrimSpace(string(b))
		if len(t) >= 16 {
			return t
		}
	}
	buf := make([]byte, 24)
	_, _ = rand.Read(buf)
	t := hex.EncodeToString(buf)
	_ = os.MkdirAll("/etc/netductor/secrets", 0o700)
	_ = os.WriteFile(path, []byte(t+"\n"), 0o600)
	return t
}

func armBuildRecovery(ttl time.Duration) {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	bRecMu.Lock()
	defer bRecMu.Unlock()
	if bRecSrv != nil {
		_ = bRecSrv.Close()
		bRecSrv = nil
	}
	bRecTok = ensureBuildRecoveryToken()
	bRecUntil = time.Now().Add(ttl)
	mux := http.NewServeMux()
	mux.HandleFunc("/recovery/latest", func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(auth)), []byte(bRecTok)) != 1 {
			http.Error(w, "unauthorized", 401)
			return
		}
		dir := buildRecoveryDir()
		ents, err := os.ReadDir(dir)
		if err != nil || len(ents) == 0 {
			http.Error(w, "no backup", 404)
			return
		}
		var latest string
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			n := e.Name()
			if n > latest {
				latest = n
			}
		}
		if latest == "" {
			http.Error(w, "no backup", 404)
			return
		}
		w.Header().Set("X-Netductor-Backup-Name", latest)
		http.ServeFile(w, r, filepath.Join(dir, latest))
	})
	mux.HandleFunc("/recovery/components", func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(auth)), []byte(bRecTok)) != 1 {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("dirs\nhardening\n"))
	})
	srv := &http.Server{Addr: ":8790", Handler: mux}
	bRecSrv = srv
	go func() {
		fmt.Fprintf(os.Stderr, "build recovery armed until %s on :8790\n", bRecUntil.Format(time.RFC3339))
		_ = srv.ListenAndServe()
	}()
	go func(deadline time.Time) {
		time.Sleep(time.Until(deadline))
		bRecMu.Lock()
		defer bRecMu.Unlock()
		if bRecSrv != nil {
			_ = bRecSrv.Close()
			bRecSrv = nil
			fmt.Fprintln(os.Stderr, "build recovery disarmed")
		}
	}(bRecUntil)
}
