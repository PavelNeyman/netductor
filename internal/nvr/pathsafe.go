package nvr

import (
	"path/filepath"
	"strings"
)

// PathUnderRoot reports whether absPath is under root (after Clean).
func PathUnderRoot(root, absPath string) bool {
	root = filepath.Clean(root)
	absPath = filepath.Clean(absPath)
	if root == "" || absPath == "" {
		return false
	}
	sep := string(filepath.Separator)
	if absPath == root {
		return true
	}
	return strings.HasPrefix(absPath, root+sep)
}
