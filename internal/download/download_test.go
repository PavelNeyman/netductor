package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetHTTPS(t *testing.T) {
	body := []byte("hello-netductor")
	sum := sha256.Sum256(body)
	want := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "bin")
	err := Get(srv.URL+"/x", dest, Options{
		AllowHTTPPrivate: true,
		ExpectedSHA256:   want,
		FileMode:         0o755,
		MaxBytes:         1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != string(body) {
		t.Fatalf("body %q", got)
	}
}

func TestGetRejectsPlainHTTPPublic(t *testing.T) {
	err := Get("http://example.com/x", filepath.Join(t.TempDir(), "x"), Options{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetSizeLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 100))
	}))
	defer srv.Close()
	err := Get(srv.URL, filepath.Join(t.TempDir(), "x"), Options{
		AllowHTTPPrivate: true,
		MaxBytes:         50,
	})
	if err == nil {
		t.Fatal("expected size error")
	}
}

func TestGetChecksumFailurePreservesExistingDestination(t *testing.T) {
	body := []byte("untrusted-new-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "binary")
	if err := os.WriteFile(dest, []byte("known-good"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Get(srv.URL, dest, Options{
		AllowHTTPPrivate: true,
		ExpectedSHA256:   "0000000000000000000000000000000000000000000000000000000000000000",
		MaxBytes:         1024,
		FileMode:         0o755,
	})
	if err == nil {
		t.Fatal("expected checksum failure")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "known-good" {
		t.Fatalf("existing destination changed on failed download: %q", got)
	}
}

func TestGetLeavesNoTemporaryFileAfterFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("too-large"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := Get(srv.URL, filepath.Join(dir, "binary"), Options{
		AllowHTTPPrivate: true,
		MaxBytes:         4,
	})
	if err == nil {
		t.Fatal("expected size failure")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary files remain after failure: %v", entries)
	}
}

func TestIPFallbackTriesResolvedAddressesInOrder(t *testing.T) {
	var attempted []string
	transport := newTransportWithIPFallback(&http.Transport{}, map[string]struct{}{
		"release-assets.githubusercontent.com": {},
	}, func(ctx context.Context, host string) ([]net.IPAddr, error) {
		if host != "release-assets.githubusercontent.com" {
			t.Fatalf("unexpected lookup host %q", host)
		}
		return []net.IPAddr{{IP: net.ParseIP("192.0.2.10")}, {IP: net.ParseIP("192.0.2.20")}}, nil
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		attempted = append(attempted, address)
		if strings.HasPrefix(address, "192.0.2.10:") {
			return nil, errors.New("unreachable")
		}
		client, server := net.Pipe()
		_ = server.Close()
		return client, nil
	})
	conn, err := transport.DialContext(context.Background(), "tcp", "release-assets.githubusercontent.com:443")
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	if len(attempted) != 2 || attempted[0] != "192.0.2.10:443" || attempted[1] != "192.0.2.20:443" {
		t.Fatalf("unexpected dial sequence: %v", attempted)
	}
}

func TestIPFallbackDoesNotRewriteOrResolveOtherHosts(t *testing.T) {
	var lookedUp bool
	var dialed string
	transport := newTransportWithIPFallback(&http.Transport{}, map[string]struct{}{
		"release-assets.githubusercontent.com": {},
	}, func(ctx context.Context, host string) ([]net.IPAddr, error) {
		lookedUp = true
		return nil, nil
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		dialed = address
		client, server := net.Pipe()
		_ = server.Close()
		return client, nil
	})
	conn, err := transport.DialContext(context.Background(), "tcp", "github.com:443")
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	if lookedUp || dialed != "github.com:443" {
		t.Fatalf("unexpected fallback behavior: lookedUp=%v dialed=%q", lookedUp, dialed)
	}
}

func TestIPFallbackReportsAllDialFailures(t *testing.T) {
	transport := newTransportWithIPFallback(&http.Transport{}, map[string]struct{}{
		"release-assets.githubusercontent.com": {},
	}, func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("192.0.2.10")}, {IP: net.ParseIP("192.0.2.20")}}, nil
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, errors.New("unreachable")
	})
	_, err := transport.DialContext(context.Background(), "tcp", "release-assets.githubusercontent.com:443")
	if err == nil || !strings.Contains(err.Error(), "all addresses") {
		t.Fatalf("expected aggregated dial failure, got %v", err)
	}
}
