package update

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// withHostComponentLock holds an exclusive cross-process flock for component/instance
// for the duration of fn. Non-blocking: concurrent Apply/Rollback fails fast.
func withHostComponentLock(component ComponentID, instance InstanceID, fn func() error) error {
	if instance == "" {
		instance = "local"
	}
	dir := filepath.Join(paths.StateDir(), "update", "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("update: lock dir: %w", err)
	}
	safe := string(component) + "-" + string(instance)
	lockPath := filepath.Join(dir, safe+".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("update: open lock: %w", err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("update: component %s/%s apply already in progress", component, instance)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

// hostLockPath is exposed for tests.
func hostLockPath(component ComponentID, instance InstanceID) string {
	if instance == "" {
		instance = "local"
	}
	return filepath.Join(paths.StateDir(), "update", "locks", string(component)+"-"+string(instance)+".lock")
}

// tryHostLockOnce attempts LOCK_EX|NB and holds until unlock is called (tests).
func tryHostLockOnce(component ComponentID, instance InstanceID) (unlock func(), err error) {
	dir := filepath.Join(paths.StateDir(), "update", "locks")
	_ = os.MkdirAll(dir, 0o700)
	f, err := os.OpenFile(hostLockPath(component, instance), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

// sleep holds the lock briefly for concurrent tests.
func holdHostLock(component ComponentID, instance InstanceID, d time.Duration) (done <-chan struct{}) {
	ch := make(chan struct{})
	go func() {
		_ = withHostComponentLock(component, instance, func() error {
			time.Sleep(d)
			close(ch)
			return nil
		})
	}()
	return ch
}
