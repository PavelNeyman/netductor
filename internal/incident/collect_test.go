package incident

import "testing"

func TestCollectNoPanic(t *testing.T) {
	_, _, _ = Collect(0)
	_, _, _ = Collect(1)
}
