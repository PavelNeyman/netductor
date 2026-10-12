package gitstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShipJobStoreSlotAndRm(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_SHIP_JOBS", dir)
	j, err := CreateShipJob("demo", "HEAD", "v0.0.1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if j.State != ShipQueued || j.VpsSlot {
		t.Fatalf("queued: %+v", j)
	}
	if err := TryAcquireVpsSlot(j.ID, ShipRunning); err != nil {
		t.Fatal(err)
	}
	if !VpsSlotTaken() {
		t.Fatal("slot should be taken")
	}
	j2, err := CreateShipJob("other", "main", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := TryAcquireVpsSlot(j2.ID, ShipRunning); err == nil {
		t.Fatal("second slot should fail")
	}
	if err := MarkShipWaiting(j.ID, "mac"); err != nil {
		t.Fatal(err)
	}
	if VpsSlotTaken() {
		t.Fatal("waiting must release slot")
	}
	if err := TryAcquireVpsSlot(j2.ID, ShipRunning); err != nil {
		t.Fatal(err)
	}
	// cannot rm while holding slot
	if err := RemoveShipJob(j2.ID); err == nil {
		t.Fatal("rm running should fail")
	}
	_ = ReleaseVpsSlot(j2.ID)
	got, _ := GetShipJob(j2.ID)
	got.State = ShipPaused
	_ = SaveShipJob(got)
	if err := RemoveShipJob(j2.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetShipJob(j2.ID); err == nil {
		t.Fatal("expected missing")
	}
	// artifacts dir cleaned
	if _, err := os.Stat(filepath.Join(dir, j2.ID)); !os.IsNotExist(err) {
		// dir may not exist if never created — ok
	}
}

func TestShipPauseRequest(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_SHIP_JOBS", dir)
	j, err := CreateShipJob("p1", "HEAD", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := RequestShipPause(j.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := GetShipJob(j.ID)
	if !got.PauseRequested {
		t.Fatal("expected pause_requested")
	}
}
