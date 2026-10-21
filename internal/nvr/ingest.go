package nvr

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
)

var cameraIDRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

// validCameraID rejects path traversal and unsafe ids (M-1).
func validCameraID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || id == "." || id == ".." {
		return false
	}
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") {
		return false
	}
	return cameraIDRe.MatchString(id)
}

// underRoot reports whether absPath is inside root (after Clean; best-effort symlink check).
func underRoot(root, absPath string) bool {
	root = filepath.Clean(root)
	absPath = filepath.Clean(absPath)
	if root == "" {
		return false
	}
	sep := string(os.PathSeparator)
	if absPath == root {
		return true
	}
	return strings.HasPrefix(absPath, root+sep)
}

// IngestSegment writes an uploaded segment under path/<camera_id>/.
func IngestSegment(cameraID, filename string, r io.Reader) (string, int64, error) {
	cameraID = strings.TrimSpace(cameraID)
	if !validCameraID(cameraID) {
		return "", 0, fmt.Errorf("invalid camera_id")
	}
	cfg := LoadConfig()
	root := filepath.Clean(cfg.Path)
	dir := filepath.Join(root, cameraID)
	if !underRoot(root, dir) {
		return "", 0, fmt.Errorf("camera_id escapes nvr root")
	}
	// Resolve root symlinks when root exists; fail closed if resolution fails on existing path.
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		realRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			return "", 0, fmt.Errorf("nvr root EvalSymlinks: %w", err)
		}
		root = realRoot
		dir = filepath.Join(root, cameraID)
		if !underRoot(root, dir) {
			return "", 0, fmt.Errorf("camera_id escapes nvr root")
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", 0, err
	}
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", 0, fmt.Errorf("camera dir EvalSymlinks: %w", err)
	}
	if !underRoot(root, realDir) {
		return "", 0, fmt.Errorf("camera dir escapes nvr root via symlink")
	}
	dir = realDir
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == ".." {
		filename = time.Now().UTC().Format("2006-01-02_15-04-05") + ".mp4"
	}
	low := strings.ToLower(filename)
	ok := false
	for _, s := range []string{".mp4", ".mkv", ".ts", ".m4v"} {
		if strings.HasSuffix(low, s) {
			ok = true
			break
		}
	}
	if !ok {
		filename = filename + ".mp4"
	}
	dest := filepath.Join(dir, filename)
	if !underRoot(root, dest) {
		return "", 0, fmt.Errorf("destination escapes nvr root")
	}
	if realDest, err := filepath.EvalSymlinks(filepath.Dir(dest)); err == nil && !underRoot(root, realDest) {
		return "", 0, fmt.Errorf("destination parent escapes nvr root")
	}
	tmp := dest + ".partial"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, err
	}
	n, err := io.Copy(f, r)
	_ = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return "", 0, err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", 0, err
	}
	AppendEvent("segment", cameraID, filename)
	mc := LoadMotion()
	if mc.AlertOnSegment && InMotionWindow(mc, Now()) {
		notify.AlertOnce("nvr-seg-"+cameraID, "NVR segment: camera "+cameraID+" file "+filename)
	}
	return dest, n, nil
}
