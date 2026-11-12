package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiterPerIP(t *testing.T) {
	rl := NewRateLimiter(5, 5)
	ok := rl.Wrap(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	call := func(xff string) int {
		r := httptest.NewRequest(http.MethodPost, "/api/ask", nil)
		r.Header.Set("X-Forwarded-For", xff)
		w := httptest.NewRecorder()
		ok(w, r)
		return w.Code
	}

	for i := 1; i <= 5; i++ {
		if code := call("1.1.1.1"); code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, code)
		}
	}
	if code := call("1.1.1.1"); code != http.StatusTooManyRequests {
		t.Fatalf("6th request: got %d, want 429", code)
	}
	// first X-Forwarded-For entry is the client; another client is unaffected
	if code := call("2.2.2.2, 10.0.0.1"); code != http.StatusOK {
		t.Fatalf("other IP: got %d, want 200", code)
	}
}
