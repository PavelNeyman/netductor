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
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"crypto/tls"
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
	const dir = "/etc/netductor/secrets"
	path := filepath.Join(dir, "recovery_token")
	if b, err := os.ReadFile(path); err == nil {
		t := strings.TrimSpace(string(b))
		if len(t) >= 16 {
			return t, nil
		}
		return "", fmt.Errorf("existing recovery token is invalid")
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read recovery token: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create recovery secrets directory: %w", err)
	}
	buf := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generate recovery token: %w", err)
	}
	t := hex.EncodeToString(buf)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return "", fmt.Errorf("read concurrently created recovery token: %w", readErr)
			}
			existing := strings.TrimSpace(string(b))
			if len(existing) < 16 {
				return "", fmt.Errorf("concurrently created recovery token is invalid")
			}
			return existing, nil
		}
		return "", fmt.Errorf("create recovery token: %w", err)
	}
	if _, err := io.WriteString(f, t+"\n"); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write recovery token: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("sync recovery token: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close recovery token: %w", err)
	}
	d, err := os.Open(dir)
	if err != nil {
		return "", fmt.Errorf("open recovery secrets directory for sync: %w", err)
	}
	if err := d.Sync(); err != nil {
		_ = d.Close()
		return "", fmt.Errorf("sync recovery secrets directory: %w", err)
	}
	if err := d.Close(); err != nil {
		return "", fmt.Errorf("close recovery secrets directory: %w", err)
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
	var nets []*net.IPNet
	if allow := strings.TrimSpace(os.Getenv("NETDUCTOR_RECOVERY_ALLOW_CIDR")); allow != "" {
		var parseErrs []string
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
				parseErrs = append(parseErrs, fmt.Sprintf("%q: %v", p, err))
				continue
			}
			nets = append(nets, n)
		}
		if len(nets) == 0 {
			return fmt.Errorf("NETDUCTOR_RECOVERY_ALLOW_CIDR set but no valid CIDR (%v)", parseErrs)
		}
		if len(parseErrs) > 0 {
			return fmt.Errorf("NETDUCTOR_RECOVERY_ALLOW_CIDR has invalid entries: %v", parseErrs)
		}
	}
	if os.Getenv("NETDUCTOR_RECOVERY_UFW") == "1" {
		if err := exec.Command("ufw", "allow", recoveryPort+"/tcp", "comment", "netductor-recovery").Run(); err != nil {
			return fmt.Errorf("recovery ufw allow %s: %w", recoveryPort, err)
		}
	}
	serveKey := false // backup key is offline-only; SERVE_KEY removed

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
	mux.HandleFunc("/recovery/upload", auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}
		const maxUpload = 512 << 20 // 512 MiB
		dir := "/var/lib/netductor/backups/peers/core"
		_ = os.MkdirAll(dir, 0o700)
		name := r.Header.Get("X-Netductor-Backup-Name")
		if name == "" {
			name = "netductor-upload-" + time.Now().UTC().Format("20060102-150405") + ".ndenc"
		}
		name = filepath.Base(name)
		if name == "." || name == ".." || name == "" {
			http.Error(w, "bad name", 400)
			return
		}
		// refuse overwriting critical sidecars without explicit name policy
		base := strings.ToLower(name)
		if base == "backup_key.txt" || base == "components.txt" {
			http.Error(w, "reserved name", 400)
			return
		}
		outPath := filepath.Join(dir, name)
		tmp, err := os.CreateTemp(dir, ".upload-*")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		tmpName := tmp.Name()
		r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
		n, err := io.Copy(tmp, r.Body)
		cerr := tmp.Close()
		if err != nil {
			_ = os.Remove(tmpName)
			http.Error(w, err.Error(), 400)
			return
		}
		if cerr != nil {
			_ = os.Remove(tmpName)
			http.Error(w, cerr.Error(), 500)
			return
		}
		if n == 0 {
			_ = os.Remove(tmpName)
			http.Error(w, "empty body", 400)
			return
		}
		if err := os.Rename(tmpName, outPath); err != nil {
			_ = os.Remove(tmpName)
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "stored %s (%d bytes)\n", outPath, n)
	}))
	mux.HandleFunc("/recovery/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok\n"))
	})

	addr := bind + ":" + recoveryPort
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Minute, WriteTimeout: 5 * time.Minute}
	ctx, cancel := context.WithCancel(context.Background())
	recSrv = srv
	recCancel = cancel
	recArmed = true
	recUntil = time.Now().Add(ttl)

	certFile, keyFile, tlsOn, tlsErr := ensureRecoveryTLS()
	if os.Getenv("NETDUCTOR_RECOVERY_TLS") != "0" && (tlsErr != nil || !tlsOn) {
		disarmLocked()
		if tlsErr != nil {
			return fmt.Errorf("recovery TLS required: %w", tlsErr)
		}
		return fmt.Errorf("recovery TLS required but certs unavailable")
	}
	go func() {
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
		fmt.Fprintf(os.Stderr, "recovery-serve: ARMED http://%s until %s (Bearer; key on wire=%v; NETDUCTOR_RECOVERY_TLS=0)\n",
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
func ensureRecoveryTLS() (certFile, keyFile string, ok bool, err error) {
	if os.Getenv("NETDUCTOR_RECOVERY_TLS") == "0" {
		return "", "", false, nil
	}
	const dir = "/etc/netductor/secrets"
	if err := ensureRecoverySecretsDir(dir); err != nil {
		return "", "", false, err
	}

	certInfo, certErr := os.Lstat(recoveryCert)
	keyInfo, keyErr := os.Lstat(recoveryKey)
	certExists := certErr == nil
	keyExists := keyErr == nil
	if certErr != nil && !os.IsNotExist(certErr) {
		return "", "", false, fmt.Errorf("inspect recovery certificate: %w", certErr)
	}
	if keyErr != nil && !os.IsNotExist(keyErr) {
		return "", "", false, fmt.Errorf("inspect recovery private key: %w", keyErr)
	}
	if certExists != keyExists {
		return "", "", false, fmt.Errorf("recovery TLS certificate/key pair is incomplete; refusing implicit rotation")
	}
	if certExists {
		if err := validateRecoveryTLSFile(recoveryCert, certInfo, false); err != nil {
			return "", "", false, fmt.Errorf("validate recovery certificate: %w", err)
		}
		if err := validateRecoveryTLSFile(recoveryKey, keyInfo, true); err != nil {
			return "", "", false, fmt.Errorf("validate recovery private key: %w", err)
		}
		pair, err := tls.LoadX509KeyPair(recoveryCert, recoveryKey)
		if err != nil {
			return "", "", false, fmt.Errorf("load recovery TLS key pair: %w", err)
		}
		if len(pair.Certificate) == 0 {
			return "", "", false, fmt.Errorf("recovery TLS certificate chain is empty")
		}
		cert, err := x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			return "", "", false, fmt.Errorf("parse recovery TLS certificate: %w", err)
		}
		now := time.Now()
		if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
			return "", "", false, fmt.Errorf("recovery TLS certificate is outside its validity period")
		}
		if err := writeRecoveryFingerprint(cert.Raw); err != nil {
			return "", "", false, fmt.Errorf("persist recovery TLS fingerprint: %w", err)
		}
		return recoveryCert, recoveryKey, true, nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", false, fmt.Errorf("generate recovery TLS key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", false, fmt.Errorf("generate recovery TLS serial: %w", err)
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "netductor-recovery"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", false, fmt.Errorf("create recovery TLS certificate: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", false, fmt.Errorf("marshal recovery TLS private key: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		return "", "", false, fmt.Errorf("encode recovery TLS PEM failed")
	}
	// Stage both files durably before publishing either final path. Hard links
	// provide no-replace semantics; a crash between links leaves an incomplete
	// pair, which the next startup rejects rather than silently rotating.
	certTmp, err := writeRecoveryTemp(dir, ".recovery-cert-*", certPEM, 0o600)
	if err != nil {
		return "", "", false, fmt.Errorf("stage recovery certificate: %w", err)
	}
	defer os.Remove(certTmp)
	keyTmp, err := writeRecoveryTemp(dir, ".recovery-key-*", keyPEM, 0o600)
	if err != nil {
		return "", "", false, fmt.Errorf("stage recovery private key: %w", err)
	}
	defer os.Remove(keyTmp)
	if err := os.Link(certTmp, recoveryCert); err != nil {
		return "", "", false, fmt.Errorf("publish recovery certificate without replacement: %w", err)
	}
	if err := os.Link(keyTmp, recoveryKey); err != nil {
		_ = os.Remove(recoveryCert)
		return "", "", false, fmt.Errorf("publish recovery private key without replacement: %w", err)
	}
	if err := syncRecoveryDir(dir); err != nil {
		return "", "", false, fmt.Errorf("sync recovery TLS directory: %w", err)
	}
	if _, err := tls.LoadX509KeyPair(recoveryCert, recoveryKey); err != nil {
		_ = os.Remove(recoveryCert)
		_ = os.Remove(recoveryKey)
		_ = syncRecoveryDir(dir)
		return "", "", false, fmt.Errorf("verify newly created recovery TLS pair: %w", err)
	}
	sum := sha256.Sum256(der)
	fp := fmt.Sprintf("sha256:%x", sum[:])
	if err := writeRecoveryFingerprint(der); err != nil {
		return "", "", false, fmt.Errorf("persist recovery TLS fingerprint: %w", err)
	}
	fmt.Fprintf(os.Stderr, "recovery-tls: cert %s\nrecovery-tls: fingerprint %s\n  on primary: echo %s > /etc/netductor/secrets/recovery_peer.fp\n", recoveryCert, fp, fp)
	return recoveryCert, recoveryKey, true, nil
}

func ensureRecoverySecretsDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create secrets directory: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect secrets directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("secrets path is not a private directory")
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 {
		return fmt.Errorf("secrets directory is not owned by root")
	}
	return nil
}

func validateRecoveryTLSFile(path string, info os.FileInfo, private bool) error {
	if info == nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is not a regular non-symlink file", path)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%s has group/other permissions", path)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 {
		return fmt.Errorf("%s is not owned by root", path)
	}
	if private && info.Size() > 1<<20 {
		return fmt.Errorf("%s exceeds size limit", path)
	}
	if info.Size() == 0 || info.Size() > 1<<20 {
		return fmt.Errorf("%s has invalid size", path)
	}
	return nil
}

func writeRecoveryTemp(dir, pattern string, data []byte, mode os.FileMode) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	path := f.Name()
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if err := f.Chmod(mode); err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	if err := f.Sync(); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	ok = true
	return path, nil
}

func syncRecoveryDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	if err := d.Sync(); err != nil {
		_ = d.Close()
		return err
	}
	return d.Close()
}

func writeRecoveryFingerprint(der []byte) error {
	sum := sha256.Sum256(der)
	data := []byte(fmt.Sprintf("sha256:%x\\n", sum[:]))
	dir := filepath.Dir("/etc/netductor/secrets/recovery.fp")
	tmp, err := writeRecoveryTemp(dir, ".recovery-fp-*", data, 0o600)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if err := os.Rename(tmp, "/etc/netductor/secrets/recovery.fp"); err != nil {
		return err
	}
	return syncRecoveryDir(dir)
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
