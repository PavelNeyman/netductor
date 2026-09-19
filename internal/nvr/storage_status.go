package nvr

import (
	"strings"
	"os"
	"path/filepath"
	"syscall"
)

// StorageStatus is a quick health snapshot for doctor/UI.
type StorageStatus struct {
	Path       string  `json:"path"`
	Exists     bool    `json:"exists"`
	Writable   bool    `json:"writable"`
	MountPoint bool    `json:"mount_point"` // true if path is a mountpoint (/proc/mounts)
	EncryptedHint string `json:"encrypted_hint,omitempty"`
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
	st.MountPoint = isMountPoint(cfg.Path)
	if !st.MountPoint {
		st.EncryptedHint = "path is not a mountpoint — consider gocryptfs/LUKS (netductor nvr prepare-storage)"
	}
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


func isMountPoint(path string) bool {
	b, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == abs {
			return true
		}
	}
	// also accept prefix match for nested
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && (fields[1] == path || strings.HasPrefix(abs, fields[1]+"/") && fields[1] != "/") {
			// only exact path counts as "this dir is mount"
			if fields[1] == abs || fields[1] == path {
				return true
			}
		}
	}
	return false
}
