package controlplane

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest	"os"
	"path/filepath"
	"testing"
)

func TestPullLatestSHA(t *testing.T) {
	payload := []byte("ndenc-test-payload")
	sum := sha256.Sum256(payload)
	sha := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/control/backup/latest" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "auth", 401)
			return
		}
		w.Header().Set("X-Netductor-Backup-Name", "test.ndenc")
		w.Header().Set("X-Netductor-Backup-SHA256", sha)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	path, name, got, err := PullLatest(srv.URL, "tok", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if name != "test.ndenc" || got != sha {
		t.Fatalf("name=%s sha=%s", name, got)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != string(payload) {
		t.Fatalf("file %v", err)
	}
	if filepath.Dir(path) != InboxDir() {
		t.Fatalf("inbox %s", path)
	}
}
