package ratelimiter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter"
	tokenbucketlimiter "github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/tokenBucketLimiter"
)

type requestLimiterFunc func(context.Context, string) (bool, error)

func (f requestLimiterFunc) Allow(ctx context.Context, userID string) (bool, error) {
	return f(ctx, userID)
}

func TestRoutes(t *testing.T) {
	const userID = "user+42@example.com"
	checkPath := "/check?userid=" + url.QueryEscape(userID)
	tests := []struct {
		name      string
		method    string
		path      string
		want      int
		wantCalls int
	}{
		{"health", http.MethodGet, "/healthz", http.StatusOK, 0},
		{"check", http.MethodGet, checkPath, http.StatusOK, 1},
		{"head health", http.MethodHead, "/healthz", http.StatusOK, 0},
		{"head check", http.MethodHead, checkPath, http.StatusOK, 1},
		{"unknown path", http.MethodGet, "/unknown", http.StatusNotFound, 0},
		{"unsupported health method", http.MethodPost, "/healthz", http.StatusMethodNotAllowed, 0},
		{"unsupported check method", http.MethodPost, checkPath, http.StatusMethodNotAllowed, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			request := httptest.NewRequest(tt.method, tt.path, nil).WithContext(ctx)
			limiter := tokenbucketlimiter.New(0, 30)
			calls := 0
			fake := requestLimiterFunc(func(gotContext context.Context, gotUserID string) (bool, error) {
				calls++
				if gotContext != request.Context() {
					t.Error("Allow did not receive the exact request context")
				}
				if gotUserID != userID {
					t.Errorf("Allow received user ID %q, want %q", gotUserID, userID)
				}
				return limiter.Allow(gotContext, gotUserID)
			})
			handler := ratelimiter.NewHandler(fake)
			if calls != 0 {
				t.Fatal("factory called Allow before any request arrived")
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.want {
				t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, response.Code, tt.want)
			}
			if calls != tt.wantCalls {
				t.Errorf("Allow called %d times, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestNewHandlerHasIndependentBuckets(t *testing.T) {
	const capacity = 30
	first := ratelimiter.NewHandler(tokenbucketlimiter.New(0, capacity))
	second := ratelimiter.NewHandler(tokenbucketlimiter.New(0, capacity))

	for index, handler := range []http.Handler{first, second} {
		for attempt := 1; attempt <= capacity+1; attempt++ {
			request := httptest.NewRequest(http.MethodGet, "/check?userid=alice", nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			want := http.StatusOK
			if attempt > capacity {
				want = http.StatusTooManyRequests
			}
			if response.Code != want {
				t.Fatalf("handler %d, attempt %d: status = %d, want %d", index+1, attempt, response.Code, want)
			}
		}
	}
}

func TestNewHandlerSharesInjectedBuckets(t *testing.T) {
	const capacity = 5
	limiter := tokenbucketlimiter.New(0, capacity)
	routers := []http.Handler{ratelimiter.NewHandler(limiter), ratelimiter.NewHandler(limiter)}

	for attempt := 1; attempt <= capacity+2; attempt++ {
		request := httptest.NewRequest(http.MethodGet, "/check?userid=alice", nil)
		response := httptest.NewRecorder()
		routers[(attempt-1)%len(routers)].ServeHTTP(response, request)

		want := http.StatusOK
		if attempt > capacity {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("attempt %d across shared routers: status = %d, want %d", attempt, response.Code, want)
		}
	}
}

func TestConcurrentRequestsKeepUserAllowancesIndependent(t *testing.T) {
	const (
		userCount       = 8
		requestsPerUser = 40
		capacity        = 30
		requestCount    = userCount * requestsPerUser
	)
	type result struct {
		user   int
		status int
	}

	handler := ratelimiter.NewHandler(tokenbucketlimiter.New(0, capacity))
	start := make(chan struct{})
	results := make(chan result, requestCount)
	var wg sync.WaitGroup
	for request := 0; request < requestCount; request++ {
		wg.Add(1)
		go func(user int) {
			defer wg.Done()
			<-start
			response := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/check?userid="+strconv.Itoa(user), nil)
			handler.ServeHTTP(response, req)
			results <- result{user: user, status: response.Code}
		}(request % userCount)
	}
	close(start)
	wg.Wait()
	close(results)

	var allowed, denied [userCount]int
	for got := range results {
		switch got.status {
		case http.StatusOK:
			allowed[got.user]++
		case http.StatusTooManyRequests:
			denied[got.user]++
		default:
			t.Errorf("user %d got unexpected status %d", got.user, got.status)
		}
	}
	for user := 0; user < userCount; user++ {
		if allowed[user] != capacity || denied[user] != requestsPerUser-capacity {
			t.Errorf("user %d: allowed %d, denied %d; want %d allowed and %d denied", user, allowed[user], denied[user], capacity, requestsPerUser-capacity)
		}
	}
}
