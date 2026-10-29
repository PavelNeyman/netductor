package update

import (
	"fmt"
	"sort"
	"strings"
)

// RequiredLinuxNodeAssets is the minimum set published for VPS node/tg/agent.
// Align with scripts/verify-release-binaries.sh.
func RequiredLinuxNodeAssets() []string {
	arches := []string{"amd64", "arm64", "arm", "mipsle", "riscv64"}
	var out []string
	for _, a := range arches {
		out = append(out,
			"netductor-linux-"+a,
			"netductor-tg-linux-"+a,
			"netductor-agent-linux-"+a,
		)
	}
	out = append(out, "SHA256SUMS")
	return out
}

// ValidateAssetNameMatchesContract ensures name equals AssetName for the component/platform.
func ValidateAssetNameMatchesContract(comp ComponentID, p Platform, name string) error {
	want := AssetName(comp, p)
	if want == "" {
		return fmt.Errorf("update: unsupported component/platform %s %s/%s", comp, p.GOOS, p.GOARCH)
	}
	if name != want {
		return fmt.Errorf("update: asset %q != contract %q", name, want)
	}
	return nil
}

// ParseSHA256SUMSLines returns map basename → hex digest (first field).
func ParseSHA256SUMSLines(body string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sum, name := fields[0], fields[len(fields)-1]
		name = strings.TrimPrefix(name, "./")
		if sum == "" || name == "" {
			continue
		}
		out[name] = strings.ToLower(sum)
	}
	return out
}

// ValidateSUMSCoverRequired checks every required binary (except SHA256SUMS itself) has a checksum entry.
func ValidateSUMSCoverRequired(sums map[string]string, required []string) []string {
	var missing []string
	for _, r := range required {
		if r == "SHA256SUMS" {
			continue
		}
		if sums[r] == "" {
			missing = append(missing, r)
		}
	}
	sort.Strings(missing)
	return missing
}

// UserFacingVersionLabel returns status + display for UI.
// confirmedRunning must be true only after health/version probe for this job.
func UserFacingVersionLabel(desired string, confirmedRunning bool, running string) (status string, display string) {
	desired = strings.TrimPrefix(strings.TrimSpace(desired), "v")
	running = strings.TrimPrefix(strings.TrimSpace(running), "v")
	if confirmedRunning && running != "" {
		return "running", running
	}
	if desired != "" {
		return "target", desired
	}
	if running != "" {
		return "installed_unconfirmed", running
	}
	return "unknown", ""
}
