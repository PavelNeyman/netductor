package nvr

import (
	"fmt"
	"github.com/PavelNeyman/netductor/internal/notify"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IngestSegment writes an uploaded segment under path/<camera_id>/.
func IngestSegment(cameraID, filename string, r io.Reader) (string, int64, error) {
	cameraID = strings.TrimSpace(cameraID)
	if cameraID == "" {
		return "", 0, fmt.Errorf("camera_id required")
	}
	cfg := LoadConfig()
	dir := filepath.Join(cfg.Path, cameraID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", 0, err
	}
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == ".." {
		filename = time.Now().UTC().Format("2006-01-02_15-04-05") + ".mp4"
	}
	// only allow safe media suffixes
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
