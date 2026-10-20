package tlsle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	preHookPath  = "/etc/letsencrypt/renewal-hooks/pre/netductor-free80.sh"
	postHookPath = "/etc/letsencrypt/renewal-hooks/post/netductor-after80.sh"
	deployHook   = "/etc/letsencrypt/renewal-hooks/deploy/netductor-redirect.sh"
)

// EnsureRenewHooks installs certbot pre/post/deploy hooks so renew frees :80 and restores redirect.
// Safe to call on every stack apply / host-baseline (idempotent).
func EnsureRenewHooks() error {
	if err := os.MkdirAll(filepath.Dir(preHookPath), 0o755); err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(postHookPath), 0o755)
	_ = os.MkdirAll(filepath.Dir(deployHook), 0o755)

	pre := "#!/bin/sh\n# netductor: free :80 for certbot standalone renew\nsystemctl stop netductor-redirect 2>/dev/null || true\n"
	post := "#!/bin/sh\n# netductor: after renew attempt — restore redirect\nsystemctl start netductor-redirect 2>/dev/null || true\nsystemctl restart netductor-redirect 2>/dev/null || true\n"
	deploy := "#!/bin/sh\n# netductor: cert deployed — restart HTTPS redirect on :8443\nsystemctl restart netductor-redirect 2>/dev/null || true\n"

	for path, body := range map[string]string{
		preHookPath:  pre,
		postHookPath: post,
		deployHook:   deploy,
	} {
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

// Port80Listeners returns ss lines that look like listen on TCP port 80.
func Port80Listeners() string {
	out, err := exec.Command("ss", "-tlnp").CombinedOutput()
	if err != nil {
		return ""
	}
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.HasSuffix(f, ":80") || strings.Contains(f, "]:80") {
				lines = append(lines, strings.TrimSpace(line))
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

// HooksInstalled reports whether pre+deploy hooks exist and are executable.
func HooksInstalled() bool {
	for _, p := range []string{preHookPath, deployHook} {
		st, err := os.Stat(p)
		if err != nil || st.Mode()&0o100 == 0 {
			return false
		}
	}
	return true
}
