package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/handlers"
	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/limiter"
)

func TestCheckLimitPerUser(t *testing.T) {
	handler := handlers.CheckLimit(limiter.NewUserLimiters())

	// Exhaust Alice's allowance before checking that Bob has his own allowance.
	for _, userID := range []string{"alice", "bob"} {
		for attempt := 1; attempt <= 8; attempt++ {
			request := httptest.NewRequest(http.MethodGet, "/check?userid="+userID, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			want := http.StatusOK
			if attempt > 5 {
				want = http.StatusTooManyRequests
			}
			if response.Code != want {
				t.Fatalf("user %s, attempt %d: status = %d, want %d", userID, attempt, response.Code, want)
			}
		}
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/check?userid=alice", nil))
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("Alice after Bob's requests: status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
}

func TestCheckLimitSharesCollectionState(t *testing.T) {
	users := limiter.NewUserLimiters()
	first := handlers.CheckLimit(users)
	second := handlers.CheckLimit(users)

	for attempt := 1; attempt <= 7; attempt++ {
		handler := first
		if attempt%2 == 0 {
			handler = second
		}
		request := httptest.NewRequest(http.MethodGet, "/check?userid=alice", nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		want := http.StatusOK
		if attempt > 5 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("attempt %d across shared handlers: status = %d, want %d", attempt, response.Code, want)
		}
	}
}
