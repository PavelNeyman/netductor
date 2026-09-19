package nvr

import (
	"os"
	"path/filepath"
	"syscall"
)

// StorageStatus is a quick health snapshot for doctor/UI.
type StorageStatus struct {
	Path       string  `json:"path"`
	Exists     bool    `json:"exists"`
	Writable   bool    `json:"writable"`
	FreeGB     float64 `json:"free_gb"`
	TotalGB    float64 `json:"total_gb"`
	SegmentCnt int     `json:"segment_cnt"`
	SegmentGB  float64 `json:"segment_gb"`
}

// GetStorageStatus probes configured path.
func GetStorageStatus() StorageStatus {
	cfg := LoadConfig()
	st := StorageStatus{Path: cfg.Path}
	fi, err := os.Stat(cfg.Path)
	if err != nil || !fi.IsDir() {
		return st
	}
	st.Exists = true
	test := filepath.Join(cfg.Path, ".nvr_write_test")
	if err := os.WriteFile(test, []byte("ok"), 0o600); err == nil {
		st.Writable = true
		_ = os.Remove(test)
	}
	var s syscall.Statfs_t
	if syscall.Statfs(cfg.Path, &s) == nil {
		st.FreeGB = float64(s.Bavail*uint64(s.Bsize)) / (1024 * 1024 * 1024)
		st.TotalGB = float64(s.Blocks*uint64(s.Bsize)) / (1024 * 1024 * 1024)
	}
	files, _ := ListSegmentFiles(cfg.Path)
	st.SegmentCnt = len(files)
	var sum int64
	for _, f := range files {
		sum += f.Size
	}
	st.SegmentGB = float64(sum) / (1024 * 1024 * 1024)
	return st
}
