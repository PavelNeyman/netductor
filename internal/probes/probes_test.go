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
