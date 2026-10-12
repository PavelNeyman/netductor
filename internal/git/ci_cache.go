package gitstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// CICacheRoot holds dependency caches (not release artifacts).
func CICacheRoot() string {
	return filepath.Join(paths.StateDir(), "ci-cache")
}

// CacheKeyFromWorkDir hashes lockfiles if present.
func CacheKeyFromWorkDir(workDir string) string {
	var parts []byte
	for _, f := range []string{"go.sum", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "Cargo.lock", "poetry.lock"} {
		b, err := os.ReadFile(filepath.Join(workDir, f))
		if err == nil {
			parts = append(parts, b...)
			parts = append(parts, []byte(f)...)
		}
	}
	if len(parts) == 0 {
		h := sha256.Sum256([]byte("empty"))
		return hex.EncodeToString(h[:8])
	}
	h := sha256.Sum256(parts)
	return hex.EncodeToString(h[:12])
}

// CICacheDir for project+key.
func CICacheDir(project, key string) string {
	return filepath.Join(CICacheRoot(), sanitize(project), sanitize(key))
}

// EnsureCICache prepares cache dir and returns env hints for common tools.
func EnsureCICache(project, workDir string) (cacheDir string, env []string) {
	key := CacheKeyFromWorkDir(workDir)
	cacheDir = CICacheDir(project, key)
	_ = os.MkdirAll(cacheDir, 0o755)
	// touch for LRU
	_ = os.WriteFile(filepath.Join(cacheDir, ".stamp"), []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)
	gocache := filepath.Join(cacheDir, "go-build")
	gomod := filepath.Join(cacheDir, "go-mod")
	npm := filepath.Join(cacheDir, "npm")
	_ = os.MkdirAll(gocache, 0o755)
	_ = os.MkdirAll(gomod, 0o755)
	_ = os.MkdirAll(npm, 0o755)
	env = []string{
		"GOCACHE=" + gocache,
		"GOMODCACHE=" + gomod,
		"npm_config_cache=" + npm,
		"NETDUCTOR_CI_CACHE=" + cacheDir,
	}
	return cacheDir, env
}

// PruneCICache deletes cache dirs older than keepDays or over max projects.
func PruneCICache(keepDays int) (removed int, err error) {
	if keepDays <= 0 {
		keepDays = 14
	}
	root := CICacheRoot()
	cutoff := time.Now().Add(-time.Duration(keepDays) * 24 * time.Hour)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.Name() == ".stamp" && info.ModTime().Before(cutoff) {
			dir := filepath.Dir(path)
			if dir != root {
				_ = os.RemoveAll(dir)
				removed++
			}
		}
		return nil
	})
	return removed, nil
}

// FormatCacheInfo for doctor.
func FormatCacheInfo() string {
	root := CICacheRoot()
	var size int64
	_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return fmt.Sprintf("ci-cache root=%s ~%dMB", root, size/1024/1024)
}
