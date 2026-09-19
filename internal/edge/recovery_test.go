package edge

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRecoveryCodeRoundTrip(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	t.Setenv("NETDUCTOR_ETC", t.TempDir())
	_ = filepath.Join // keep import used if needed
	code, exp, err := IssueRecoveryCode("home", "test", time.Hour)
	if err != nil || code == "" || exp.Before(time.Now()) {
		t.Fatal(code, exp, err)
	}
	if !ValidRecoveryOrBootstrap("Bearer " + code) {
		t.Fatal("valid")
	}
	site, ok := ConsumeRecoveryCode(code)
	if !ok || site != "home" {
		t.Fatal(site, ok)
	}
	if _, ok := ConsumeRecoveryCode(code); ok {
		t.Fatal("reuse")
	}
}
