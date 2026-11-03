package mtls

import (
	"math/big"
	"os"
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
	if _, _, _, err := IssueClientFromNewCA("secondary-test"); err != nil {
		t.Fatalf("idempotent issuance retry: %v", err)
	}
	retryInfo, err := ReadClientInfo("secondary-test")
	if err != nil {
		t.Fatal(err)
	}
	if retryInfo.Serial != newInfo.Serial {
		t.Fatalf("issuance retry changed serial: got %s want %s", retryInfo.Serial, newInfo.Serial)
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
	if err := CARolloverFinish(); err != nil {
		t.Fatalf("CARolloverFinish after verified adoption: %v", err)
	}
	if status := CARolloverStatus(); status["phase"] != "done" {
		t.Fatalf("CA rollover phase after promotion = %v, want done", status["phase"])
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

func TestCARolloverFailsClosedOnCorruptJournal(t *testing.T) {
	setupRotationTest(t)
	if err := os.WriteFile(rolloverPath(), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CARolloverStart(); err == nil {
		t.Fatal("CA rollover start accepted a corrupt journal")
	}
	if status := CARolloverStatus(); status["error"] == nil {
		t.Fatalf("CA rollover status did not report corrupt journal: %+v", status)
	}
}

func TestCARolloverRetriesOldSerialRevocationAfterDurableConfirmation(t *testing.T) {
	setupRotationTest(t)
	oldInfo, err := ReadClientInfo("secondary-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := CARolloverStart(); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := IssueClientFromNewCA("secondary-test"); err != nil {
		t.Fatal(err)
	}
	newInfo, err := ReadClientInfo("secondary-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(revokePath(), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if confirmed, err := ConfirmCARolloverClient("secondary-test", newInfo.Serial); err == nil || confirmed {
		t.Fatalf("confirmation should report revocation persistence failure: confirmed=%v err=%v", confirmed, err)
	}
	if err := os.WriteFile(revokePath(), []byte("{\"entries\":[]}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if confirmed, err := ConfirmCARolloverClient("secondary-test", newInfo.Serial); err != nil || !confirmed {
		t.Fatalf("confirmation retry = %v, err=%v", confirmed, err)
	}
	oldSerial, ok := new(big.Int).SetString(oldInfo.Serial, 16)
	if !ok {
		t.Fatalf("invalid old serial %q", oldInfo.Serial)
	}
	if revoked, err := IsRevoked(oldSerial); err != nil || !revoked {
		t.Fatalf("old certificate revoked after retry = %v, err=%v", revoked, err)
	}
}

func TestEnsureAllRollsBackInterruptedCAPromotion(t *testing.T) {
	setupRotationTest(t)
	oldCA, err := os.ReadFile(path(CACertFile))
	if err != nil {
		t.Fatal(err)
	}
	oldKey, err := os.ReadFile(path(CAKeyFile))
	if err != nil {
		t.Fatal(err)
	}
	oldServer, err := os.ReadFile(path(ServerCertFile))
	if err != nil {
		t.Fatal(err)
	}
	oldServerKey, err := os.ReadFile(path(ServerKeyFile))
	if err != nil {
		t.Fatal(err)
	}
	oldClient, err := os.ReadFile(path(ClientCertFile))
	if err != nil {
		t.Fatal(err)
	}
	oldClientKey, err := os.ReadFile(path(ClientKeyFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := CARolloverStart(); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{
		{CACertFile, rolloverBackupCACert},
		{CAKeyFile, rolloverBackupCAKey},
		{ServerCertFile, rolloverBackupServerCert},
		{ServerKeyFile, rolloverBackupServerKey},
		{ClientCertFile, rolloverBackupClientCert},
		{ClientKeyFile, rolloverBackupClientKey},
	} {
		if err := copyDurable(path(pair[0]), path(pair[1])); err != nil {
			t.Fatalf("backup %s: %v", pair[0], err)
		}
	}
	state, err := loadRollover()
	if err != nil {
		t.Fatal(err)
	}
	state.Phase = "promoting"
	if err := saveRollover(state); err != nil {
		t.Fatal(err)
	}
	newCA, err := os.ReadFile(path(CANewCertFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := writeDurable(path(CACertFile), newCA, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureAll("127.0.0.1"); err != nil {
		t.Fatalf("EnsureAll recovery: %v", err)
	}
	for _, check := range []struct {
		path string
		want []byte
	}{
		{path(CACertFile), oldCA},
		{path(CAKeyFile), oldKey},
		{path(ServerCertFile), oldServer},
		{path(ServerKeyFile), oldServerKey},
		{path(ClientCertFile), oldClient},
		{path(ClientKeyFile), oldClientKey},
	} {
		got, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("read restored %s: %v", check.path, err)
		}
		if string(got) != string(check.want) {
			t.Fatalf("interrupted promotion did not restore %s", check.path)
		}
	}
	state, err = loadRollover()
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != "dual" {
		t.Fatalf("rollover phase after recovery = %q, want dual", state.Phase)
	}
}

func TestCARolloverRequiresCompleteRegisteredSecondaryInventory(t *testing.T) {
	setupRotationTest(t)
	if err := CARolloverStart(); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := IssueClientFromNewCA("secondary-test"); err != nil {
		t.Fatal(err)
	}
	info, err := ReadClientInfo("secondary-test")
	if err != nil {
		t.Fatal(err)
	}
	if confirmed, err := ConfirmCARolloverClient("secondary-test", info.Serial); err != nil || !confirmed {
		t.Fatalf("confirm Secondary identity = %v, err=%v", confirmed, err)
	}
	if err := CARolloverValidateClients([]string{"secondary-test", "offline-secondary"}); err == nil {
		t.Fatal("CA rollover accepted a registered Secondary without a confirmed new identity")
	}
	if err := CARolloverValidateClients([]string{"secondary-test"}); err != nil {
		t.Fatalf("confirmed registered Secondary rejected: %v", err)
	}
}
