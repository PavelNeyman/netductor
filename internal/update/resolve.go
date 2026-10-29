package update

import (
	"fmt"
	"runtime"
	"strings"
)

// TrustNoteSHA256SUMS documents the Phase 2 trust boundary.
// Checksums published next to binaries on the same GitHub Release do not provide
// independent publisher authenticity (an attacker who replaces assets can replace SUMS).
// Independent signing remains a later hardening item.
const TrustNoteSHA256SUMS = "SHA256SUMS is integrity-only within the release channel, not independent provenance"

// ResolvedAsset is the server-side result of selecting a release binary.
type ResolvedAsset struct {
	Component ComponentID
	Tag       string
	Name      string // basename, e.g. netductor-linux-amd64
	SHA256    string // lowercase hex from SHA256SUMS
	Platform  Platform
}

// PlatformFromRuntime returns GOOS/GOARCH for the current process.
func PlatformFromRuntime() Platform {
	return Platform{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
}

// ResolveReleaseAsset picks the canonical asset for component+platform at tag
// and requires a SHA256SUMS entry. Clients must not pass free-form asset names.
func ResolveReleaseAsset(comp ComponentID, p Platform, tag string) (ResolvedAsset, error) {
	var zero ResolvedAsset
	if !ValidComponent(comp) {
		return zero, fmt.Errorf("update: unsupported component %q", comp)
	}
	tag, err := ValidReleaseTag(tag)
	if err != nil {
		return zero, err
	}
	name := AssetName(comp, p)
	if name == "" {
		return zero, fmt.Errorf("update: no asset for component %s on %s/%s", comp, p.GOOS, p.GOARCH)
	}
	sums, err := FetchSHA256SUMS(tag)
	if err != nil {
		return zero, fmt.Errorf("update: %s: %w", TrustNoteSHA256SUMS, err)
	}
	sum, ok := sums[name]
	if !ok {
		// try case-insensitive key match
		for k, v := range sums {
			if strings.EqualFold(k, name) {
				sum, ok = v, true
				name = k
				break
			}
		}
	}
	if !ok || sum == "" {
		return zero, fmt.Errorf("update: %s missing from SHA256SUMS for %s", name, tag)
	}
	return ResolvedAsset{
		Component: comp,
		Tag:       tag,
		Name:      name,
		SHA256:    strings.ToLower(sum),
		Platform:  p,
	}, nil
}

// ResolveReleaseAssetLocal is like ResolveReleaseAsset but uses an already-parsed SUMS map
// (for tests and offline mirrors).
func ResolveReleaseAssetLocal(comp ComponentID, p Platform, tag string, sums map[string]string) (ResolvedAsset, error) {
	var zero ResolvedAsset
	if !ValidComponent(comp) {
		return zero, fmt.Errorf("update: unsupported component %q", comp)
	}
	name := AssetName(comp, p)
	if name == "" {
		return zero, fmt.Errorf("update: no asset for component %s on %s/%s", comp, p.GOOS, p.GOARCH)
	}
	sum, ok := sums[name]
	if !ok {
		return zero, fmt.Errorf("update: %s missing from SHA256SUMS", name)
	}
	return ResolvedAsset{
		Component: comp,
		Tag:       tag,
		Name:      name,
		SHA256:    strings.ToLower(sum),
		Platform:  p,
	}, nil
}
