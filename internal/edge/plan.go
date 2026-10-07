package edge

import "fmt"

// DeployIntent is optional operator/template desire (CLI flags / overlay).
type DeployIntent struct {
	ConfigureNet bool // operator asked to push network UCI
	GuestEnable  bool
	VPNEnable    bool // template vpn.enabled
	WiFiSSID     string
	ExpandFS     bool // operator asked to expand root filesystem
	OverlayExt   bool // operator asked extroot/overlay on external disk
}

// PlanStep is one module decision for deploy-time SSH apply.
type PlanStep struct {
	Module string `json:"module"`
	Action string `json:"action"` // apply | skip
	Reason string `json:"reason,omitempty"`
}

// DeployPlan is the full ordered plan.
type DeployPlan struct {
	Preset string     `json:"preset"`
	Steps  []PlanStep `json:"steps"`
}

// BuildPlan intersects preset modules with facts and intent.
// Unmet soft requirements → skip; agent_install always apply if in preset.
func BuildPlan(preset string, facts DeviceFacts, intent DeployIntent) DeployPlan {
	if preset == "" {
		preset = SuggestPreset(facts)
	}
	want := map[string]bool{}
	for _, m := range PresetModules(preset) {
		want[m] = true
	}
	var steps []PlanStep
	for _, mod := range AllModuleOrder {
		if !want[mod] {
			continue
		}
		steps = append(steps, decideStep(mod, facts, intent))
	}
	return DeployPlan{Preset: preset, Steps: steps}
}

func decideStep(mod string, facts DeviceFacts, intent DeployIntent) PlanStep {
	switch mod {
	case ModAgentInstall:
		return PlanStep{Module: mod, Action: "apply", Reason: "critical"}
	case ModLANBaseline:
		if !intent.ConfigureNet {
			return PlanStep{Module: mod, Action: "skip", Reason: "configure-net not requested"}
		}
		return PlanStep{Module: mod, Action: "apply"}
	case ModWANBaseline:
		if !intent.ConfigureNet {
			return PlanStep{Module: mod, Action: "skip", Reason: "configure-net not requested"}
		}
		if !facts.WANCapable() {
			return PlanStep{Module: mod, Action: "skip", Reason: "no wan capability (single-NIC or no uci wan)"}
		}
		return PlanStep{Module: mod, Action: "apply"}
	case ModWiFiAP:
		if !intent.ConfigureNet && intent.WiFiSSID == "" {
			return PlanStep{Module: mod, Action: "skip", Reason: "no wifi intent"}
		}
		if !facts.HasRadios() {
			return PlanStep{Module: mod, Action: "skip", Reason: "no radios"}
		}
		return PlanStep{Module: mod, Action: "apply"}
	case ModGuest:
		if !intent.GuestEnable {
			return PlanStep{Module: mod, Action: "skip", Reason: "guest not requested"}
		}
		if !facts.HasRadios() {
			return PlanStep{Module: mod, Action: "skip", Reason: "no radios"}
		}
		return PlanStep{Module: mod, Action: "apply"}
	case ModVPNClient:
		if !intent.VPNEnable {
			return PlanStep{Module: mod, Action: "skip", Reason: "vpn not requested"}
		}
		if !facts.WANCapable() {
			return PlanStep{Module: mod, Action: "skip", Reason: "no uplink path at plan time"}
		}
		return PlanStep{Module: mod, Action: "apply", Reason: "may need default route after wan"}
	case ModFSExpand:
		if !facts.Storage.ExpandHint && !facts.Storage.HasMMC {
			return PlanStep{Module: mod, Action: "skip", Reason: "no expand candidate (no mmc / hint)"}
		}
		if !intent.ExpandFS {
			return PlanStep{Module: mod, Action: "skip", Reason: "optional — enable to resize root on storage"}
		}
		return PlanStep{Module: mod, Action: "apply", Reason: "operator requested fs expand"}
	case ModOverlay:
		if !facts.Storage.HasUSBDisk && !facts.Storage.HasMMC {
			return PlanStep{Module: mod, Action: "skip", Reason: "no external/mmc storage"}
		}
		if !intent.OverlayExt {
			return PlanStep{Module: mod, Action: "skip", Reason: "optional — enable for extroot/overlay"}
		}
		return PlanStep{Module: mod, Action: "apply"}
	case ModSSHHarden:
		return PlanStep{Module: mod, Action: "apply"}
	default:
		return PlanStep{Module: mod, Action: "skip", Reason: fmt.Sprintf("unknown module %s", mod)}
	}
}

// Applying returns module IDs with action apply.
func (p DeployPlan) Applying() []string {
	var out []string
	for _, s := range p.Steps {
		if s.Action == "apply" {
			out = append(out, s.Module)
		}
	}
	return out
}
