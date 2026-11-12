package channels

import "testing"

func TestUserFromClashMetaDoesNotUseHost(t *testing.T) {
	meta := map[string]any{
		"host":     "chatgpt.com",
		"sourceIP": "46.147.247.69",
		"type":     "vless/relay-in",
	}
	user, uuid := userFromClashMeta(meta)
	if user != "client:46.147.247.69" {
		t.Fatalf("got user=%q uuid=%q, want client:sourceIP", user, uuid)
	}
	meta["user"] = "alice"
	user, _ = userFromClashMeta(meta)
	if user != "alice" {
		t.Fatalf("got %q, want alice", user)
	}
}
