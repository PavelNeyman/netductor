package update

import (
	"fmt"
	"strings"
)

// Edge agent physical identity (OpenWrt / edge hosts). Logical component ID is ComponentAgent ("agent").
// Decision (Phase 4): do NOT rename the binary or config paths until a proven migration exists.
const (
	EdgeBinaryPrimary  = "/usr/sbin/netductor-agent"
	EdgeBinaryFallback = "/usr/bin/netductor-agent"
	EdgeConfigDir      = "/etc/netductor-agent"
	EdgeConfigFile     = "/etc/netductor-agent/config"
	EdgeMTLSDir        = "/etc/netductor-agent/mtls"
	EdgeInitScript     = "/etc/init.d/netductor-agent" // procd when present
	EdgeUpdateConfirm  = "confirm=yes"                 // required by agent_update
)

// SupportedEdgeAgentGOARCH is the set of GOARCH values we publish as netductor-agent-linux-*.
var SupportedEdgeAgentGOARCH = []string{"amd64", "arm64", "arm", "mipsle", "riscv64"}

// NormalizeEdgeGOARCH maps probe/uname hints to a published GOARCH or rejects ambiguity.
// Does not invent mips (big-endian) or armv6→arm without explicit known GOARM policy.
func NormalizeEdgeGOARCH(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "linux/")
	s = strings.TrimPrefix(s, "goarch=")
	switch s {
	case "amd64", "x86_64", "x64":
		return "amd64", nil
	case "arm64", "aarch64":
		return "arm64", nil
	case "arm", "armv7", "armv7l", "armhf":
		return "arm", nil // published agent is GOARM=7 class
	case "armv6", "armv6l":
		return "", fmt.Errorf("edge arch %q ambiguous: armv6 not published (need explicit arm GOARM=6 asset or refuse)", raw)
	case "mipsle", "mipsel", "mipsel_24kc", "mipsel_24kc_24kf":
		return "mipsle", nil
	case "mips", "mips_24kc":
		return "", fmt.Errorf("edge arch %q is big-endian mips — not published as mipsle; refuse auto-select", raw)
	case "riscv64":
		return "riscv64", nil
	case "", "auto":
		return "", fmt.Errorf("edge arch required (empty/auto)")
	default:
		// Already a supported token?
		for _, a := range SupportedEdgeAgentGOARCH {
			if s == a {
				return a, nil
			}
		}
		return "", fmt.Errorf("unsupported edge arch %q", raw)
	}
}

// EdgeBinaryCandidates returns install paths in preference order (no rename).
func EdgeBinaryCandidates() []string {
	return []string{EdgeBinaryPrimary, EdgeBinaryFallback}
}

// AgentAssetNameNormalized returns release asset after NormalizeEdgeGOARCH.
func AgentAssetNameNormalized(rawArch string) (string, error) {
	a, err := NormalizeEdgeGOARCH(rawArch)
	if err != nil {
		return "", err
	}
	return AgentAssetName(a), nil
}
