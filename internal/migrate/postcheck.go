package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// PostCheck verifies minimal host state after migrations (role-aware).
// Does not start services; only filesystem/config presence.
func PostCheck() error {
	var errs []string
	for _, d := range []string{paths.StateDir(), paths.EtcDir()} {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			errs = append(errs, "missing dir "+d)
		}
	}
	roleFile := filepath.Join(paths.EtcDir(), "role")
	if b, err := os.ReadFile(roleFile); err != nil {
		errs = append(errs, "role file missing: "+roleFile)
	} else if len(strings.TrimSpace(string(b))) == 0 {
		errs = append(errs, "empty role file")
	}
	_ = firewall.DetectRole() // ensure package linked; role file is source of truth after migration
	jp := journalPath()
	if _, err := os.Stat(jp); err != nil {
		errs = append(errs, "migrate journal missing after run")
	}
	if len(errs) == 0 {
		return nil
	}
	msg := "post-check:"
	for _, e := range errs {
		msg += " " + e + ";"
	}
	return fmt.Errorf("%s", msg)
}
