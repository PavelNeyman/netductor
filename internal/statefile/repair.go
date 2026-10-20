package statefile

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckReport is one state file probe.
type CheckReport struct {
	Path   string `json:"path"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// CheckJSON tries LoadJSON into empty map/slice via raw message pattern: just open and decode to any.
func CheckJSON(path string) CheckReport {
	var dest any
	err := LoadJSON(path, &dest)
	if IsNotExist(err) {
		return CheckReport{Path: path, OK: true, Detail: "absent (ok)"}
	}
	if err != nil {
		return CheckReport{Path: path, OK: false, Detail: err.Error()}
	}
	return CheckReport{Path: path, OK: true, Detail: "readable"}
}

// RepairFileMode ensures 0600 on existing file (no content rewrite).
func RepairFileMode(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if st.Mode().Perm()&0o077 != 0 {
		return os.Chmod(path, 0o600)
	}
	return nil
}

// EnsureParentDir 0700.
func EnsureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

// FormatReports for CLI/doctor.
func FormatReports(rs []CheckReport) string {
	var b string
	for _, r := range rs {
		tag := "OK"
		if !r.OK {
			tag = "FAIL"
		}
		b += fmt.Sprintf("%s  %s — %s\n", tag, r.Path, r.Detail)
	}
	return b
}
