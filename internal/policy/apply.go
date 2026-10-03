package policy

import "fmt"

// ApplyStatus is returned by ApplyRoutes (P1 will implement real sing-box rewrite).
type ApplyStatus struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Pending bool   `json:"pending"` // true until sing-box enforcement lands
}

// ApplyRoutes rebuilds sing-box route rules from all policies.
// Phase P1: real generation. Until then this is a no-op success with pending=true
// so API/CLI can call it safely after policy writes.
func ApplyRoutes() (ApplyStatus, error) {
	_, err := EnsureCatalog()
	if err != nil {
		return ApplyStatus{OK: false, Message: err.Error()}, err
	}
	return ApplyStatus{
		OK:      true,
		Pending: true,
		Message: "policy stored; sing-box route enforcement not yet applied (phase P1)",
	}, nil
}

// ApplyRoutesOrError wraps ApplyRoutes for callers that only need error.
func ApplyRoutesOrError() error {
	st, err := ApplyRoutes()
	if err != nil {
		return err
	}
	if !st.OK {
		return fmt.Errorf("%s", st.Message)
	}
	return nil
}
