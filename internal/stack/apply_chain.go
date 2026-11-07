package stack

import (
	"fmt"
	"os"
	"strings"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/upgrade"
)

// ResolveApplyPath builds CF-15 hop list for tag using local release store + target meta.
func ResolveApplyPath(tag string) ([]upgrade.PathStep, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		var err error
		tag, err = ndupdate.LatestReleaseTag()
		if err != nil {
			return nil, err
		}
	}
	tag = strings.TrimPrefix(tag, "v")
	local := upgrade.LocalVersion()
	var hist []upgrade.Meta
	if tags, err := ndupdate.ListLocalTags(); err == nil {
		for _, tg := range tags {
			if m, ok, _ := upgrade.FetchMeta(tg); ok {
				hist = append(hist, m)
			}
		}
	}
	if m, ok, err := upgrade.FetchMeta(tag); err != nil {
		return nil, err
	} else if ok {
		hist = append(hist, m)
	}
	return upgrade.ResolveUpgradePath(local, tag, hist)
}

// FormatApplyPath is a CLI one-liner for hop chain.
func FormatApplyPath(steps []upgrade.PathStep) string {
	return upgrade.FormatPathCLI(steps)
}

// ApplyChain resolves the upgrade path and applies each hop in order (CF-15 phase C, control node).
// Single-hop paths are identical to Apply. On intermediate failure, stops; no auto-skip.
func ApplyChain(tag string) error {
	steps, err := ResolveApplyPath(tag)
	if err != nil {
		return err
	}
	if len(steps) == 0 {
		return fmt.Errorf("stack apply --chain: empty path")
	}
	fmt.Fprintf(os.Stderr, "stack apply --chain: %s\n", FormatApplyPath(steps))
	for i, s := range steps {
		hop := "v" + strings.TrimPrefix(s.Tag, "v")
		fmt.Fprintf(os.Stderr, "stack apply --chain: hop %d/%d %s\n", i+1, len(steps), hop)
		if err := Apply(hop); err != nil {
			return fmt.Errorf("stack apply --chain hop %s: %w", hop, err)
		}
	}
	return nil
}
