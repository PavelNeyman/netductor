package deploy

import (
	"encoding/json"
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

func TestMaterialJSONForSCPKeepsKeysAndSlot0Defaults(t *testing.T) {
	raw := `{"peer_id":"peer-x","slot":0,"sp_priv_p":"PRIVP","sp_priv_s":"PRIVS","ps_priv_p":"PSPRIVP","ps_priv_s":"PSPRIVS"}`
	mat := peerMaterialSCP{PeerID: "peer-x", Slot: 0}
	if err := validatePeerMaterialForSCP(&mat, "peer-x"); err != nil {
		t.Fatal(err)
	}
	out, err := materialJSONForSCP(raw, mat)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	s := svcpaths.SlotTable()[0]
	if got["sp_priv_p"] != "PRIVP" || got["ps_priv_s"] != "PSPRIVS" {
		t.Fatalf("keys dropped: %v", got)
	}
	if int(got["port_sp"].(float64)) != s.PortSP || int(got["wss_ps"].(float64)) != s.WSS_PS {
		t.Fatalf("defaults not in shipped JSON: %v", got)
	}
	if got["iface_sp"] != s.IfaceSP || got["addr_p_sp"] != s.AddrP_SP {
		t.Fatalf("iface/addr not shipped: %v", got)
	}
}
