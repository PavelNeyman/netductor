package paths

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// MigrateFromOpt moves legacy /opt/netductor data into FHS locations (best-effort).
func MigrateFromOpt() error {
	legacy := "/opt/netductor"
	if st, err := os.Stat(legacy); err != nil || !st.IsDir() {
		return nil
	}
	pairs := [][2]string{
		{filepath.Join(legacy, "profiles"), ProfilesDir()},
		{filepath.Join(legacy, "lampac"), LampacDir()},
		{filepath.Join(legacy, "runtime", "telegram"), TelegramDir()},
		{filepath.Join(legacy, "runtime", "api", "admin"), AdminRoot()},
		{filepath.Join(legacy, "scripts"), ScriptsDir()},
	}
	for _, p := range pairs {
		src, dst := p[0], p[1]
		if st, err := os.Stat(src); err != nil || !st.IsDir() {
			continue
		}
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		// if dst empty, move; else rsync-ish copy
		if _, err := os.Stat(dst); err != nil {
			if err := os.Rename(src, dst); err != nil {
				_ = exec.Command("cp", "-a", src+"/.", dst+"/").Run()
			}
		} else {
			_ = exec.Command("cp", "-an", src+"/.", dst+"/").Run()
		}
		fmt.Fprintf(os.Stderr, "migrate: %s → %s\n", src, dst)
	}
	// binaries: prefer /usr/local/bin already
	for _, name := range []string{"netductor", "netductor-tg", "netductor-telegram-bot"} {
		src := filepath.Join(legacy, "bin", name)
		dst := filepath.Join(BinDir(), name)
		if _, err := os.Stat(src); err == nil {
			_ = exec.Command("install", "-m", "755", src, dst).Run()
		}
	}
	return nil
}
