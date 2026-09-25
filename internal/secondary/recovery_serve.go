package secondary

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
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

var (
	recMu     sync.Mutex
	recSrv    *http.Server
	recCancel context.CancelFunc
	recUntil  time.Time
	recArmed  bool
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
	// Recovery is OFF until arm. While armed (short TTL), default bind is WAN-visible
	// so a wiped primary can pull backup from secondary. Mitigations: TTL, Bearer token,
	// TLS, optional NETDUCTOR_RECOVERY_ALLOW_CIDR. Use BIND=127.0.0.1 only for local tests.
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
			if !recoveryTokenOK(tok, got) {
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
		certFile, keyFile, tlsOn := ensureRecoveryTLS()
		if tlsOn {
			if recoveryBindIsLoopback(bind) {
				fmt.Fprintln(os.Stderr, "recovery-serve: loopback bind (local test; remote recover needs 0.0.0.0 or public IP)")
			}
			fmt.Fprintf(os.Stderr, "recovery-serve: ARMED https://%s until %s (Bearer; key on wire=%v)\n",
				addr, recUntil.Format(time.RFC3339), serveKey)
			_ = srv.ListenAndServeTLS(certFile, keyFile)
			return
		}
		if recoveryBindIsLoopback(bind) {
			fmt.Fprintln(os.Stderr, "recovery-serve: loopback bind (local test; remote recover needs 0.0.0.0 or public IP)")
		}
		fmt.Fprintf(os.Stderr, "recovery-serve: ARMED http://%s until %s (Bearer; key on wire=%v; set RECOVERY_TLS=0 to force HTTP)\n",
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

// StartRecoveryServer does not listen on :8790. Operator arms via SSH:
//
//	netductor recovery arm --ttl 30m
func StartRecoveryServer() {
	fmt.Fprintln(os.Stderr, "recovery-serve: idle (arm via: netductor recovery arm)")
}

const recoveryCert = "/etc/netductor/secrets/recovery.crt"
const recoveryKey = "/etc/netductor/secrets/recovery.key"

// ensureRecoveryTLS returns cert/key paths. Default ON (self-signed). RECOVERY_TLS=0 → plain HTTP.
func ensureRecoveryTLS() (certFile, keyFile string, ok bool) {
	if os.Getenv("NETDUCTOR_RECOVERY_TLS") == "0" {
		return "", "", false
	}
	if st, err := os.Stat(recoveryCert); err == nil && st.Size() > 0 {
		if st2, err2 := os.Stat(recoveryKey); err2 == nil && st2.Size() > 0 {
			return recoveryCert, recoveryKey, true
		}
	}
	_ = os.MkdirAll("/etc/netductor/secrets", 0o700)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "recovery-tls: keygen: %v\n", err)
		return "", "", false
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "netductor-recovery"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "recovery-tls: cert: %v\n", err)
		return "", "", false
	}
	cf, err := os.OpenFile(recoveryCert, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", false
	}
	_ = pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	_ = cf.Close()
	kf, err := os.OpenFile(recoveryKey, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", false
	}
	b, _ := x509.MarshalECPrivateKey(key)
	_ = pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	_ = kf.Close()
	return recoveryCert, recoveryKey, true
}

func recoveryBindIsLoopback(bind string) bool {
	bind = strings.TrimSpace(bind)
	if bind == "" || bind == "127.0.0.1" || bind == "::1" || bind == "localhost" {
		return true
	}
	ip := net.ParseIP(bind)
	return ip != nil && ip.IsLoopback()
}

func recoveryTokenOK(want, got string) bool {
	if want == "" || got == "" {
		return false
	}
	a := sha256.Sum256([]byte(want))
	b := sha256.Sum256([]byte(got))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}
