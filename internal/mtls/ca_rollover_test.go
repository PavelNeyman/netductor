package mtls

import (
	"math/big"
	"testing"
)

func TestCARolloverRequiresObservedNewTLSIdentityBeforeFinish(t *testing.T) {
	setupRotationTest(t)
	oldInfo, err := ReadClientInfo("secondary-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := CARolloverStart(); err != nil {
		t.Fatalf("CARolloverStart: %v", err)
	}
	_, _, _, err = IssueClientFromNewCA("secondary-test")
	if err != nil {
		t.Fatalf("IssueClientFromNewCA: %v", err)
	}
	newInfo, err := ReadClientInfo("secondary-test")
	if err != nil {
		t.Fatal(err)
	}
	if newInfo.Serial == oldInfo.Serial {
		t.Fatal("CA rollover did not replace client certificate")
	}
	oldSerial, ok := new(big.Int).SetString(oldInfo.Serial, 16)
	if !ok {
		t.Fatalf("invalid old serial %q", oldInfo.Serial)
	}
	if revoked, err := IsRevoked(oldSerial); err != nil || revoked {
		t.Fatalf("old certificate revoked before adoption: revoked=%v err=%v", revoked, err)
	}
	if err := CARolloverFinish(); err == nil {
		t.Fatal("CA rollover finished before the new TLS identity was observed")
	}
	if confirmed, err := ConfirmCARolloverClient("secondary-test", oldInfo.Serial); err != nil || confirmed {
		t.Fatalf("old TLS identity confirmation = %v, err=%v", confirmed, err)
	}
	confirmed, err := ConfirmCARolloverClient("secondary-test", newInfo.Serial)
	if err != nil || !confirmed {
		t.Fatalf("new TLS identity confirmation = %v, err=%v", confirmed, err)
	}
	if revoked, err := IsRevoked(oldSerial); err != nil || !revoked {
		t.Fatalf("old certificate not revoked after adoption: revoked=%v err=%v", revoked, err)
	}
	if err := CARolloverFinish(); err == nil {
		t.Fatal("CA promotion proceeded without a crash-recovery transaction")
	}
}

func TestCARolloverAbortRefusesIssuedClientIdentity(t *testing.T) {
	setupRotationTest(t)
	if err := CARolloverStart(); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := IssueClientFromNewCA("secondary-test"); err != nil {
		t.Fatal(err)
	}
	if err := CARolloverAbort(); err == nil {
		t.Fatal("abort succeeded after issuing a new-CA client identity")
	}
}

func TestCARolloverCanAbortBeforeIssuingClients(t *testing.T) {
	setupRotationTest(t)
	if err := CARolloverStart(); err != nil {
		t.Fatal(err)
	}
	if err := CARolloverAbort(); err != nil {
		t.Fatalf("abort before issuance: %v", err)
	}
	if err := CARolloverStart(); err != nil {
		t.Fatalf("restart after abort: %v", err)
	}
}
