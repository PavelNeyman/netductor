package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
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

func openBuildRecoveryBackup(path string) (*os.File, unix.Stat_t, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, unix.Stat_t{}, err
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return nil, unix.Stat_t{}, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) {
		_ = unix.Close(fd)
		return nil, unix.Stat_t{}, fmt.Errorf("backup is not a regular file owned by the service user")
	}
	f := os.NewFile(uintptr(fd), path)
	if f == nil {
		_ = unix.Close(fd)
		return nil, unix.Stat_t{}, fmt.Errorf("invalid backup file descriptor")
	}
	return f, st, nil
}

func ensureBuildRecoveryToken() (string, error) {
	const path = "/etc/netductor/secrets/recovery_token"
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err == nil {
		var st unix.Stat_t
		if statErr := unix.Fstat(fd, &st); statErr != nil {
			_ = unix.Close(fd)
			return "", fmt.Errorf("inspect recovery token: %w", statErr)
		}
		if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&0o077 != 0 || st.Size > 4096 {
			_ = unix.Close(fd)
			return "", fmt.Errorf("recovery token must be a private regular file owned by the service user")
		}
		f := os.NewFile(uintptr(fd), path)
		if f == nil {
			_ = unix.Close(fd)
			return "", fmt.Errorf("open recovery token: invalid file descriptor")
		}
		b, readErr := io.ReadAll(io.LimitReader(f, 4097))
		closeErr := f.Close()
		if readErr != nil {
			return "", fmt.Errorf("read recovery token: %w", readErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("close recovery token: %w", closeErr)
		}
		t := strings.TrimSpace(string(b))
		if len(b) > 4096 || len(t) < 16 {
			return "", fmt.Errorf("existing recovery token is invalid; refusing implicit rotation")
		}
		return t, nil
	}
	if err != unix.ENOENT {
		return "", fmt.Errorf("open recovery token: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create recovery secrets directory: %w", err)
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate recovery token: %w", err)
	}
	t := hex.EncodeToString(buf)
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".recovery_token-*")
	if err != nil {
		return "", fmt.Errorf("create recovery token temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("secure recovery token: %w", err)
	}
	if _, err := io.WriteString(tmp, t+"\n"); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("write recovery token: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("sync recovery token: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close recovery token: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", fmt.Errorf("install recovery token: %w", err)
	}
	d, err := os.Open(dir)
	if err != nil {
		return "", fmt.Errorf("open recovery secrets directory: %w", err)
	}
	syncErr := d.Sync()
	closeErr := d.Close()
	if syncErr != nil {
		return "", fmt.Errorf("sync recovery secrets directory: %w", syncErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close recovery secrets directory: %w", closeErr)
	}
	return t, nil
}

func armBuildRecovery(ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	bRecMu.Lock()
	defer bRecMu.Unlock()
	if bRecSrv != nil {
		_ = bRecSrv.Close()
		bRecSrv = nil
	}
	token, err := ensureBuildRecoveryToken()
	if err != nil {
		return err
	}
	bRecTok = token
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
			n := e.Name()
			if e.IsDir() || strings.ContainsAny(n, "\r\n") ||
				(!strings.HasSuffix(n, ".ndenc") && !strings.HasSuffix(n, ".tar.gz")) {
				continue
			}
			if n > latest {
				latest = n
			}
		}
		if latest == "" {
			http.Error(w, "no backup", 404)
			return
		}
		f, st, err := openBuildRecoveryBackup(filepath.Join(dir, latest))
		if err != nil {
			http.Error(w, "backup unavailable", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		w.Header().Set("X-Netductor-Backup-Name", latest)
		http.ServeContent(w, r, latest, time.Unix(st.Mtim.Sec, st.Mtim.Nsec), f)
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
	srv := &http.Server{Addr: ":8790", Handler: mux, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 120 * time.Second}
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
	return nil
}
