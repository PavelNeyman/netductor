package main

import "testing"

func TestParseCLIOpts(t *testing.T) {
	o := parseCLIOpts([]string{"status", "--json", "extra"})
	if !o.JSON || o.DryRun || o.Help {
		t.Fatalf("opts=%+v", o)
	}
	if len(o.Rest) != 2 || o.Rest[0] != "status" || o.Rest[1] != "extra" {
		t.Fatalf("rest=%v", o.Rest)
	}
	o = parseCLIOpts([]string{"--dry-run", "apply", "v1.2.3", "-h"})
	if !o.DryRun || !o.Help || len(o.Rest) != 2 {
		t.Fatalf("opts=%+v", o)
	}
	o = parseCLIOpts([]string{"-j", "digest"})
	if !o.JSON || o.Rest[0] != "digest" {
		t.Fatalf("opts=%+v", o)
	}
}
