package sites

import "testing"

func TestUpsertList(t *testing.T) {
	s, err := Upsert(Site{ID: "test-home", Name: "Home", RPiID: "rpi1"})
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "test-home" {
		t.Fatal(s)
	}
	list, err := List()
	if err != nil || len(list) < 1 {
		t.Fatal(list, err)
	}
}
