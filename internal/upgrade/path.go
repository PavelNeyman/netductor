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
// Rules:
//   - If target has no min_from or local already satisfies it → single hop [target].
//   - Otherwise insert the oldest intermediate tag whose version is >= min_from
//     and still < target, then re-evaluate (multi-hop if nested floors appear).
//   - history must include target (or a Meta with matching Version); missing
//     target meta is treated as no floor (single hop).
//
// Returns tags without a required "v" prefix normalized; caller may add "v".
func ResolveUpgradePath(local, target string, history []Meta) ([]PathStep, error) {
	local = normalizeVer(local)
	target = normalizeVer(target)
	if target == "" {
		return nil, fmt.Errorf("upgrade path: empty target")
	}
	if local != "" && !ndupdate.Newer(target, local) && local != target {
		// target older or equal: still allow single hop to target (rollback-ish)
		if local == target {
			return []PathStep{{Tag: target}}, nil
		}
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
		// Unknown target floor → skip-version single hop (phase A behavior when no upgrade.json).
		return []PathStep{{Tag: target}}, nil
	}

	var path []PathStep
	current := local
	guard := 0
	for {
		guard++
		if guard > 64 {
			return path, fmt.Errorf("upgrade path: too many hops (local=%s target=%s)", local, target)
		}
		minFrom := normalizeVer(tgtMeta.MinFrom)
		// When walking intermediates, use that tag's own min_from for the next hop check
		// only after we pick the hop; for reaching *target* we need current >= target.MinFrom.
		if err := CheckLocalAllowed(current, minFrom); err == nil {
			path = append(path, PathStep{
				Tag:     target,
				MinFrom: minFrom,
				Reason:  strings.Join(tgtMeta.Reasons, "; "),
			})
			return path, nil
		}
		// Need an intermediate: oldest tag T with current < T <= target and T >= minFrom
		// Prefer T == minFrom if present, else first T where T >= minFrom and T < target.
		var hop string
		for _, v := range vers {
			if v == current {
				continue
			}
			// only consider versions strictly above current and <= target
			if current != "" && !ndupdate.Newer(v, current) && v != current {
				continue
			}
			if current != "" && v == current {
				continue
			}
			if ndupdate.Newer(v, target) {
				continue // past target
			}
			if minFrom != "" && ndupdate.Newer(minFrom, v) {
				continue // still below required floor version
			}
			// v >= minFrom and v <= target
			if v == target {
				continue // would be final hop; if we are here final is blocked so skip
			}
			hop = v
			break
		}
		if hop == "" {
			return path, fmt.Errorf("upgrade path: local %s < min_from %s for %s; no intermediate tag in history that satisfies the floor", current, minFrom, target)
		}
		hm := byVer[hop]
		path = append(path, PathStep{
			Tag:     hop,
			MinFrom: normalizeVer(hm.MinFrom),
			Reason:  strings.Join(hm.Reasons, "; "),
		})
		current = hop
	}
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
