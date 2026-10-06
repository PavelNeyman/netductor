package notify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAlertsChatIDFromSecret(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", dir)
	sec := filepath.Join(dir, "secrets")
	if err := os.MkdirAll(sec, 0o700); err != nil {
		t.Fatal(err)
	}
	if AlertsChatID() != "" {
		t.Fatalf("expected empty, got %q", AlertsChatID())
	}
	if err := os.WriteFile(filepath.Join(sec, "telegram_alerts_chat_id"), []byte("-100123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := AlertsChatID(); got != "-100123" {
		t.Fatalf("got %q", got)
	}
	if !AlertsChannelConfigured() {
		t.Fatal("expected configured")
	}
	st := GetAlertsRoutingStatus()
	if st.Mode != "channel" || !st.ChannelSet {
		t.Fatalf("status %+v", st)
	}
}

func TestSendTestAlertWithoutSecrets(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", dir)
	err := SendTestAlert("unit-test")
	if err == nil {
		t.Fatal("expected error without telegram secrets")
	}
}
