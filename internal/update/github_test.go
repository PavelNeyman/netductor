package update

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	if !Newer("v0.8.0", "0.7.39-dev") {
		t.Fatal("0.8 > 0.7")
	}
	if Newer("0.8.0", "0.8.0") {
		t.Fatal("equal")
	}
	if Newer("0.7.9", "0.8.0") {
		t.Fatal("older")
	}
}

func TestParseSHA256SUMS(t *testing.T) {
	body := []byte("abc123  netductor-agent-linux-arm64\ndef456 *netductor-agent-linux-amd64\n# comment\n\n")
	m, err := ParseSHA256SUMS(body)
	if err != nil {
		t.Fatal(err)
	}
	if m["netductor-agent-linux-arm64"] != "abc123" {
		t.Fatalf("arm64: %v", m)
	}
	if m["netductor-agent-linux-amd64"] != "def456" {
		t.Fatalf("amd64: %v", m)
	}
	if _, err := ParseSHA256SUMS([]byte("# only\n")); err == nil {
		t.Fatal("expected empty error")
	}
}

func TestAgentAssetName(t *testing.T) {
	if AgentAssetName("mipsle") != "netductor-agent-linux-mipsle" {
		t.Fatal(AgentAssetName("mipsle"))
	}
}

func TestFetchSHA256SUMSBodyRetriesTransientHTTPFailure(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "temporary failure", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  netductor-linux-amd64\n"))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: time.Second}
	body, err := fetchSHA256SUMSBody(client, srv.URL)
	if err != nil {
		t.Fatalf("fetch after transient failure: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls.Load())
	}
	if len(body) == 0 {
		t.Fatal("expected checksum body")
	}
}
