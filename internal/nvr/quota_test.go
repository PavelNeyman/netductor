package nvr

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIngestCameraQuota(t *testing.T) {
	dir := t.TempDir()
	_ = SaveConfig(Config{Path: dir, RetentionDays: 1, MaxGB: 1, MaxCameraGB: 0.000001, SegmentSec: 60, RotateIntervalSec: 300, RecordEnabled: true})
	// MaxCameraGB ~1 byte effectively after float — use tiny by writing large first via direct file
	cam := filepath.Join(dir, "cam1")
	_ = os.MkdirAll(cam, 0o700)
	_ = os.WriteFile(filepath.Join(cam, "old.mp4"), bytes.Repeat([]byte("x"), 2048), 0o600)
	_, _, err := IngestSegment("cam1", "new.mp4", bytes.NewReader([]byte("hello")))
	if err == nil {
		t.Fatal("expected camera quota error")
	}
}
