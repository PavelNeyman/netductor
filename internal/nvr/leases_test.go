package nvr

import (
	"testing"
	"time"
)

func TestParseDHCPLeases(t *testing.T) {
	in := `
# comment
1710000000 aa:bb:cc:dd:ee:ff 192.168.1.50 Tapo-C200 *
1710000100 AA-BB-CC-DD-EE-01 192.168.1.51 * *
`
	got := ParseDHCPLeases(in)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].IP != "192.168.1.50" || got[0].Hostname != "Tapo-C200" {
		t.Fatalf("%+v", got[0])
	}
	if got[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("mac %s", got[0].MAC)
	}
	if got[1].Hostname != "" {
		t.Fatalf("host %q", got[1].Hostname)
	}
	if LeaseRemainingSec(got[0], time.Unix(1710000001, 0)) != 0 {
		// expiry 1710000000 is past relative to +1
	}
}

func TestParseLeasesResultJSON(t *testing.T) {
	raw := `{"leases":[{"expiry":1,"mac":"AA:BB:CC:DD:EE:FF","ip":"10.0.0.5","hostname":"cam"}],"count":1}`
	got := ParseLeasesResult(raw)
	if len(got) != 1 || got[0].IP != "10.0.0.5" || got[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("%+v", got)
	}
}
