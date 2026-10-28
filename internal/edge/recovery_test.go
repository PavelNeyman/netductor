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

func TestBoundBlocksReuse(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	t.Setenv("NETDUCTOR_ETC", t.TempDir())
	t.Setenv("NETDUCTOR_EDGE_DIR", t.TempDir())
	code, _, err := IssueRecoveryCode("s3", "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, rid, ok := ReserveRecoveryCode(code)
	if !ok {
		t.Fatal("reserve")
	}
	if !BindRecoveryCode(code, rid) {
		t.Fatal("bind")
	}
	// Abort must not free bound code
	AbortRecoveryCode(code, rid)
	_, _, ok2 := ReserveRecoveryCode(code)
	if ok2 {
		t.Fatal("bound code must not be reservable again")
	}
	if !CommitRecoveryCode(code, rid) {
		t.Fatal("commit after bind")
	}
}

func TestFinalizeRecoveryAfterEnroll(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	t.Setenv("NETDUCTOR_ETC", t.TempDir())
	t.Setenv("NETDUCTOR_EDGE_DIR", t.TempDir())
	code, _, err := IssueRecoveryCode("sf", "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, rid, ok := ReserveRecoveryCode(code)
	if !ok {
		t.Fatal("reserve")
	}
	if !FinalizeRecoveryAfterEnroll(code, rid) {
		t.Fatal("finalize")
	}
	_, _, ok2 := ReserveRecoveryCode(code)
	if ok2 {
		t.Fatal("used code must not reserve")
	}
	// Wrong reserve id after finalize is no-op success path already used
}

func TestReservedNotStolenAfterTTL(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	t.Setenv("NETDUCTOR_ETC", t.TempDir())
	t.Setenv("NETDUCTOR_EDGE_DIR", t.TempDir())
	code, _, err := IssueRecoveryCode("st", "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, rid, ok := ReserveRecoveryCode(code)
	if !ok {
		t.Fatal("reserve")
	}
	// Backdate ReservedAt beyond the old 120s steal window — must still refuse.
	mu.Lock()
	list, err := loadRecovery()
	if err != nil {
		mu.Unlock()
		t.Fatal(err)
	}
	found := false
	for i := range list {
		if list[i].ReserveID == rid {
			list[i].ReservedAt = time.Now().Unix() - 600 // 10 minutes ago
			found = true
			break
		}
	}
	if !found {
		mu.Unlock()
		t.Fatal("rid not in store")
	}
	if err := saveRecovery(list); err != nil {
		mu.Unlock()
		t.Fatal(err)
	}
	mu.Unlock()

	_, _, ok2 := ReserveRecoveryCode(code)
	if ok2 {
		t.Fatal("must not steal reservation older than 120s")
	}
	if _, okPeek := PeekRecoveryCode(code); okPeek {
		t.Fatal("peek must not treat aged reservation as free")
	}
	AbortRecoveryCode(code, rid)
	_, rid2, ok3 := ReserveRecoveryCode(code)
	if !ok3 || rid2 == "" {
		t.Fatal("after abort can reserve again")
	}
}

func TestDeviceSiteAndStatus(t *testing.T) {
	setup(t)
	Enroll(map[string]any{"device_id": "sitedev"})
	if DeviceStatus("sitedev") != StatusPending {
		t.Fatal(DeviceStatus("sitedev"))
	}
	_, _ = Approve("sitedev")
	if DeviceStatus("sitedev") != StatusApproved {
		t.Fatal(DeviceStatus("sitedev"))
	}
	if DeviceSiteID("sitedev") != "" {
		t.Fatal("expected empty site")
	}
	if err := SetSiteID("sitedev", "home"); err != nil {
		t.Fatal(err)
	}
	if DeviceSiteID("sitedev") != "home" {
		t.Fatal(DeviceSiteID("sitedev"))
	}
}
