package update

import (
	"errors"
	"fmt"
	"strings"
)

// FailureClass is a stable token for UI/logs (not localized).
type FailureClass string

const (
	FailResolve     FailureClass = "resolve"
	FailDownload    FailureClass = "download"
	FailChecksum    FailureClass = "checksum"
	FailPermission  FailureClass = "permission"
	FailDisk        FailureClass = "disk"
	FailInstall     FailureClass = "install"
	FailHealth      FailureClass = "health"
	FailConflict    FailureClass = "conflict"
	FailRollback    FailureClass = "rollback"
	FailInterrupted FailureClass = "interrupted"
	FailUnsupported FailureClass = "unsupported"
	FailUnknown     FailureClass = "unknown"
)

// ClassifyFailure maps an error to a FailureClass for operators and tests.
func ClassifyFailure(err error) FailureClass {
	if err == nil {
		return ""
	}
	s := strings.ToLower(err.Error())
	switch {
	case strings.Contains(s, "job already active"), strings.Contains(s, "conflict"):
		return FailConflict
	case strings.HasPrefix(s, "rollback:"), strings.Contains(s, "rollback:"):
		return FailRollback
	case strings.Contains(s, "sha256sums required"), strings.Contains(s, "resolve"):
		return FailResolve
	case strings.Contains(s, "checksum mismatch"), strings.Contains(s, "no checksum"):
		return FailChecksum
	case strings.Contains(s, "no space"), strings.Contains(s, "enospc"):
		return FailDisk
	case strings.Contains(s, "permission denied"), strings.Contains(s, "eacces"):
		return FailPermission
	case strings.Contains(s, "unsupported"), strings.Contains(s, "no asset"):
		return FailUnsupported
	case strings.Contains(s, "download"), strings.Contains(s, "http "):
		return FailDownload
	case strings.Contains(s, "health"), strings.Contains(s, "version probe"), strings.Contains(s, "unit "):
		return FailHealth
	case strings.Contains(s, "install"), strings.Contains(s, "destination"), strings.Contains(s, "preserve previous"):
		return FailInstall
	case strings.Contains(s, "interrupt"), strings.Contains(s, "signal"):
		return FailInterrupted
	default:
		return FailUnknown
	}
}

// ErrRollbackFailed wraps health failure when .prev restore also failed.
func ErrRollbackFailed(health, restore error) error {
	if restore == nil {
		return health
	}
	return fmt.Errorf("rollback: restore previous binary failed: %v (health: %w)", restore, health)
}

func isChecksumish(err error) bool {
	return errors.Is(err, err) && ClassifyFailure(err) == FailChecksum
}
