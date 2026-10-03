package vpn

import "github.com/PavelNeyman/netductor/internal/policy"

// ApplyAccessPolicies rebuilds sing-box with current user policies (P1 enforcement).
func ApplyAccessPolicies() (policy.ApplyStatus, error) {
	_, err := policy.EnsureCatalog()
	if err != nil {
		return policy.ApplyStatus{OK: false, Message: err.Error()}, err
	}
	if err := ApplyConfig(); err != nil {
		// secrets missing on fresh box is ok for unit tests / dry paths
		return policy.ApplyStatus{
			OK:      false,
			Pending: false,
			Message: err.Error(),
		}, err
	}
	return policy.ApplyStatus{
		OK:      true,
		Pending: false,
		Message: "sing-box config rewritten with service access policies",
	}, nil
}
