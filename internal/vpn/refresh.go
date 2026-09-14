package vpn

import (
	"fmt"
	"strings"
)

// RefreshLinks rewrites client artifacts for all users (or one name if non-empty).
func RefreshLinks(onlyName string) (int, error) {
	r, err := loadRegistry()
	if err != nil {
		return 0, err
	}
	n := 0
	var errs []string
	for _, u := range r.Users {
		if onlyName != "" && u.Name != onlyName {
			continue
		}
		if err := writeArtifacts(u.Name, u.UUID, u.Hy2Password); err != nil {
			errs = append(errs, u.Name+": "+err.Error())
			continue
		}
		n++
	}
	if len(errs) > 0 {
		return n, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return n, nil
}
