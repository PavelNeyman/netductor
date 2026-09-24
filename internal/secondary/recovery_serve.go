package secondary

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const recoveryPort = "8790"

var (
	recMu      sync.Mutex
	recSrv     *http.Server
	recCancel  context.CancelFunc
	recUntil   time.Time
	recArmed   bool
)

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

// RecoveryStatus returns whether HTTP recovery is currently armed.
func RecoveryStatus() (armed bool, until time.Time) {
	recMu.Lock()
	defer recMu.Unlock()
	return recArmed, recUntil
}

// DisarmRecovery stops the recovery HTTP server and closes ufw rule if we opened it.
func DisarmRecovery() {
	recMu.Lock()
	defer recMu.Unlock()
	disarmLocked()
}

func disarmLocked() {
	if recCancel != nil {
		recCancel()
		recCancel = nil
	}
	if recSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = recSrv.Shutdown(ctx)
		cancel()
		recSrv = nil
	}
	recArmed = false
	recUntil = time.Time{}
	if os.Getenv("NETDUCTOR_RECOVERY_UFW") == "1" {
		_ = exec.Command("ufw", "delete", "allow", recoveryPort+"/tcp").Run()
	}
	fmt.Fprintln(os.Stderr, "recovery-serve: disarmed")
}

// ArmRecovery starts recovery HTTP for ttl (default 30m, max 2h). Idempotent refresh.
func ArmRecovery(ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	if ttl > 2*time.Hour {
		ttl = 2 * time.Hour
	}
	recMu.Lock()
	defer recMu.Unlock()
	if recArmed && recSrv != nil {
		// extend TTL
		if recCancel != nil {
			recCancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		recCancel = cancel
		recUntil = time.Now().Add(ttl)
		go func(deadline time.Time, c context.CancelFunc) {
			t := time.NewTimer(time.Until(deadline))
			defer t.Stop()
			select {
			case <-t.C:
				DisarmRecovery()
			case <-ctx.Done():
			}
		}(recUntil, cancel)
		fmt.Fprintf(os.Stderr, "recovery-serve: TTL extended until %s\n", recUntil.Format(time.RFC3339))
		return nil
	}
	tok, err := EnsureRecoveryToken()
	if err != nil {
		return err
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
		lastOK      time.Time
		fails       int
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
			if !st.lastOK.IsZero() && time.Since(st.lastOK) < 500*time.Millisecond {
				mu.Unlock()
				http.Error(w, "rate", 429)
				return
			}
			got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			if subtle.ConstantTimeCompare([]byte(got), []byte(tok)) != 1 {
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
	ctx, cancel := context.WithCancel(context.Background())
	recSrv = srv
	recCancel = cancel
	recArmed = true
	recUntil = time.Now().Add(ttl)

	go func() {
		fmt.Fprintf(os.Stderr, "recovery-serve: ARMED %s until %s (Bearer token; key on wire=%v)\n",
			addr, recUntil.Format(time.RFC3339), serveKey)
		_ = srv.ListenAndServe()
	}()
	go func(deadline time.Time) {
		t := time.NewTimer(time.Until(deadline))
		defer t.Stop()
		select {
		case <-t.C:
			DisarmRecovery()
		case <-ctx.Done():
		}
	}(recUntil)
	return nil
}

// StartRecoveryServer — legacy name. Default: do NOT listen always.
// NETDUCTOR_RECOVERY_ALWAYS=1 → arm for 24h (tests/legacy).
// Otherwise only knock watcher + manual `netductor recovery arm`.
func StartRecoveryServer() {
	if os.Getenv("NETDUCTOR_RECOVERY_ALWAYS") == "1" {
		_ = ArmRecovery(24 * time.Hour)
		return
	}
	go StartRecoveryKnockWatcher()
	fmt.Fprintln(os.Stderr, "recovery-serve: idle (arm via CLI or port-knock; ALWAYS=1 to force)")
}

// StartRecoveryKnockWatcher listens for a unique TCP sequence then arms recovery.
// Default ports: 41222,41223,41224 (override NETDUCTOR_RECOVERY_KNOCK=p1,p2,p3).
// Window: 5s between knocks. Not the SSH port itself (SSH stays normal auth);
// sequence is separate unused ports so we don't interfere with OpenSSH.
func StartRecoveryKnockWatcher() {
	if os.Getenv("NETDUCTOR_RECOVERY_KNOCK") == "0" {
		return
	}
	raw := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_KNOCK"))
	if raw == "" {
		raw = "41222,41223,41224"
	}
	var ports []int
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			continue
		}
		ports = append(ports, n)
	}
	if len(ports) < 2 {
		fmt.Fprintln(os.Stderr, "recovery-knock: need ≥2 ports")
		return
	}
	type hit struct {
		idx int
		at  time.Time
		ip  string
	}
	var mu sync.Mutex
	progress := map[string]*hit{} // ip → progress

	armTTL := 30 * time.Minute
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_ARM_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			armTTL = d
		}
	}

	for i, port := range ports {
		i, port := i, port
		go func() {
			ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
			if err != nil {
				fmt.Fprintf(os.Stderr, "recovery-knock: listen %d: %v\n", port, err)
				return
			}
			fmt.Fprintf(os.Stderr, "recovery-knock: watching :%d (step %d/%d)\n", port, i+1, len(ports))
			for {
				c, err := ln.Accept()
				if err != nil {
					continue
				}
				ip, _, _ := net.SplitHostPort(c.RemoteAddr().String())
				_ = c.Close()
				mu.Lock()
				h := progress[ip]
				now := time.Now()
				if h == nil || now.Sub(h.at) > 5*time.Second {
					h = &hit{idx: -1, ip: ip}
					progress[ip] = h
				}
				if h.idx+1 == i {
					h.idx = i
					h.at = now
					if h.idx == len(ports)-1 {
						delete(progress, ip)
						mu.Unlock()
						fmt.Fprintf(os.Stderr, "recovery-knock: sequence OK from %s → arm %s\n", ip, armTTL)
						_ = ArmRecovery(armTTL)
						continue
					}
				} else if i == 0 {
					h.idx = 0
					h.at = now
				} else {
					delete(progress, ip)
				}
				mu.Unlock()
			}
		}()
	}
}
