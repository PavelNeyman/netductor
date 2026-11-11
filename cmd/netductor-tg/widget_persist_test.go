package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWidgetUIStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_TELEGRAM", dir)
	setDual(99101, dualPair{InfoID: 11, NavID: 22, InfoShell: shellSpark, NavShell: shellHub})
	setLastAppMsg(99101, 22)
	path := filepath.Join(dir, "widget_ui.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 20 {
		t.Fatalf("short store: %s", b)
	}
	// clear dual entry
	forgetDual(99101)
	setLastAppMsg(99101, 0)
}
