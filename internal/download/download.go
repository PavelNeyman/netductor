// Package download is the shared HTTPS (and optional private-HTTP) fetch path
// for install, update, deploy, and agent. R9: one policy for scheme, size, SHA.
package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

func validateDownloadURL(raw string, allowHTTPPrivate bool) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("download: bad url")
	}
	switch u.Scheme {
	case "https":
		return u, nil
	case "http":
		if !allowHTTPPrivate {
			return nil, fmt.Errorf("download: only https URLs allowed")
		}
		ip := net.ParseIP(u.Hostname())
		if ip == nil || !(ip.IsPrivate() || ip.IsLoopback()) {
			return nil, fmt.Errorf("download: http only allowed to private/loopback IP")
		}
		return u, nil
	default:
		return nil, fmt.Errorf("download: only http/https URLs")
	}
}

// NewTransportWithIPFallback returns a transport that retries resolved addresses for
// explicitly allowlisted HTTPS hosts. The request URL and TLS server name remain
// the original hostname, so normal certificate and hostname verification apply.
func NewTransportWithIPFallback(hosts ...string) *http.Transport {
	allow := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
		if host != "" {
			allow[host] = struct{}{}
		}
	}
	return newTransportWithIPFallback(http.DefaultTransport.(*http.Transport).Clone(), allow,
		net.DefaultResolver.LookupIPAddr, (&net.Dialer{Timeout: 10 * time.Second}).DialContext)
}

type ipLookupFunc func(context.Context, string) ([]net.IPAddr, error)
type contextDialFunc func(context.Context, string, string) (net.Conn, error)

func newTransportWithIPFallback(base *http.Transport, allow map[string]struct{}, lookup ipLookupFunc, dial contextDialFunc) *http.Transport {
	transport := base.Clone()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return dial(ctx, network, address)
		}
		if _, ok := allow[strings.ToLower(strings.TrimSuffix(host, "."))]; !ok {
			return dial(ctx, network, address)
		}
		ips, lookupErr := lookup(ctx, host)
		if lookupErr != nil {
			return nil, fmt.Errorf("download: resolve %s: %w", host, lookupErr)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("download: no addresses returned for %s", host)
		}
		var failures []error
		for _, ip := range ips {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			target := net.JoinHostPort(ip.String(), port)
			conn, err := dial(ctx, network, target)
			if err == nil {
				return conn, nil
			}
			failures = append(failures, fmt.Errorf("%s: %w", target, err))
		}
		return nil, fmt.Errorf("download: all addresses for %s failed: %w", host, errors.Join(failures...))
	}
	return transport
}

// Get downloads rawURL to dest with scheme/size policy and optional SHA256 check.
func Get(rawURL, dest string, opt Options) error {
	opt.normalize()

	u, err := validateDownloadURL(rawURL, opt.AllowHTTPPrivate)
	if err != nil {
		return err
	}

	client := &http.Client{
		Transport: NewTransportWithIPFallback("release-assets.githubusercontent.com"),
		Timeout: opt.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("download: too many redirects")
			}
			if _, err := validateDownloadURL(req.URL.String(), opt.AllowHTTPPrivate); err != nil {
				return err
			}
			return nil
		},
	}
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

	dir := filepath.Dir(dest)
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("download: create destination directory: %w", err)
	}
	tf, err := os.CreateTemp(dir, ".nd-dl-*")
	if err != nil {
		return err
	}
	tmp := tf.Name()
	cleanup := func() { _ = os.Remove(tmp) }
	if err := tf.Chmod(0o600); err != nil {
		_ = tf.Close()
		cleanup()
		return fmt.Errorf("download: chmod temporary file: %w", err)
	}
	n, err := io.Copy(tf, io.LimitReader(resp.Body, opt.MaxBytes))
	if err != nil {
		_ = tf.Close()
		cleanup()
		return err
	}
	if n >= opt.MaxBytes {
		_ = tf.Close()
		cleanup()
		return fmt.Errorf("download exceeds %d byte limit", opt.MaxBytes)
	}
	if want := strings.TrimSpace(opt.ExpectedSHA256); want != "" {
		got, err := SHA256File(tmp)
		if err != nil {
			_ = tf.Close()
			cleanup()
			return err
		}
		if !strings.EqualFold(got, want) {
			_ = tf.Close()
			cleanup()
			return fmt.Errorf("checksum mismatch for %s", dest)
		}
	}
	if opt.FileMode != 0 {
		if err := tf.Chmod(opt.FileMode); err != nil {
			_ = tf.Close()
			cleanup()
			return fmt.Errorf("download: chmod destination mode: %w", err)
		}
	}
	if err := tf.Sync(); err != nil {
		_ = tf.Close()
		cleanup()
		return fmt.Errorf("download: sync temporary file: %w", err)
	}
	if err := tf.Close(); err != nil {
		cleanup()
		return fmt.Errorf("download: close temporary file: %w", err)
	}
	// The temporary file lives in the destination directory, so rename is
	// same-filesystem and must remain atomic. Never fall back to in-place write.
	if err := os.Rename(tmp, dest); err != nil {
		cleanup()
		return fmt.Errorf("download: atomic install: %w", err)
	}
	df, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("download: open destination directory after rename: %w", err)
	}
	syncErr := df.Sync()
	closeErr := df.Close()
	if syncErr != nil {
		return fmt.Errorf("download: sync destination directory: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("download: close destination directory: %w", closeErr)
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
