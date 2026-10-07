package deploy

import "testing"

func TestParseFactsProbe_RPiLike(t *testing.T) {
	raw := `
ARCH=aarch64
OS=openwrt
BOARD_NAME=raspberrypi,3-model-b
MODEL=Raspberry Pi 3 Model B
SERIAL=00000000abc
LAN_DEVICE=br-lan
LAN_IP=192.168.1.1
LAN_PROTO=static
WAN_PRESENT=0
MEM_TOTAL_KB=948000
MEM_AVAIL_KB=700000
HAS_MMC=1
HAS_USB_DISK=0
IFACE eth0 ethernet 1
IFACE wlan0 wireless 0
RADIO radio0 2g
SSID default_radio0 radio0 ap 0 OpenWrt
`
	f := parseFactsProbe(raw)
	if f.Arch != "arm64" {
		t.Fatalf("arch %s", f.Arch)
	}
	if f.Model == "" || f.Serial == "" {
		t.Fatalf("model/serial %q %q", f.Model, f.Serial)
	}
	if f.UCI.WANPresent {
		t.Fatal("wan should be false")
	}
	if f.EthernetCount() != 1 {
		t.Fatalf("eth %d", f.EthernetCount())
	}
	if !f.HasRadios() || f.MemAvailKB == 0 {
		t.Fatal("radios/mem")
	}
	if !f.Storage.HasMMC {
		t.Fatal("mmc")
	}
	if len(f.SSIDs) != 1 {
		t.Fatalf("ssids %d", len(f.SSIDs))
	}
	if f.WANCapable() {
		t.Fatal("should not be wan capable")
	}
}

func TestParseFactsProbe_CudyLike(t *testing.T) {
	raw := `
ARCH=mips
OS=openwrt
BOARD_NAME=cudy,tr1200
LAN_DEVICE=br-lan
LAN_IP=192.168.1.1
WAN_PRESENT=1
WAN_PROTO=dhcp
IFACE eth0 ethernet 1
IFACE eth1 ethernet 1
RADIO radio0 2g
RADIO radio1 5g
`
	f := parseFactsProbe(raw)
	if f.Arch != "mipsle" {
		t.Fatalf("arch %s", f.Arch)
	}
	if !f.WANCapable() {
		t.Fatal("wan capable")
	}
	if len(f.Bands()) != 2 {
		t.Fatalf("bands %v", f.Bands())
	}
}
