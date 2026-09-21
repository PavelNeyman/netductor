package httpx

import (
	"testing"
	"time"
)

func TestPlaneLimiter(t *testing.T) {
	lim := NewPlaneLimiter(3, time.Minute)
	lim.banAfter = 100 // no ban in this test
	for i := 0; i < 3; i++ {
		if !lim.Allow("1.2.3.4") {
			t.Fatalf("allow %d", i)
		}
	}
	if lim.Allow("1.2.3.4") {
		t.Fatal("should deny")
	}
	if !lim.Allow("9.9.9.9") {
		t.Fatal("other IP")
	}
}

func TestPlaneLimiterBan(t *testing.T) {
	lim := NewPlaneLimiter(1, time.Minute)
	lim.banAfter = 2
	lim.banTTL = time.Hour
	_ = lim.Allow("1.1.1.1")
	if lim.Allow("1.1.1.1") { // deny #1
		t.Fatal("expect deny")
	}
	if lim.Allow("1.1.1.1") { // deny #2 -> ban
		t.Fatal("expect deny")
	}
	if lim.Allow("1.1.1.1") {
		t.Fatal("banned")
	}
}
