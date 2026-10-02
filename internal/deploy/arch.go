package deploy

import (
	"fmt"
	"os"
	"strings"
)

// Supported agent GOARCH asset suffixes (netductor-agent-linux-<arch>).
var agentArches = map[string]bool{
	"amd64":   true,
	"arm64":   true,
	"arm":     true, // GOARM=7
	"mipsle":  true, // OpenWrt ramips (Cudy TR1200, MT7628, …)
	"riscv64": true,
}

// ValidAgentArch reports whether s is a known agent asset arch (or "auto").
func ValidAgentArch(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == "auto" {
		return true
	}
	return agentArches[s]
}

// MapUnameToGoArch maps uname -m / OpenWrt machine strings to Go GOARCH asset names.
func MapUnameToGoArch(uname string) (string, error) {
	u := strings.ToLower(strings.TrimSpace(uname))
	u = strings.Split(u, "\n")[0]
	u = strings.TrimSpace(u)
	switch {
	case u == "x86_64" || u == "amd64":
		return "amd64", nil
	case u == "aarch64" || u == "arm64":
		return "arm64", nil
	case strings.HasPrefix(u, "armv7") || u == "arm" || strings.HasPrefix(u, "armv6"):
		return "arm", nil
	case u == "mips" || u == "mipsel" || strings.HasPrefix(u, "mips"):
		// OpenWrt ramips/mt76x8 (MT7628) is little-endian softfloat → mipsle
		return "mipsle", nil
	case u == "riscv64":
		return "riscv64", nil
	case u == "":
		return "", fmt.Errorf("empty uname -m")
	default:
		return "", fmt.Errorf("unsupported machine %q (set --arch override)", uname)
	}
}

// ProbeRouterArch SSHs to the router and returns GOARCH for the agent asset.
// Also returns raw uname and a short note (opkg/apk presence).
func ProbeRouterArch(password, keyPath, user, host, keyPassphrase string) (goarch, uname, note string, err error) {
	script := `uname -m; command -v opkg >/dev/null && echo PKG=opkg || true; command -v apk >/dev/null && echo PKG=apk || true; [ -f /etc/openwrt_release ] && . /etc/openwrt_release 2>/dev/null; echo DISTRIB_ARCH=${DISTRIB_ARCH:-}; echo DISTRIB_RELEASE=${DISTRIB_RELEASE:-}`
	out, err := runSSHOnPort(factorySSHPort(), password, keyPath, user, host, script, keyPassphrase)
	if err != nil {
		return "", "", "", fmt.Errorf("probe router arch: %w\n%s", err, out)
	}
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Warning:") || strings.HasPrefix(line, "**") {
			continue
		}
		if strings.Contains(line, "post-quantum") || strings.Contains(line, "openssh.com/pq") {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "", "", "", fmt.Errorf("probe router arch: empty output")
	}
	uname = lines[0]
	goarch, err = MapUnameToGoArch(uname)
	if err != nil {
		// fallback: DISTRIB_ARCH hints
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "DISTRIB_ARCH=") {
				da := strings.TrimPrefix(line, "DISTRIB_ARCH=")
				if mapped, e2 := mapOpenWrtArch(da); e2 == nil {
					goarch = mapped
					err = nil
					break
				}
			}
		}
		if err != nil {
			return "", uname, strings.TrimSpace(out), err
		}
	}
	var notes []string
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		notes = append(notes, line)
	}
	note = strings.Join(notes, "; ")
	return goarch, uname, note, nil
}

func mapOpenWrtArch(distrib string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(distrib))
	switch {
	case strings.Contains(d, "aarch64") || strings.Contains(d, "arm64"):
		return "arm64", nil
	case strings.Contains(d, "x86_64") || strings.Contains(d, "amd64"):
		return "amd64", nil
	case strings.HasPrefix(d, "arm_") || strings.Contains(d, "arm_cortex"):
		return "arm", nil
	case strings.Contains(d, "mipsel") || strings.Contains(d, "mips_24") || d == "mipsel_24kc":
		return "mipsle", nil
	case strings.Contains(d, "riscv"):
		return "riscv64", nil
	default:
		return "", fmt.Errorf("unknown DISTRIB_ARCH %q", distrib)
	}
}

// ResolveAgentArch returns override if set and not auto; otherwise probes the router.
func ResolveAgentArch(override, password, keyPath, user, host, keyPassphrase string) (string, error) {
	o := strings.ToLower(strings.TrimSpace(override))
	if o != "" && o != "auto" {
		if !agentArches[o] {
			return "", fmt.Errorf("invalid agent arch %q (want amd64|arm64|arm|mipsle|riscv64|auto)", override)
		}
		fmt.Fprintln(os.Stderr, "==> agent arch override:", o)
		return o, nil
	}
	goarch, uname, note, err := ProbeRouterArch(password, keyPath, user, host, keyPassphrase)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "==> probed router uname=%s → agent arch %s", uname, goarch)
	if note != "" {
		fmt.Fprintf(os.Stderr, " (%s)", note)
	}
	fmt.Fprintln(os.Stderr)
	return goarch, nil
}
