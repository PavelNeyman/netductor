package vpn

import (
	"github.com/PavelNeyman/netductor/internal/nvr"
	"github.com/PavelNeyman/netductor/internal/policy"
)

func init() {
	policy.ServiceEnabled = func(id string) bool {
		if id == "nvr" {
			return nvr.LoadConfig().RecordEnabled
		}
		return true
	}
}

// ApplyAccessPolicies rebuilds sing-box with current user policies (P1 enforcement).
func ApplyAccessPolicies() (policy.ApplyStatus, error) {
	_, err := policy.EnsureCatalog()
	if err != nil {
		return policy.ApplyStatus{OK: false, Message: err.Error()}, err
	}
	if err := ApplyConfig(); err != nil {
		return policy.ApplyStatus{OK: false, Pending: false, Message: err.Error()}, err
	}
	return policy.ApplyStatus{OK: true, Pending: false, Message: "sing-box config rewritten with service access policies"}, nil
}
