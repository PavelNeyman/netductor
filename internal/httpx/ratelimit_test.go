package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPlaneLimiter(t *testing.T) {
	lim := NewPlaneLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !lim.Allow("1.2.3.4") {
			t.Fatalf("allow %d", i)
		}
	}
	if lim.Allow("1.2.3.4") {
		t.Fatal("should rate limit")
	}
	if !lim.Allow("9.9.9.9") {
		t.Fatal("other IP")
	}
}

func TestMiddleware(t *testing.T) {
	lim := NewPlaneLimiter(1, time.Minute)
	h := lim.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("first %d", w.Code)
	}
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r)
	if w2.Code != 429 {
		t.Fatalf("second %d", w2.Code)
	}
}
