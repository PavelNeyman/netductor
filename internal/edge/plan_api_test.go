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
