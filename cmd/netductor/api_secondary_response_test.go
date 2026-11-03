package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

func TestSecondaryDeviceAPIViewDoesNotExposeAgentSecrets(t *testing.T) {
	d := secondary.Device{
		ID: "secondary-0123456789abcdef",
		Token: "super-secret-agent-token",
		LastCmd: "ssh_keys",
		LastCmdLog: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakePublicKey operator",
	}
	raw, err := json.Marshal(toSecondaryDeviceAPIView(d))
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if strings.Contains(got, d.Token) || strings.Contains(got, "\"token\"") {
		t.Fatalf("API view exposes agent token: %s", got)
	}
	if strings.Contains(got, "AAAAC3NzaC1lZDI1NTE5AAAAIFakePublicKey") {
		t.Fatalf("API view exposes raw authorized_keys log: %s", got)
	}
	if !strings.Contains(got, d.ID) {
		t.Fatalf("API view lost device identity: %s", got)
	}
}
