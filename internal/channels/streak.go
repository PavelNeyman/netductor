package channels

import (
	"sync"
	"time"
)

// In-process fail streak for collect-tick hysteresis (survives within process lifetime).
var (
	failStreakMu sync.Mutex
	failStreak   = map[string]int{}
	failStreakAt = map[string]time.Time{}
)

// FailStreakInc increments and returns the new streak for key.
func FailStreakInc(key string) int {
	failStreakMu.Lock()
	defer failStreakMu.Unlock()
	failStreak[key]++
	failStreakAt[key] = time.Now()
	return failStreak[key]
}

// FailStreakReset clears the streak when the condition recovers.
func FailStreakReset(key string) {
	failStreakMu.Lock()
	defer failStreakMu.Unlock()
	delete(failStreak, key)
	delete(failStreakAt, key)
}
