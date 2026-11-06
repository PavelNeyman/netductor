package upgrade

import (
	"strings"
	"testing"
)

func TestDeclareAndAggregate(t *testing.T) {
	ResetFloorsForTest()
	DeclareBinaryFloor("a", "0.9.100", "first")
	DeclareBinaryFloor("b", "0.9.50", "lower")
	DeclareBinaryFloor("c", "0.9.200", "highest")
	if g := AggregateMinFrom(); g != "0.9.200" {
		t.Fatalf("AggregateMinFrom=%q want 0.9.200", g)
	}
	m := BuildMeta("v0.9.250")
	if m.Version != "0.9.250" || m.MinFrom != "0.9.200" {
		t.Fatalf("meta=%+v", m)
	}
	if len(m.Floors) != 3 {
		t.Fatalf("floors=%d", len(m.Floors))
	}
}

func TestCheckLocalAllowed(t *testing.T) {
	if err := CheckLocalAllowed("0.9.150", ""); err != nil {
		t.Fatal(err)
	}
	if err := CheckLocalAllowed("0.9.150", "0.9.100"); err != nil {
		t.Fatal(err)
	}
	if err := CheckLocalAllowed("0.9.100", "0.9.100"); err != nil {
		t.Fatal(err)
	}
	err := CheckLocalAllowed("0.9.90", "0.9.100")
	if err == nil || !strings.Contains(err.Error(), "min_from") {
		t.Fatalf("want refuse, got %v", err)
	}
	err = CheckLocalAllowed("", "0.9.100")
	if err == nil {
		t.Fatal("empty local must refuse when floor set")
	}
}

func TestDuplicateIDLastWins(t *testing.T) {
	ResetFloorsForTest()
	DeclareBinaryFloor("x", "0.9.1", "a")
	DeclareBinaryFloor("x", "0.9.9", "b")
	fs := Floors()
	if len(fs) != 1 || fs[0].MinBinary != "0.9.9" {
		t.Fatalf("%+v", fs)
	}
}
