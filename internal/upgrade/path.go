package upgrade

import (
	"fmt"
	"sort"
	"strings"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

// PathStep is one hop in an upgrade chain (CF-15 phase B).
type PathStep struct {
	Tag     string `json:"tag"`
	MinFrom string `json:"min_from,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// ResolveUpgradePath builds an ordered tag list from local toward target using
// published Meta history (newest or oldest order accepted; sorted ascending).
//
// Each hop H is only chosen when current already satisfies H's own min_from
// (intermediate floors are not skipped — BUG-1).
//
// Missing target meta → single hop (skip-version OK).
func ResolveUpgradePath(local, target string, history []Meta) ([]PathStep, error) {
	local = normalizeVer(local)
	target = normalizeVer(target)
	if target == "" {
		return nil, fmt.Errorf("upgrade path: empty target")
	}
	if local != "" && local == target {
		return []PathStep{{Tag: target}}, nil
	}

	byVer := map[string]Meta{}
	var vers []string
	for _, m := range history {
		v := normalizeVer(m.Version)
		if v == "" {
			continue
		}
		byVer[v] = m
		vers = append(vers, v)
	}
	sort.Slice(vers, func(i, j int) bool {
		return ndupdate.Newer(vers[j], vers[i]) // ascending: older first
	})

	tgtMeta, hasTgt := byVer[target]
	if !hasTgt {
		return []PathStep{{Tag: target}}, nil
	}

	stepOf := func(v string) PathStep {
		m := byVer[v]
		return PathStep{
			Tag:     v,
			MinFrom: normalizeVer(m.MinFrom),
			Reason:  strings.Join(m.Reasons, "; "),
		}
	}

	var path []PathStep
	current := local
	for guard := 0; guard < 64; guard++ {
		tgtMin := normalizeVer(tgtMeta.MinFrom)
		if err := CheckLocalAllowed(current, tgtMin); err == nil {
			path = append(path, stepOf(target))
			return path, nil
		}
		// Oldest intermediate installable from current (respect hop min_from).
		var hop string
		for _, v := range vers {
			if v == current || v == target {
				continue
			}
			if current != "" && !ndupdate.Newer(v, current) {
				continue
			}
			if ndupdate.Newer(v, target) {
				continue
			}
			hm := byVer[v]
			if err := CheckLocalAllowed(current, normalizeVer(hm.MinFrom)); err != nil {
				continue
			}
			hop = v
			break
		}
		if hop == "" {
			return path, fmt.Errorf("upgrade path: local %s < min_from %s for %s; no installable intermediate in history", current, tgtMin, target)
		}
		path = append(path, stepOf(hop))
		current = hop
	}
	return path, fmt.Errorf("upgrade path: too many hops (local=%s target=%s)", local, target)
}

// FormatPathCLI is a one-line operator hint.
func FormatPathCLI(steps []PathStep) string {
	if len(steps) == 0 {
		return "(empty path)"
	}
	parts := make([]string, 0, len(steps))
	for _, s := range steps {
		parts = append(parts, "v"+normalizeVer(s.Tag))
	}
	return strings.Join(parts, " → ")
}
