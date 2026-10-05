package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/PavelNeyman/netductor/internal/session"
)

func TestEnvOr(t *testing.T) {
	_ = os.Setenv("ND_TEST_X", "yes")
	t.Cleanup(func() { _ = os.Unsetenv("ND_TEST_X") })
	if envOr("ND_TEST_X", "no") != "yes" {
		t.Fatal()
	}
	if envOr("ND_TEST_MISSING", "def") != "def" {
		t.Fatal()
	}
}

func TestWriteReadJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, 200, map[string]any{"ok": true})
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "json") {
		t.Fatal(rr.Header())
	}
	body := `{"id":"x","hostname":"nd-core-1"}`
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	m := readJSON(r)
	if m["id"] != "x" {
		t.Fatal(m)
	}
}

func TestRequireSession(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_ETC", dir)
	t.Cleanup(func() { _ = os.Unsetenv("NETDUCTOR_ETC") })
	_ = os.MkdirAll(dir+"/sessions", 0o700)
	tok, _, err := session.Create(1, "t", "")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if requireSession(rr, r) {
		t.Fatal("no token")
	}
	rr2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.Header.Set("Authorization", "Bearer "+tok)
	if !requireSession(rr2, r2) {
		t.Fatal("with token")
	}
	_ = json.Valid([]byte(`{}`))
}

func TestAPIPathExemptFromSession(t *testing.T) {
	if !apiPathExemptFromSession("/api/edge/heartbeat") {
		t.Fatal("heartbeat should be exempt")
	}
	if !apiPathExemptFromSession("/api/secondary/agent/heartbeat") {
		t.Fatal("secondary agent should be exempt")
	}
	if apiPathExemptFromSession("/api/stack/apply") {
		t.Fatal("stack apply must require session")
	}
}
