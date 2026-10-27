package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseKV(t *testing.T) {
	var c config
	parseKV(`
# comment
SERVER=https://example.com
TOKEN=secret
DEVICE_ID=ow-1
INTERVAL=30
BADLINE
INTERVAL=nope
`, &c)
	if c.Server != "https://example.com" || c.Token != "secret" || c.DeviceID != "ow-1" || c.Interval != 30 {
		t.Fatalf("%+v", c)
	}
}

func TestLoadConfigEnv(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "cfg")
	_ = os.WriteFile(conf, []byte("SERVER=http://127.0.0.1:8787\nTOKEN=tok\n"), 0o600)
	_ = os.Setenv("NETDUCTOR_AGENT_CONF", conf)
	_ = os.Setenv("NETDUCTOR_DEVICE_ID", "dev-env")
	_ = os.Setenv("NETDUCTOR_INTERVAL", "15")
	t.Cleanup(func() {
		_ = os.Unsetenv("NETDUCTOR_AGENT_CONF")
		_ = os.Unsetenv("NETDUCTOR_DEVICE_ID")
		_ = os.Unsetenv("NETDUCTOR_INTERVAL")
	})
	c := loadConfig()
	if c.Server != "http://127.0.0.1:8787" || c.Token != "tok" || c.DeviceID != "dev-env" || c.Interval != 15 {
		t.Fatalf("%+v", c)
	}
}

func TestTruncate(t *testing.T) {
	if truncate("  hi  ", 10) != "hi" {
		t.Fatal()
	}
	s := truncate("abcdefghijKLM", 5)
	if s != "abcde…" {
		t.Fatal(s)
	}
}

func TestFileSHA256(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	_ = os.WriteFile(p, []byte("abc"), 0o644)
	got, err := fileSHA256(p)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("abc"))
	if got != hex.EncodeToString(sum[:]) {
		t.Fatal(got)
	}
}

func TestDoJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer t" {
			w.WriteHeader(401)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()
	b, err := doJSON(srv.Client(), http.MethodGet, srv.URL, "t", nil)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["ok"] != true {
		t.Fatal(m)
	}
	_, err = doJSON(srv.Client(), http.MethodGet, srv.URL, "bad", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCollectMetricsShape(t *testing.T) {
	m := collectMetrics()
	if m == nil {
		t.Fatal()
	}
}

func TestEnsureEnrolledPendingAndApproved(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_AGENT_DIR", dir)
	t.Cleanup(func() { _ = os.Unsetenv("NETDUCTOR_AGENT_DIR") })

	state := "pending"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/edge/enroll" && r.Method == http.MethodPost:
			if r.Header.Get("Authorization") != "Bearer boot" {
				w.WriteHeader(401)
				return
			}
			if state == "pending" {
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "pending"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "approved", "device_token": "devtok-32-chars-xxxxxxxxxxxxxxx"})
		case r.URL.Path == "/api/edge/token-ack" && r.Method == http.MethodPost:
			if r.Header.Get("Authorization") != "Bearer devtok-32-chars-xxxxxxxxxxxxxxx" {
				w.WriteHeader(401)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/api/edge/heartbeat":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/api/edge/commands":
			_ = json.NewEncoder(w).Encode(map[string]any{"commands": []any{
				map[string]any{"id": "1", "action": "ping", "arg": ""},
			}})
		case r.URL.Path == "/api/edge/cmd_result":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	cfg := config{Server: srv.URL, Token: "boot", DeviceID: "site1", Interval: 1}
	err := ensureEnrolled(srv.Client(), &cfg)
	if err == nil || !strings.Contains(err.Error(), "pending") {
		t.Fatalf("want pending, got %v", err)
	}

	state = "approved"
	if err := ensureEnrolled(srv.Client(), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "devtok-32-chars-xxxxxxxxxxxxxxx" {
		t.Fatal(cfg.Token)
	}
	if loadDeviceToken() == "" {
		t.Fatal("token not saved")
	}
	// second call uses saved token
	cfg.Token = "boot"
	if err := ensureEnrolled(srv.Client(), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "devtok-32-chars-xxxxxxxxxxxxxxx" {
		t.Fatal("should load saved token")
	}

	if err := heartbeat(srv.Client(), cfg); err != nil {
		t.Fatal(err)
	}
	if err := pollCmds(srv.Client(), cfg); err != nil {
		t.Fatal(err)
	}
}

func TestRunCmdPing(t *testing.T) {
	if runCmd(nil, config{}, "ping", "") != "pong" {
		t.Fatal()
	}
	out := runCmd(nil, config{}, "status", "")
	if out == "" || out[0] != '{' {
		t.Fatal(out)
	}
}
