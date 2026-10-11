package svcpaths

import "testing"

func TestPeerOf30(t *testing.T) {
	if peerOf("10.87.10.1/30") != "10.87.10.2" {
		t.Fatal(peerOf("10.87.10.1/30"))
	}
	if peerOf("10.87.10.2/30") != "10.87.10.1" {
		t.Fatal()
	}
	if peerOf("10.0.0.1/24") != "" {
		t.Fatal()
	}
}

func TestDefaultPolicy(t *testing.T) {
	if DefaultPolicy().FailThreshold < 1 {
		t.Fatal()
	}
}
