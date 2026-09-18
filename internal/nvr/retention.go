package nvr

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// SegmentFile is a recorded chunk on disk.
type SegmentFile struct {
	Path    string    `json:"path"`
	Camera  string    `json:"camera"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// RetentionReport is the result of a rotate pass.
type RetentionReport struct {
	Deleted       int     `json:"deleted"`
	DeletedBytes  int64   `json:"deleted_bytes"`
	Kept          int     `json:"kept"`
	TotalBytes    int64   `json:"total_bytes"`
	FreeBytes     int64   `json:"free_bytes,omitempty"`
	Reasons       []string `json:"reasons,omitempty"`
	Path          string  `json:"path"`
	At            int64   `json:"at"`
}

// ListSegmentFiles walks Path for media files.
func ListSegmentFiles(root string) ([]SegmentFile, error) {
	var out []SegmentFile
	if root == "" {
		return out, nil
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if !strings.HasSuffix(name, ".mp4") && !strings.HasSuffix(name, ".mkv") && !strings.HasSuffix(name, ".ts") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		cam := filepath.Base(filepath.Dir(path))
		if cam == filepath.Base(root) {
			cam = ""
		}
		out = append(out, SegmentFile{Path: path, Camera: cam, Size: info.Size(), ModTime: info.ModTime()})
		return nil
	})
	return out, err
}

func diskFree(path string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		// try parent
		if err2 := syscall.Statfs(filepath.Dir(path), &st); err2 != nil {
			return 0
		}
	}
	return int64(st.Bavail) * int64(st.Bsize)
}

// RunRetention deletes old/large segments per Config.
// Policy: (1) age > RetentionDays (2) total size > MaxGB oldest first (3) free < MinFreeGB oldest first.
func RunRetention(cfg Config) (RetentionReport, error) {
	rep := RetentionReport{Path: cfg.Path, At: Now().Unix()}
	if cfg.Path == "" {
		return rep, nil
	}
	_ = os.MkdirAll(cfg.Path, 0o700)
	files, err := ListSegmentFiles(cfg.Path)
	if err != nil {
		return rep, err
	}
	// oldest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.Before(files[j].ModTime)
	})

	var total int64
	for _, f := range files {
		total += f.Size
	}
	rep.TotalBytes = total
	rep.FreeBytes = diskFree(cfg.Path)
	rep.Kept = len(files)

	now := Now()
	toDelete := map[string]string{} // path -> reason

	if cfg.RetentionDays > 0 {
		cut := now.Add(-time.Duration(cfg.RetentionDays) * 24 * time.Hour)
		for _, f := range files {
			if f.ModTime.Before(cut) {
				toDelete[f.Path] = "age"
			}
		}
	}

	maxBytes := int64(cfg.MaxGB * 1024 * 1024 * 1024)
	if cfg.MaxGB > 0 {
		// after age deletes, recompute
		var remain []SegmentFile
		var remainBytes int64
		for _, f := range files {
			if _, ok := toDelete[f.Path]; ok {
				continue
			}
			remain = append(remain, f)
			remainBytes += f.Size
		}
		for i := 0; i < len(remain) && remainBytes > maxBytes; i++ {
			toDelete[remain[i].Path] = "max_gb"
			remainBytes -= remain[i].Size
		}
	}

	if cfg.MinFreeGB > 0 {
		minFree := int64(cfg.MinFreeGB * 1024 * 1024 * 1024)
		free := diskFree(cfg.Path)
		var remain []SegmentFile
		for _, f := range files {
			if _, ok := toDelete[f.Path]; ok {
				continue
			}
			remain = append(remain, f)
		}
		for i := 0; i < len(remain) && free < minFree; i++ {
			toDelete[remain[i].Path] = "min_free"
			free += remain[i].Size
		}
	}

	for path, reason := range toDelete {
		fi, err := os.Stat(path)
		sz := int64(0)
		if err == nil {
			sz = fi.Size()
		}
		if err := os.Remove(path); err != nil {
			rep.Reasons = append(rep.Reasons, "fail:"+path+":"+err.Error())
			continue
		}
		rep.Deleted++
		rep.DeletedBytes += sz
		rep.Kept--
		rep.TotalBytes -= sz
		rep.Reasons = append(rep.Reasons, reason+":"+filepath.Base(path))
	}

	// persist last report
	_ = saveRetentionReport(rep)
	return rep, nil
}

func retentionReportPath() string {
	return filepath.Join(dir(), "last_retention.json")
}

func saveRetentionReport(r RetentionReport) error {
	_ = ensure()
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(retentionReportPath(), append(b, '\n'), 0o600)
}

// LastRetentionReport loads last report if any.
func LastRetentionReport() (RetentionReport, bool) {
	b, err := os.ReadFile(retentionReportPath())
	if err != nil {
		return RetentionReport{}, false
	}
	var r RetentionReport
	if json.Unmarshal(b, &r) != nil {
		return RetentionReport{}, false
	}
	return r, true
}
