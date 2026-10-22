package install

import "testing"

func TestValidHostnameStrict(t *testing.T) {
	ok := []string{"nd-primary", "host.example.com", "a1"}
	bad := []string{"", "has space", "quo'te", "semi;colon", "a$(rm)", "../../../etc", "x\ny"}
	for _, s := range ok {
		if !validHostnameStrict(s) {
			t.Fatalf("want ok %q", s)
		}
	}
	for _, s := range bad {
		if validHostnameStrict(s) {
			t.Fatalf("want reject %q", s)
		}
	}
}
