package install

import (
	"fmt"
	"path/filepath"

	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/statefile"
	"github.com/PavelNeyman/netductor/internal/tlsle"
)

// RepairStateOnUpgrade runs idempotent fixes after stack apply / host-baseline:
// LE renew hooks, state JSON readability + mode 0600, parent dirs.
// Does not rewrite application data content (safe on every upgrade).
func RepairStateOnUpgrade() []string {
	var notes []string
	if err := tlsle.EnsureRenewHooks(); err != nil {
		notes = append(notes, "le hooks: "+err.Error())
	} else {
		notes = append(notes, "le renew hooks ok")
	}

	files := []string{
		filepath.Join(paths.EdgeDir(), "devices.json"),
		filepath.Join(paths.EdgeDir(), "recovery_codes.json"),
		filepath.Join(paths.StateDir(), "vpn_sub_tokens.json"),
		filepath.Join(paths.StateDir(), "sites", "sites.json"),
		filepath.Join(paths.StateDir(), "reboot-challenge.json"),
	}
	for _, f := range files {
		_ = statefile.EnsureParentDir(f)
		r := statefile.CheckJSON(f)
		if !r.OK {
			notes = append(notes, "state bad: "+r.Path+": "+r.Detail)
			continue
		}
		if r.Detail != "absent (ok)" {
			if err := statefile.RepairFileMode(f); err != nil {
				notes = append(notes, "chmod: "+f+": "+err.Error())
			}
		}
	}
	return notes
}

// PrintRepair for CLI.
func PrintRepair() {
	for _, n := range RepairStateOnUpgrade() {
		fmt.Println(n)
	}
}
