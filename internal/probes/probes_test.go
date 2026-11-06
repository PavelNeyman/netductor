package probes

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultLoadSave(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_ETC", dir)
	t.Cleanup(func() { _ = os.Unsetenv("NETDUCTOR_ETC") })

	d := Default()
	raw, _ := d["probes"].([]any)
	if len(raw) < 3 {
		t.Fatal(d)
	}
	// no file → default
	got := Load()
	if got["probes"] == nil {
		t.Fatal("expected default")
	}
	if err := Save(d); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "probes.json"))
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		t.Fatal(string(b))
	}
	got2 := Load()
	if got2["probes"] == nil {
		t.Fatal()
	}
}

func TestProbeTCPLocal(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	go func() {
		c, _ := ln.Accept()
		if c != nil {
			_ = c.Close()
		}
	}()
	r := probeTCP("127.0.0.1", port, 2*time.Second)
	if ok, _ := r["ok"].(bool); !ok {
		t.Fatalf("%v", r)
	}
}

func TestProbeHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	r := probeHTTP(srv.URL, 2*time.Second, false)
	if ok, _ := r["ok"].(bool); !ok {
		t.Fatalf("%v", r)
	}
	r2 := probeHTTP("http://127.0.0.1:1/", time.Millisecond*200, false)
	if ok, _ := r2["ok"].(bool); ok {
		t.Fatal("expected fail")
	}
}

func TestRunEmpty(t *testing.T) {
	out := Run(map[string]any{})
	if len(out) != 0 {
		t.Fatal(out)
	}
	out = Run(map[string]any{"probes": []any{
		map[string]any{"name": "t", "type": "tcp", "host": "127.0.0.1", "port": 1, "timeout": 0.2},
	}})
	if len(out) != 1 {
		t.Fatal(out)
	}
}

func TestProbeTCPRemoteNotLocalSS(t *testing.T) {
	// Local listener must not make a failed remote dial report ok (CF-1).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	// TEST-NET-1 — not routed; dial fails while local port is listening.
	r := probeTCP("192.0.2.1", port, 300*time.Millisecond)
	ok, _ := r["ok"].(bool)
	if ok {
		t.Fatalf("expected ok=false when remote dial fails despite local listen: %v", r)
	}
	if via, _ := r["via"].(string); via == "ss" {
		t.Fatalf("via=ss must not appear on remote TCP probe: %v", r)
	}
	// Optional: local_listen uses ss; skip assert if ss unavailable in test env.
	loc := probeLocalListen(port, false)
	if ok, _ := loc["ok"].(bool); ok {
		return
	}
	t.Logf("local_listen not detected (ss may be missing): %v", loc)
}
