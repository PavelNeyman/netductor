package gitstore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// QueueBuildHostBackupMirror stages latest primary backup for build-hosts (triangle phase 1).
func QueueBuildHostBackupMirror() error {
	src := filepath.Join(paths.StateDir(), "backups")
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	var latest, latestName string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.Contains(n, "netductor-") {
			continue
		}
		if n > latestName {
			latestName = n
			latest = filepath.Join(src, n)
		}
	}
	if latest == "" {
		return fmt.Errorf("no primary backup found")
	}
	dstDir := filepath.Join(paths.StateDir(), "build-backup-mirror")
	_ = os.MkdirAll(dstDir, 0o700)
	dst := filepath.Join(dstDir, latestName)
	if _, err := os.Stat(dst); err == nil {
		_ = os.WriteFile(filepath.Join(dstDir, "LATEST"), []byte(latestName+"\n"), 0o600)
		return nil
	}
	if err := exec.Command("cp", "-a", latest, dst).Run(); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dstDir, "LATEST"), []byte(latestName+"\n"), 0o600)
}

// LatestBuildBackupMirror returns name and path of staged primary backup.
func LatestBuildBackupMirror() (name, path string, err error) {
	dir := filepath.Join(paths.StateDir(), "build-backup-mirror")
	b, e := os.ReadFile(filepath.Join(dir, "LATEST"))
	if e != nil {
		return "", "", fmt.Errorf("no mirror: %w", e)
	}
	name = strings.TrimSpace(string(b))
	path = filepath.Join(dir, name)
	if _, e := os.Stat(path); e != nil {
		return "", "", e
	}
	return name, path, nil
}
