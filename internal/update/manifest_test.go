package update

import (
	"strings"
	"testing"
)

func TestRequiredAssetsIncludeContractNames(t *testing.T) {
	req := RequiredLinuxNodeAssets()
	p := Platform{GOOS: "linux", GOARCH: "amd64"}
	for _, c := range []ComponentID{ComponentNode, ComponentTG, ComponentAgent} {
		name := AssetName(c, p)
		found := false
		for _, r := range req {
			if r == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s not in required set", name)
		}
		if err := ValidateAssetNameMatchesContract(c, p, name); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidateSUMSCoverRequired(t *testing.T) {
	req := RequiredLinuxNodeAssets()
	sums := map[string]string{}
	for _, r := range req {
		if r == "SHA256SUMS" {
			continue
		}
		sums[r] = strings.Repeat("a", 64)
	}
	if m := ValidateSUMSCoverRequired(sums, req); len(m) != 0 {
		t.Fatalf("missing %v", m)
	}
	delete(sums, "netductor-linux-amd64")
	m := ValidateSUMSCoverRequired(sums, req)
	if len(m) != 1 || m[0] != "netductor-linux-amd64" {
		t.Fatalf("%v", m)
	}
}

func TestUserFacingVersionLabel(t *testing.T) {
	st, d := UserFacingVersionLabel("0.9.598", false, "0.9.597")
	if st != "target" || d != "0.9.598" {
		t.Fatalf("%s %s", st, d)
	}
	st, d = UserFacingVersionLabel("0.9.598", true, "0.9.598")
	if st != "running" || d != "0.9.598" {
		t.Fatalf("%s %s", st, d)
	}
}

func TestParseSHA256SUMSLines(t *testing.T) {
	body := "abc123  netductor-linux-amd64\ndef456 ./netductor-tg-linux-amd64\n"
	m := ParseSHA256SUMSLines(body)
	if m["netductor-linux-amd64"] != "abc123" {
		t.Fatalf("%v", m)
	}
}
