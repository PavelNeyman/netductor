package edge

import "testing"

func TestDeviceIDFromAuthEmpty(t *testing.T) {
	if DeviceIDFromAuth("") != "" {
		t.Fatal("expected empty")
	}
	if DeviceIDFromAuth("Bearer short") != "" {
		t.Fatal("expected empty")
	}
}
