package deploy

import (
	"sync"
	"testing"
	"time"
)

func TestWithHostDeployLockSerializes(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	var order []int
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = withHostDeployLock("203.0.113.9", func() error {
			mu.Lock()
			order = append(order, 1)
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			mu.Lock()
			order = append(order, 2)
			mu.Unlock()
			return nil
		})
	}()
	time.Sleep(10 * time.Millisecond)
	go func() {
		defer wg.Done()
		_ = withHostDeployLock("203.0.113.9", func() error {
			mu.Lock()
			order = append(order, 3)
			mu.Unlock()
			return nil
		})
	}()
	wg.Wait()
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("not serialized: %v", order)
	}
}
