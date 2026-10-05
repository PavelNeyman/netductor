// Package download is the shared HTTPS (and optional private-HTTP) fetch path
// for install, update, deploy, and agent. R9: one policy for scheme, size, SHA.
package download

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// DefaultMaxBytes is 512 MiB (install/op path). Agent firmware uses a lower cap via Options.
const DefaultMaxBytes int64 = 512 << 20

// Options controls a single Get.
type Options struct {
	// MaxBytes hard-caps the response body (0 → DefaultMaxBytes).
	MaxBytes int64
	// Timeout for the HTTP client (0 → 10m).
	Timeout time.Duration
	// UserAgent header (empty → netductor-download).
	UserAgent string
	// AllowHTTPPrivate permits http:// only to private/loopback literal IPs (edge agent path).
	AllowHTTPPrivate bool
	// ExpectedSHA256, if non-empty, must match the file (hex, case-insensitive).
	ExpectedSHA256 string
	// FileMode for the destination after success (0 → leave Create default / chmod skipped).
	FileMode os.FileMode
}

func (o *Options) normalize() {
	if o.MaxBytes <= 0 {
		o.MaxBytes = DefaultMaxBytes
	}
	if o.Timeout <= 0 {
		o.Timeout = 10 * time.Minute
	}
	if o.UserAgent == "" {
		o.UserAgent = "netductor-download"
	}
}

// Get downloads rawURL to dest with scheme/size policy and optional SHA256 check.
func Get(rawURL, dest string, opt Options) error {
	opt.normalize()

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("download: bad url")
	}
	switch u.Scheme {
	case "https":
		// ok
	case "http":
		if !opt.AllowHTTPPrivate {
			return fmt.Errorf("download: only https URLs allowed")
		}
		ip := net.ParseIP(u.Hostname())
		if ip == nil || !(ip.IsPrivate() || ip.IsLoopback()) {
			return fmt.Errorf("download: http only allowed to private/loopback IP")
		}
	default:
		return fmt.Errorf("download: only http/https URLs")
	}

	client := &http.Client{Timeout: opt.Timeout}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", opt.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: HTTP %d %s", resp.StatusCode, rawURL)
	}

	tmp := dest + ".new"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, opt.MaxBytes))
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n >= opt.MaxBytes {
		_ = os.Remove(tmp)
		return fmt.Errorf("download exceeds %d byte limit", opt.MaxBytes)
	}

	if want := strings.TrimSpace(opt.ExpectedSHA256); want != "" {
		got, err := SHA256File(tmp)
		if err != nil {
			_ = os.Remove(tmp)
			return err
		}
		if !strings.EqualFold(got, want) {
			_ = os.Remove(tmp)
			return fmt.Errorf("checksum mismatch for %s", dest)
		}
	}

	if opt.FileMode != 0 {
		_ = os.Chmod(tmp, opt.FileMode)
	}
	if err := os.Rename(tmp, dest); err != nil {
		// cross-device fallback
		in, rerr := os.ReadFile(tmp)
		if rerr != nil {
			_ = os.Remove(tmp)
			return err
		}
		mode := opt.FileMode
		if mode == 0 {
			mode = 0o644
		}
		if werr := os.WriteFile(dest, in, mode); werr != nil {
			_ = os.Remove(tmp)
			return werr
		}
		_ = os.Remove(tmp)
	}
	return nil
}

// SHA256File returns the hex digest of path.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
