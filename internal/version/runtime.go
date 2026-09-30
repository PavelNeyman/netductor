package version

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

const FilePath = "/etc/netductor/VERSION"

// Running is the operator-facing installed version (prefer on-disk VERSION).
func Running() string {
	if b, err := os.ReadFile(FilePath); err == nil {
		s := strings.TrimPrefix(strings.TrimSpace(string(b)), "v")
		if s != "" {
			return s
		}
	}
	if s := fromBinary("/usr/local/bin/netductor"); s != "" {
		return s
	}
	if s := fromBinary("/usr/local/bin/netductor-tg"); s != "" {
		return s
	}
	return strings.TrimPrefix(Release, "v")
}

func fromBinary(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, f := range strings.Fields(strings.TrimSpace(string(out))) {
		if len(f) > 0 && f[0] >= '0' && f[0] <= '9' && strings.Contains(f, ".") {
			return strings.TrimPrefix(f, "v")
		}
	}
	return ""
}
