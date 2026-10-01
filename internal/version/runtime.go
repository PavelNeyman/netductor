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

func verLess(a, b string) bool {
	a, b = verNorm(a), verNorm(b)
	if a == "" || b == "" {
		return false
	}
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(as) {
			fmt.Sscanf(as[i], "%d", &ai)
		}
		if i < len(bs) {
			fmt.Sscanf(bs[i], "%d", &bi)
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
