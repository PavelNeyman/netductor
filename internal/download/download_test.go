package download

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
