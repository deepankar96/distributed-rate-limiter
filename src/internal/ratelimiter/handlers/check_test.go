package handlers_test

import (
	"context"
	"errors"
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

type requestLimiterFunc func(context.Context, string) (bool, error)

func (f requestLimiterFunc) Allow(ctx context.Context, userID string) (bool, error) {
	return f(ctx, userID)
}

type checkResponseRecorder struct {
	*httptest.ResponseRecorder
	statusWrites []int
}

func (r *checkResponseRecorder) WriteHeader(status int) {
	r.statusWrites = append(r.statusWrites, status)
	r.ResponseRecorder.WriteHeader(status)
}

func TestCheckLimitUsesInterfaceDecisionContextAndQueryUserID(t *testing.T) {
	backendError := errors.New("private backend connection details")
	tests := []struct {
		name       string
		allowed    bool
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "allowed", allowed: true, wantStatus: http.StatusOK},
		{name: "denied", wantStatus: http.StatusTooManyRequests},
		{name: "error", err: backendError, wantStatus: http.StatusServiceUnavailable, wantBody: "rate limiter unavailable\n"},
		{name: "error overrides allowed", allowed: true, err: backendError, wantStatus: http.StatusServiceUnavailable, wantBody: "rate limiter unavailable\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const userID = "user+42@example.com"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			request := httptest.NewRequest(http.MethodGet, "/check?userid="+url.QueryEscape(userID), nil).WithContext(ctx)
			calls := 0
			fake := requestLimiterFunc(func(gotContext context.Context, gotUserID string) (bool, error) {
				calls++
				if gotContext != request.Context() {
					t.Error("Allow did not receive the exact request context")
				}
				if gotUserID != userID {
					t.Errorf("Allow received user ID %q, want %q", gotUserID, userID)
				}
				return tt.allowed, tt.err
			})
			handler := handlers.CheckLimit(fake)
			if calls != 0 {
				t.Fatal("factory called Allow before any request arrived")
			}
			response := &checkResponseRecorder{ResponseRecorder: httptest.NewRecorder()}
			handler.ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := response.Body.String(); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if len(response.statusWrites) != 1 || response.statusWrites[0] != tt.wantStatus {
				t.Errorf("WriteHeader calls = %v, want [%d]", response.statusWrites, tt.wantStatus)
			}
			if calls != 1 {
				t.Errorf("Allow called %d times, want 1", calls)
			}
		})
	}
}
