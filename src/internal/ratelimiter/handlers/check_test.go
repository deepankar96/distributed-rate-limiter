package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/handlers"

	tokenbucketlimiter "github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/tokenBucketLimiter"
)

func TestCheckLimitPerUser(t *testing.T) {
	implementations := []struct {
		name       string
		newLimiter func() handlers.RequestLimiter
	}{

		// Zero refill keeps the HTTP contract tests independent of wall-clock timing.
		{"token bucket", func() handlers.RequestLimiter { return tokenbucketlimiter.New(0, 5) }},
	}
	for _, implementation := range implementations {
		t.Run(implementation.name, func(t *testing.T) {
			handler := handlers.CheckLimit(implementation.newLimiter())
			for _, userID := range []string{"alice", "bob"} {
				for attempt := 1; attempt <= 8; attempt++ {
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/check?userid="+userID, nil))
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
		})
	}
}

func TestCheckLimitSharesCollectionState(t *testing.T) {
	implementations := []struct {
		name    string
		limiter handlers.RequestLimiter
	}{

		{"token bucket", tokenbucketlimiter.New(0, 5)},
	}
	for _, implementation := range implementations {
		t.Run(implementation.name, func(t *testing.T) {
			first := handlers.CheckLimit(implementation.limiter)
			second := handlers.CheckLimit(implementation.limiter)
			for attempt := 1; attempt <= 7; attempt++ {
				handler := first
				if attempt%2 == 0 {
					handler = second
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/check?userid=alice", nil))
				want := http.StatusOK
				if attempt > 5 {
					want = http.StatusTooManyRequests
				}
				if response.Code != want {
					t.Fatalf("attempt %d across shared handlers: status = %d, want %d", attempt, response.Code, want)
				}
			}
		})
	}
}

type requestLimiterFunc func(string) bool

func (f requestLimiterFunc) Allow(userID string) bool { return f(userID) }

func TestCheckLimitUsesInterfaceDecisionAndQueryUserID(t *testing.T) {
	for _, allowed := range []bool{true, false} {
		name := "denied"
		if allowed {
			name = "allowed"
		}
		t.Run(name, func(t *testing.T) {
			const userID = "user+42@example.com"
			calls := 0
			fake := requestLimiterFunc(func(got string) bool {
				calls++
				if got != userID {
					t.Errorf("Allow received user ID %q, want %q", got, userID)
				}
				return allowed
			})
			handler := handlers.CheckLimit(fake)
			if calls != 0 {
				t.Fatal("factory called Allow before any request arrived")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/check?userid="+url.QueryEscape(userID), nil))
			want := http.StatusTooManyRequests
			if allowed {
				want = http.StatusOK
			}
			if response.Code != want {
				t.Errorf("status = %d, want %d", response.Code, want)
			}
			if calls != 1 {
				t.Errorf("Allow called %d times, want 1", calls)
			}
		})
	}
}
