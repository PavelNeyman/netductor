package version

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

const FilePath = "/etc/netductor/VERSION"

// Running is the operator-facing installed version.
// Binary is ground truth; VERSION file is synced to match (avoids stale 0.9.121 after partial apply/rollback).
func Running() string {
	bin := firstNonEmpty(
		fromBinary("/usr/local/bin/netductor"),
		fromBinary("/usr/local/bin/netductor-tg"),
	)
	file := ""
	if b, err := os.ReadFile(FilePath); err == nil {
		file = strings.TrimPrefix(strings.TrimSpace(string(b)), "v")
	}
	if bin != "" {
		if file != bin {
			_ = os.WriteFile(FilePath, []byte(bin+"\n"), 0o644)
		}
		return bin
	}
	if file != "" {
		return file
	}
	return strings.TrimPrefix(Release, "v")
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").CombinedOutput()
	if err != nil {
		return ""
	}
	// "netductor 0.9.129 (node)" or "netductor-tg 0.9.129"
	for _, f := range strings.Fields(strings.TrimSpace(string(out))) {
		if len(f) > 0 && f[0] >= '0' && f[0] <= '9' && strings.Contains(f, ".") {
			return strings.TrimPrefix(f, "v")
		}
	}
	return ""
}
