package servicenet

import "testing"

func TestEnabledAndCollect(t *testing.T) {
	_ = Enabled()
	_ = Collect()
}
