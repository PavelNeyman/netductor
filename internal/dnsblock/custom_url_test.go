package dnsblock

import "testing"

func TestValidateCustomListURL(t *testing.T) {
	if err := validateCustomListURL("https://example.com/list.txt"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		"http://example.com/x",
		"https://localhost/x",
		"https://127.0.0.1/x",
		"https://10.0.0.1/x",
		"https://user:pass@example.com/x",
		"ftp://example.com/x",
	} {
		if err := validateCustomListURL(bad); err == nil {
			t.Fatalf("want err for %s", bad)
		}
	}
}

func TestResolveListURLCatalog(t *testing.T) {
	u, err := resolveListURL("hagezi-multi")
	if err != nil || u == "" {
		t.Fatal(err, u)
	}
}
