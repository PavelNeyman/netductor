package edge

import "testing"

func TestResolvePlan_InlineFacts(t *testing.T) {
	f := DeviceFacts{
		Arch:   "arm64",
		Board:  "raspberrypi,3-model-b",
		Model:  "Raspberry Pi 3 Model B",
		Ifaces: []NetIface{{Name: "eth0", Type: "ethernet"}},
		Radios: []RadioFact{{Name: "radio0", Band: "2g"}},
		UCI:    UCINetFacts{LANIP: "192.168.1.1", WANPresent: false},
	}
	resp, err := ResolvePlan(PlanRequest{
		Preset: PresetSBCLab,
		Facts:  &f,
		Intent: DeployIntent{ConfigureNet: true, GuestEnable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Card.EthCount != 1 || resp.Card.WANCapable {
		t.Fatalf("card %+v", resp.Card)
	}
	for _, s := range resp.Plan.Steps {
		if s.Module == ModWANBaseline && s.Action == "apply" {
			t.Fatal("wan should not apply")
		}
	}
}

func TestCardFromFacts_Bands(t *testing.T) {
	c := CardFromFacts(DeviceFacts{
		Radios: []RadioFact{{Band: "2g"}, {Band: "5g"}},
		SSIDs:  []WiFiSSIDFact{{SSID: "Home", Disabled: false}},
	})
	if len(c.Bands) != 2 || len(c.SSIDNow) != 1 {
		t.Fatalf("%+v", c)
	}
}

func TestResolvePlan_CudyLikeDualNIC(t *testing.T) {
	f := DeviceFacts{
		Arch:  "aarch64",
		Board: "cudy,tr1200",
		Model: "Cudy TR1200",
		Ifaces: []NetIface{
			{Name: "eth0", Type: "ethernet"},
			{Name: "eth1", Type: "ethernet"},
		},
		Radios: []RadioFact{{Name: "radio0", Band: "2g"}, {Name: "radio1", Band: "5g"}},
		UCI:    UCINetFacts{LANIP: "192.168.1.1", WANPresent: true},
	}
	resp, err := ResolvePlan(PlanRequest{
		Preset: PresetTravelRouter,
		Facts:  &f,
		Intent: DeployIntent{ConfigureNet: true, GuestEnable: true, VPNEnable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Card.WANCapable || resp.Card.EthCount < 2 {
		t.Fatalf("card %+v", resp.Card)
	}
	wantApply := map[string]bool{ModWANBaseline: false, ModWiFiAP: false, ModGuest: false}
	for _, s := range resp.Plan.Steps {
		if s.Module == ModWANBaseline || s.Module == ModWiFiAP || s.Module == ModGuest {
			wantApply[s.Module] = s.Action == "apply"
		}
	}
	for mod, ok := range wantApply {
		if !ok {
			t.Fatalf("expected apply for %s: %+v", mod, resp.Plan.Steps)
		}
	}
}
