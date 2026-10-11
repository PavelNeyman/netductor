package devices

import "testing"

func TestListEmptyOK(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	_ = List()
}

func TestListByUserEmpty(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	_ = ListByUser("nobody")
}
