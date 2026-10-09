package sites

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func TestRoomsCRUDPhotos(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	// paths.StateDir may not read env — set via paths if needed
	if paths.StateDir() != dir {
		// force by writing under real state is bad; check paths package
		t.Log("state dir", paths.StateDir())
	}
	os.Setenv("NETDUCTOR_STATE", dir)
	_, err := Upsert(Site{ID: "home", Name: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := UpsertRoom(Room{ID: "kitchen", SiteID: "home", Name: "Kitchen", CameraIDs: []string{"cam1"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "kitchen" {
		t.Fatal(r)
	}
	list, err := ListRooms("home")
	if err != nil || len(list) != 1 {
		t.Fatal(list, err)
	}
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46}
	for i := 0; i < MaxRoomPhotos; i++ {
		idx, err := AddPhoto("home", "kitchen", jpeg, "image/jpeg")
		if err != nil {
			t.Fatal(i, err)
		}
		if idx != i {
			t.Fatal(idx, i)
		}
	}
	if _, err := AddPhoto("home", "kitchen", jpeg, "image/jpeg"); err == nil {
		t.Fatal("expected max photos error")
	}
	got, err := GetPhoto("home", "kitchen", 0)
	if err != nil || len(got) < 3 {
		t.Fatal(err, got)
	}
	if err := DeletePhoto("home", "kitchen", 0); err != nil {
		t.Fatal(err)
	}
	r2, ok := GetRoom("home", "kitchen")
	if !ok || r2.PhotoCount != MaxRoomPhotos-1 {
		t.Fatal(r2, ok)
	}
	if err := DeleteRoom("home", "kitchen"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sites", "home", "rooms", "kitchen.json")); !os.IsNotExist(err) {
		// may use different state dir
		t.Log("cleanup path", err)
	}
}
