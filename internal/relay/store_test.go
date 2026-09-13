package relay

import "testing"

func TestFindByTokenRejectEmpty(t *testing.T) {
	if FindByToken("") != nil {
		t.Fatal("empty token must not match")
	}
	if FindByToken("short") != nil {
		t.Fatal("short token must not match")
	}
}
