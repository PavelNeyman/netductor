package deploy

import "testing"

func TestParseFactsProbe_RPiLike(t *testing.T) {
	raw := `
ARCH=aarch64
OS=openwrt
BOARD_NAME=raspberrypi,3-model-b
LAN_DEVICE=br-lan
LAN_IP=192.168.1.1
WAN_PRESENT=0
IFACE eth0 ethernet 1
IFACE wlan0 wireless 0
RADIO radio0 2g
`
	f := parseFactsProbe(raw)
	if f.Arch != "arm64" {
		t.Fatalf("arch %s", f.Arch)
	}
	if f.UCI.WANPresent {
		t.Fatal("wan should be false")
	}
	if f.EthernetCount() != 1 {
		t.Fatalf("eth %d", f.EthernetCount())
	}
	if !f.HasRadios() {
		t.Fatal("radios")
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
	if len(f.Radios) != 2 {
		t.Fatalf("radios %d", len(f.Radios))
	}
}
