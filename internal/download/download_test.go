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
