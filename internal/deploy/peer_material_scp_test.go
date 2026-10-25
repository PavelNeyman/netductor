package deploy

import (
	"strings"
	"testing"

	"github.com/PavelNeyman/netductor/internal/svcpaths"
)

func sampleMat(slot int) peerMaterialSCP {
	s := svcpaths.SlotTable()[slot]
	return peerMaterialSCP{
		PeerID:  "peer-203-0-113-2",
		Slot:    slot,
		AddrPSP: s.AddrP_SP,
		AddrSSP: s.AddrS_SP,
		AddrPPS: s.AddrP_PS,
		AddrSPS: s.AddrS_PS,
		IfaceSP: s.IfaceSP,
		IfacePS: s.IfacePS,
		PortSP:  s.PortSP,
		PortPS:  s.PortPS,
		WSSSP:   s.WSS_SP,
		WSSPS:   s.WSS_PS,
	}
}

func TestValidatePeerMaterialForSCPOK(t *testing.T) {
	m := sampleMat(1)
	if err := validatePeerMaterialForSCP(&m, m.PeerID); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePeerMaterialForSCPWrongWSS(t *testing.T) {
	m := sampleMat(1)
	m.WSSSP = 8444 // slot0
	err := validatePeerMaterialForSCP(&m, m.PeerID)
	if err == nil || !strings.Contains(err.Error(), "ports/wss") {
		t.Fatalf("got %v", err)
	}
}

func TestValidatePeerMaterialForSCPZeroPortSlot1(t *testing.T) {
	m := sampleMat(1)
	m.PortSP = 0
	err := validatePeerMaterialForSCP(&m, m.PeerID)
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("got %v", err)
	}
}

func TestValidatePeerMaterialForSCPSlot0Defaults(t *testing.T) {
	m := peerMaterialSCP{PeerID: "peer-x", Slot: 0}
	if err := validatePeerMaterialForSCP(&m, "peer-x"); err != nil {
		t.Fatal(err)
	}
	s := svcpaths.SlotTable()[0]
	if m.PortSP != s.PortSP || m.WSSSP != s.WSS_SP || m.WSSPS != s.WSS_PS {
		t.Fatalf("defaults not applied: %+v", m)
	}
}
