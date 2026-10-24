package version

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const FilePath = "/etc/netductor/VERSION"

func verNorm(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "v")
}

// Less reports whether a is strictly older than b (SemVer major.minor.patch).
func Less(a, b string) bool { return verLess(a, b) }

func verLess(a, b string) bool {
	a, b = verNorm(a), verNorm(b)
	if a == "" || b == "" {
		return false
	}
	ap, okA := parseDotVersion(a)
	bp, okB := parseDotVersion(b)
	if !okA || !okB {
		// Unparseable versions: do not claim ordering (caller may still reject empty min).
		return false
	}
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(ap) {
			ai = ap[i]
		}
		if i < len(bp) {
			bi = bp[i]
		}
		if ai < bi {
			return true
		}
		if ai > bi {
			return false
		}
	}
	return false
}

// ValidDotVersion reports whether s is major.minor.patch digits-only (optional leading v stripped by caller via Less/Norm).
func ValidDotVersion(s string) bool {
	_, ok := parseDotVersion(verNorm(s))
	return ok
}

// parseDotVersion accepts only major.minor.patch style (digits and dots). Rejects prerelease/suffix.
func parseDotVersion(s string) ([]int, bool) {
	if s == "" {
		return nil, false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' {
			return nil, false
		}
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return nil, false // major.minor.patch only
	}
	out := make([]int, 0, 3)
	for _, p := range parts {
		if p == "" {
			return nil, false
		}
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil {
			return nil, false
		}
		if len(p) > 1 && p[0] == '0' {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

func readFileVersion() string {
	b, err := os.ReadFile(FilePath)
	if err != nil {
		return ""
	}
	return verNorm(string(b))
}

// Running is the operator-facing installed version.
// Sources: /etc/netductor/VERSION, `netductor version`, compile-time Release.
// If probe returns an *older* value than VERSION file, trust the file (fixes TG showing 0.9.121
// while CLI/VERSION already show a newer install).
func Running() string {
	file := readFileVersion()
	bin := firstNonEmpty(
		fromBinary("/usr/local/bin/netductor"),
		fromBinary("/usr/local/bin/netductor-tg"),
	)
	self := verNorm(Release)

	chosen := ""
	switch {
	case bin != "" && file != "":
		if verLess(bin, file) {
			fmt.Fprintf(os.Stderr, "version.Running: probe %s older than VERSION file %s — using file\n", bin, file)
			chosen = file
		} else if verLess(file, bin) {
			// binary newer than file — trust binary and sync file
			chosen = bin
			_ = os.WriteFile(FilePath, []byte(bin+"\n"), 0o644)
		} else {
			chosen = bin
		}
	case bin != "":
		chosen = bin
		if file != bin {
			_ = os.WriteFile(FilePath, []byte(bin+"\n"), 0o644)
		}
	case file != "":
		chosen = file
	default:
		chosen = self
	}

	// If still absurdly behind this process's own Release (same package as bot/node), prefer self.
	if self != "" && chosen != "" && verLess(chosen, self) {
		fmt.Fprintf(os.Stderr, "version.Running: chosen %s older than process Release %s — using Release\n", chosen, self)
		chosen = self
		_ = os.WriteFile(FilePath, []byte(self+"\n"), 0o644)
	}
	return chosen
}

// Detail returns bin/file/self for diagnostics (TG Fleet mismatch line).
func Detail() (bin, file, self string) {
	return firstNonEmpty(fromBinary("/usr/local/bin/netductor"), fromBinary("/usr/local/bin/netductor-tg")),
		readFileVersion(),
		verNorm(Release)
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func fromBinary(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").CombinedOutput()
	if err != nil {
		return ""
	}
	// Only parse first line — ignore any trailing noise.
	line := strings.TrimSpace(string(out))
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	for _, f := range strings.Fields(line) {
		if len(f) > 0 && f[0] >= '0' && f[0] <= '9' && strings.Contains(f, ".") {
			return strings.TrimPrefix(f, "v")
		}
	}
	return ""
}
