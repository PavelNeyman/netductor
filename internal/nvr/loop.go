package nvr

import (
	"log"
	"sync"
	"time"
)

var (
	loopOnce sync.Once
	loopStop chan struct{}
)

// StartBackground starts retention loop (idempotent).
func StartBackground() {
	loopOnce.Do(func() {
		loopStop = make(chan struct{})
		go retentionLoop()
	})
}

func retentionLoop() {
	for {
		cfg := LoadConfig()
		iv := time.Duration(cfg.RotateIntervalSec) * time.Second
		if iv < 60*time.Second {
			iv = 60 * time.Second
		}
		select {
		case <-loopStop:
			return
		case <-time.After(iv):
			if cfg.Path == "" {
				continue
			}
			if _, err := RunRetention(cfg); err != nil {
				log.Printf("nvr retention: %v", err)
			}
		}
	}
}
