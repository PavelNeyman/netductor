package channels

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndReadHistory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	pt := SamplePoint{
		TS:   time.Now().UTC(),
		Node: "primary",
		Channels: []ChannelSample{
			{ID: "C2", Peer: "sec-a", OK: true, RTTMs: 42},
			{ID: "C7", OK: true, RxBps: 1e6, TxBps: 2e5, Note: "iface=eth0"},
		},
	}
	if err := AppendHistory(pt); err != nil {
		t.Fatal(err)
	}
	path := historyPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHistory(time.Now().UTC().Add(-time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Channels) != 2 {
		t.Fatalf("got %+v", got)
	}
	gotC2, err := ReadHistory(time.Time{}, "C2")
	if err != nil || len(gotC2) != 1 || gotC2[0].Channels[0].ID != "C2" {
		t.Fatalf("filter: %+v err=%v", gotC2, err)
	}
	_ = filepath.Join(dir, "metrics")
}

func TestFormatHistoryEmpty(t *testing.T) {
	s := FormatHistoryText(nil)
	if s == "" {
		t.Fatal("expected empty message")
	}
}
