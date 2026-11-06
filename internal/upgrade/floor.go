// Package upgrade holds binary upgrade floors (CF-15): declarations in code,
// aggregate min_from for release metadata, and checks before stack apply.
package upgrade

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

// Floor is one breaking change that requires a minimum installed binary version
// before jumping to a release that includes this change.
type Floor struct {
	ID        string `json:"id"`
	MinBinary string `json:"min_binary"` // e.g. "0.9.700" (no required v prefix)
	Reason    string `json:"reason"`
}

// Meta is release-side upgrade metadata (upgrade.json).
type Meta struct {
	Version string   `json:"version"`
	MinFrom string   `json:"min_from,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
	Floors  []Floor  `json:"floors,omitempty"`
}

var (
	mu     sync.Mutex
	floors []Floor
)

// DeclareBinaryFloor registers a floor. Call from init() next to the breaking change.
// Duplicate IDs: last registration wins (tests may re-declare). Empty minBinary is ignored.
func DeclareBinaryFloor(id, minBinary, reason string) {
	id = strings.TrimSpace(id)
	minBinary = normalizeVer(minBinary)
	reason = strings.TrimSpace(reason)
	if id == "" || minBinary == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	for i := range floors {
		if floors[i].ID == id {
			floors[i] = Floor{ID: id, MinBinary: minBinary, Reason: reason}
			return
		}
	}
	floors = append(floors, Floor{ID: id, MinBinary: minBinary, Reason: reason})
}

// ResetFloorsForTest clears the registry (tests only).
func ResetFloorsForTest() {
	mu.Lock()
	floors = nil
	mu.Unlock()
}

// Floors returns a copy of registered floors sorted by ID.
func Floors() []Floor {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Floor, len(floors))
	copy(out, floors)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// AggregateMinFrom returns the highest MinBinary among floors (empty if none).
func AggregateMinFrom() string {
	best := ""
	for _, f := range Floors() {
		if best == "" || ndupdate.Newer(f.MinBinary, best) {
			best = f.MinBinary
		}
	}
	return best
}

// BuildMeta constructs upgrade.json content for a release version from the in-code registry.
func BuildMeta(version string) Meta {
	version = normalizeVer(version)
	fs := Floors()
	m := Meta{
		Version: version,
		MinFrom: AggregateMinFrom(),
		Floors:  fs,
	}
	for _, f := range fs {
		if f.Reason != "" {
			m.Reasons = append(m.Reasons, f.ID+": "+f.Reason)
		}
	}
	return m
}

// MarshalMetaJSON returns indented JSON for upgrade.json artifacts.
func MarshalMetaJSON(version string) ([]byte, error) {
	return json.MarshalIndent(BuildMeta(version), "", "  ")
}

// CheckLocalAllowed returns nil if local may jump to a release that requires minFrom.
// Empty minFrom → always allowed. local empty → refuse when minFrom is set.
func CheckLocalAllowed(local, minFrom string) error {
	minFrom = normalizeVer(minFrom)
	if minFrom == "" {
		return nil
	}
	local = normalizeVer(local)
	if local == "" {
		return fmt.Errorf("upgrade floor: local VERSION unknown; need at least %s before this release (CF-15)", minFrom)
	}
	// Allowed when local >= minFrom (not strictly older).
	if ndupdate.Newer(minFrom, local) {
		return fmt.Errorf("upgrade floor: local %s < min_from %s — install intermediate release(s) first, or upgrade stepwise (CF-15)", local, minFrom)
	}
	return nil
}

// CheckLocalAgainstRegistry uses AggregateMinFrom from this binary's registry
// (useful when applying a build that embeds these floors). Prefer release upgrade.json when present.
func CheckLocalAgainstRegistry(local string) error {
	return CheckLocalAllowed(local, AggregateMinFrom())
}

func normalizeVer(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	return v
}
