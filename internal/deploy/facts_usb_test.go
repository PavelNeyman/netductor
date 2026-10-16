package deploy

import "testing"

func TestParseFactsUSBDisk(t *testing.T) {
	out := `
ARCH=mipsle
OS=openwrt
HAS_USB_DISK=1
USB_DISK name=sda size_kb=7818184 partitions=1
BLOCK /dev/sda1: UUID="abc-123" TYPE="ext4" MOUNT="/mnt/sda1"
EXTROOT_ACTIVE=0
EXPAND_HINT=0
`
	f := parseFactsProbe(out)
	if !f.Storage.HasUSBDisk {
		t.Fatal("HasUSBDisk")
	}
	if len(f.Storage.USBDisks) != 1 {
		t.Fatalf("disks %d", len(f.Storage.USBDisks))
	}
	d := f.Storage.USBDisks[0]
	if d.Name != "sda" || d.SizeKB != 7818184 {
		t.Fatalf("%+v", d)
	}
	if d.UUID != "abc-123" || d.FS != "ext4" {
		t.Fatalf("block enrich %+v", d)
	}
}
