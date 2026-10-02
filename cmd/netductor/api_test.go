package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/session"
)

func apiEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	_ = os.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	_ = os.Setenv("NETDUCTOR_STATE", filepath.Join(root, "state"))
	_ = os.Setenv("NETDUCTOR_ROOT", filepath.Join(root, "opt"))
	_ = os.Setenv("NETDUCTOR_EDGE_DIR", filepath.Join(root, "state", "edge"))
	t.Cleanup(func() {
		for _, k := range []string{"NETDUCTOR_ETC", "NETDUCTOR_STATE", "NETDUCTOR_ROOT", "NETDUCTOR_EDGE_DIR"} {
			_ = os.Unsetenv(k)
		}
	})
	_ = os.MkdirAll(filepath.Join(root, "etc", "secrets"), 0o700)
	_ = os.MkdirAll(filepath.Join(root, "etc", "sessions"), 0o700)
	_ = os.MkdirAll(filepath.Join(root, "state", "edge"), 0o700)
	_ = os.MkdirAll(filepath.Join(root, "state"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "etc", "secrets", "edge_bootstrap_token"),
		[]byte("boot-token-at-least-32-characters-xx\n"), 0o600)
}

func TestHealth(t *testing.T) {
	apiEnv(t)
	mux := buildAPIMux()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Code, rr.Body.String())
	}
	var m map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &m)
	if m["ok"] != true {
		t.Fatal(m)
	}
}

func TestSessionRequired(t *testing.T) {
	apiEnv(t)
	mux := buildAPIMux()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/nodes", nil))
	if rr.Code != 401 {
		t.Fatal(rr.Code, rr.Body.String())
	}
}

func TestNodesWithSession(t *testing.T) {
	apiEnv(t)
	mux := buildAPIMux()
	tok, _, err := session.Create(1, "t", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = nodes.SelfRegisterLocal("nd-core-test", "core", "1.2.3.4")
	req := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Code, rr.Body.String())
	}
	var m map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &m)
	if m["nodes"] == nil {
		t.Fatal(m)
	}
}

func TestEnrollAndPending(t *testing.T) {
	apiEnv(t)
	mux := buildAPIMux()
	body := `{"device_id":"site-a","board":"cudy","wan_ip":"10.0.0.1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/edge/enroll", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer boot-token-at-least-32-characters-xx")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 && rr.Code != 201 {
		// may be 200 with pending
		if rr.Code >= 400 {
			t.Fatal(rr.Code, rr.Body.String())
		}
	}
	tok, _, _ := session.Create(1, "t", "")
	req2 := httptest.NewRequest(http.MethodGet, "/api/edge/pending", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatal(rr2.Code, rr2.Body.String())
	}
	// approve
	req3 := httptest.NewRequest(http.MethodPost, "/api/edge/approve", bytes.NewBufferString(`{"device_id":"site-a"}`))
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, req3)
	if rr3.Code != 200 {
		t.Fatal(rr3.Code, rr3.Body.String())
	}
	_ = edge.ListPending() // smoke
}

func TestSessionRevoke(t *testing.T) {
	apiEnv(t)
	mux := buildAPIMux()
	tok, _, err := session.Create(1, "t", "")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/session/revoke", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Code, rr.Body.String())
	}
	if session.Valid(tok) {
		t.Fatal("should be revoked")
	}
}

func TestEnrollUnauthorized(t *testing.T) {
	apiEnv(t)
	mux := buildAPIMux()
	req := httptest.NewRequest(http.MethodPost, "/api/edge/enroll", bytes.NewBufferString(`{"device_id":"x"}`))
	req.Header.Set("Authorization", "Bearer wrong")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatal(rr.Code)
	}
}

func TestTemplatesList(t *testing.T) {
	apiEnv(t)
	mux := buildAPIMux()
	tok, _, err := session.Create(1, "t", "")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/edge/templates", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Code, rr.Body.String())
	}
}

func TestMetricsEndpoint(t *testing.T) {
	apiEnv(t)
	mux := buildAPIMux()
	tok, _, err := session.Create(1, "t", "")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Code, rr.Body.String())
	}
}
