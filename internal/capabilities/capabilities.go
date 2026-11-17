// Package capabilities lists optional node workloads (fleet plan Phase 1).
package capabilities

// Well-known capability ids stored in nodes.Labels as "cap.<id>" = "1".
const (
	VPNIngress = "vpn_ingress"
	VPNEgress  = "vpn_egress"
	DNS        = "dns"
	Media      = "media"
	Build      = "build"
	NVR        = "nvr"
)

// LabelKey returns the nodes.Labels key for a capability.
func LabelKey(id string) string { return "cap." + id }

// Has reports whether labels enable the capability.
func Has(labels map[string]string, id string) bool {
	if labels == nil {
		return false
	}
	v := labels[LabelKey(id)]
	return v == "1" || v == "true" || v == "yes"
}

// Set enables or clears a capability on a labels map (mutates/returns map).
func Set(labels map[string]string, id string, on bool) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}
	k := LabelKey(id)
	if on {
		labels[k] = "1"
	} else {
		delete(labels, k)
	}
	return labels
}

// DefaultsForRole returns typical capabilities for a role (documentation defaults, not auto-applied).
func DefaultsForRole(role string) []string {
	switch role {
	case "secondary", "egress":
		return []string{VPNIngress, VPNEgress}
	case "primary", "control":
		return []string{DNS}
	case "build":
		return []string{Build}
	default:
		return nil
	}
}
