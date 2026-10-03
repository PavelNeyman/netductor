package policy

import "fmt"

// ApplyStatus is returned after attempting to enforce policies.
type ApplyStatus struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Pending bool   `json:"pending"`
}

// ApplyHook is set by the node binary to vpn.ApplyAccessPolicies (avoids import cycles).
var ApplyHook func() (ApplyStatus, error)

// ApplyRoutes rebuilds routes. When ApplyHook is nil (tests / library), returns pending OK.
func ApplyRoutes() (ApplyStatus, error) {
	_, err := EnsureCatalog()
	if err != nil {
		return ApplyStatus{OK: false, Message: err.Error()}, err
	}
	if ApplyHook != nil {
		return ApplyHook()
	}
	return ApplyStatus{
		OK:      true,
		Pending: true,
		Message: "policy stored; set policy.ApplyHook to enforce sing-box routes",
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
