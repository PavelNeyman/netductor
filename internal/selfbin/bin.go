// Package selfbin resolves the real CLI binary.
// Never returns a go test binary: exec'ing *.test re-runs tests (fork bomb).
package selfbin

import (
	"os"
	"path/filepath"
	"strings"
)

// CLI returns a binary safe to exec as the product CLI.
// If the current process is a test binary, falls back to fallback (default /usr/local/bin/netductor).
func CLI(fallback string) string {
	if fallback == "" {
		fallback = "/usr/local/bin/netductor"
	}
	b, err := os.Executable()
	if err != nil || b == "" || IsTestBinary(b) {
		return fallback
	}
	return b
}

// IsTestBinary reports whether path is a go test executable.
func IsTestBinary(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, ".test") || strings.Contains(path, "/_test/")
}
