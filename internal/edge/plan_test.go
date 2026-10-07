package edge

import "testing"

func TestBuildPlan_RPiFactory(t *testing.T) {
	facts := DeviceFacts{
		Arch:   "arm64",
		Board:  "raspberrypi,3-model-b",
		OS:     "openwrt",
		Ifaces: []NetIface{{Name: "eth0", Type: "ethernet", Up: true}},
		Radios: []RadioFact{{Name: "radio0", Band: "2g"}},
		UCI:    UCINetFacts{LANDevice: "br-lan", LANIP: "192.168.1.1", WANPresent: false},
	}
	intent := DeployIntent{ConfigureNet: true, GuestEnable: true, VPNEnable: true, WiFiSSID: "Lab"}
	plan := BuildPlan(PresetSBCLab, facts, intent)
	apply := map[string]bool{}
	for _, s := range plan.Steps {
		if s.Action == "apply" {
			apply[s.Module] = true
		}
	}
	if !apply[ModAgentInstall] {
		t.Fatal("agent_install must apply")
	}
	if apply[ModWANBaseline] {
		t.Fatal("wan must skip on single-NIC sbc-lab")
	}
	if !apply[ModLANBaseline] {
		t.Fatal("lan should apply when configure-net")
	}
	if !apply[ModWiFiAP] {
		t.Fatal("wifi should apply when radios present")
	}
}

func TestBuildPlan_CudyLike(t *testing.T) {
	facts := DeviceFacts{
		Arch: "mipsle",
		Board: "cudy,tr1200",
		OS:   "openwrt",
		Ifaces: []NetIface{
			{Name: "eth0", Type: "ethernet", Up: true},
			{Name: "eth1", Type: "ethernet", Up: true},
		},
		Radios: []RadioFact{{Name: "radio0", Band: "2g"}, {Name: "radio1", Band: "5g"}},
		UCI:    UCINetFacts{LANDevice: "br-lan", LANIP: "192.168.1.1", WANPresent: true, WANProto: "dhcp"},
	}
	intent := DeployIntent{ConfigureNet: true, GuestEnable: true, VPNEnable: true, WiFiSSID: "Home"}
	plan := BuildPlan(PresetTravelRouter, facts, intent)
	apply := map[string]bool{}
	for _, s := range plan.Steps {
		if s.Action == "apply" {
			apply[s.Module] = true
		}
	}
	for _, m := range []string{ModAgentInstall, ModLANBaseline, ModWANBaseline, ModWiFiAP, ModGuest, ModVPNClient, ModSSHHarden} {
		if !apply[m] {
			t.Fatalf("expected apply %s", m)
		}
	}
}

func TestSuggestPreset(t *testing.T) {
	rpi := DeviceFacts{Ifaces: []NetIface{{Type: "ethernet"}}, UCI: UCINetFacts{WANPresent: false}}
	if g := SuggestPreset(rpi); g != PresetSBCLab {
		t.Fatalf("rpi suggest %s", g)
	}
	cudy := DeviceFacts{UCI: UCINetFacts{WANPresent: true}}
	if g := SuggestPreset(cudy); g != PresetTravelRouter {
		t.Fatalf("cudy suggest %s", g)
	}
}

func TestApplySelection_AdvancedForcesWAN(t *testing.T) {
	facts := DeviceFacts{
		Ifaces: []NetIface{{Name: "eth0", Type: "ethernet"}},
		UCI:    UCINetFacts{WANPresent: false},
	}
	plan := BuildPlan(PresetSBCLab, facts, DeployIntent{ConfigureNet: true})
	sel := ModuleSelection{Advanced: true, Enabled: map[string]bool{ModWANBaseline: true}}
	plan2 := ApplySelection(plan, sel, facts)
	found := false
	for _, s := range plan2.Steps {
		if s.Module == ModWANBaseline {
			found = true
			if s.Action != "apply" {
				t.Fatalf("advanced should force wan apply, got %s (%s)", s.Action, s.Reason)
			}
		}
	}
	if !found {
		// sbc-lab preset may omit wan entirely — force by rebuilding travel + selection
		plan = BuildPlan(PresetTravelRouter, facts, DeployIntent{ConfigureNet: true})
		plan2 = ApplySelection(plan, sel, facts)
		for _, s := range plan2.Steps {
			if s.Module == ModWANBaseline && s.Action == "apply" {
				return
			}
		}
		t.Fatal("wan not forced")
	}
}
