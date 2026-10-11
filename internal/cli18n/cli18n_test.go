package cli18n

import "testing"

func TestTFallback(t *testing.T) {
	if T("___missing_key_netductor___") == "" {
		t.Fatal()
	}
}

func TestLang(t *testing.T) { _ = Lang() }
