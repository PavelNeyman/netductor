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

func TestReserveRecoveryConcurrent(t *testing.T) {
	setup := func() {
		t.Setenv("NETDUCTOR_STATE", t.TempDir())
		t.Setenv("NETDUCTOR_ETC", t.TempDir())
		t.Setenv("NETDUCTOR_EDGE_DIR", t.TempDir())
	}
	setup()
	code, _, err := IssueRecoveryCode("s1", "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, rid1, ok1 := ReserveRecoveryCode(code)
	if !ok1 || rid1 == "" {
		t.Fatal("first reserve")
	}
	_, _, ok2 := ReserveRecoveryCode(code)
	if ok2 {
		t.Fatal("second reserve must fail while first held")
	}
	if !CommitRecoveryCode(code, rid1) {
		t.Fatal("commit")
	}
	_, _, ok3 := ReserveRecoveryCode(code)
	if ok3 {
		t.Fatal("after commit must fail")
	}
}

func TestReserveAbortAllowsRetry(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	t.Setenv("NETDUCTOR_ETC", t.TempDir())
	t.Setenv("NETDUCTOR_EDGE_DIR", t.TempDir())
	code, _, err := IssueRecoveryCode("s2", "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, rid, ok := ReserveRecoveryCode(code)
	if !ok {
		t.Fatal("reserve")
	}
	AbortRecoveryCode(code, rid)
	site, rid2, ok2 := ReserveRecoveryCode(code)
	if !ok2 || site != "s2" || rid2 == "" {
		t.Fatal("retry after abort", site, ok2)
	}
	if !CommitRecoveryCode(code, rid2) {
		t.Fatal("commit")
	}
}
