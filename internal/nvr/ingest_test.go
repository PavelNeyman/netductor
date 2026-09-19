package nvr

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIngestSegment(t *testing.T) {
	dir := t.TempDir()
	_ = SaveConfig(Config{Path: dir, RetentionDays: 1, MaxGB: 10, SegmentSec: 60, RotateIntervalSec: 300})
	path, n, err := IngestSegment("cam1", "clip.mp4", bytes.NewReader([]byte("hello")))
	if err != nil || n != 5 {
		t.Fatalf("%v %d %s", err, n, path)
	}
	if _, err := os.Stat(filepath.Join(dir, "cam1", "clip.mp4")); err != nil {
		t.Fatal(err)
	}
}
