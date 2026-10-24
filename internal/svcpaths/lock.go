package svcpaths

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const failoverLockPath = "/var/lib/netductor/failover.lock"

// acquireFailoverLock exclusive-locks the failover state machine (agent tick + systemd timer).
// Returns unlock func.
func acquireFailoverLock() (func(), error) {
	if err := os.MkdirAll(filepath.Dir(failoverLockPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(failoverLockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failover lock open: %w", err)
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failover lock: %w", err)
	}
	return func() {
		_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
		_ = f.Close()
	}, nil
}
