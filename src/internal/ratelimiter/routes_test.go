package ratelimiter_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter"
)

func TestRoutes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"health", http.MethodGet, "/healthz", http.StatusOK},
		{"check", http.MethodGet, "/check?userid=alice", http.StatusOK},
		{"head health", http.MethodHead, "/healthz", http.StatusOK},
		{"head check", http.MethodHead, "/check?userid=alice", http.StatusOK},
		{"unknown path", http.MethodGet, "/unknown", http.StatusNotFound},
		{"unsupported health method", http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{"unsupported check method", http.MethodPost, "/check?userid=alice", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ratelimiter.NewHandler()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.want {
				t.Fatalf("%s %s: status = %d, want %d", tt.method, tt.path, response.Code, tt.want)
			}
		})
	}
}

func TestNewHandlerHasIndependentCounters(t *testing.T) {
	first := ratelimiter.NewHandler()
	second := ratelimiter.NewHandler()

	for index, handler := range []http.Handler{first, second} {
		for attempt := 1; attempt <= 7; attempt++ {
			request := httptest.NewRequest(http.MethodGet, "/check?userid=alice", nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			want := http.StatusOK
			if attempt > 5 {
				want = http.StatusTooManyRequests
			}
			if response.Code != want {
				t.Fatalf("handler %d, attempt %d: status = %d, want %d", index+1, attempt, response.Code, want)
			}
		}
	}

	response := httptest.NewRecorder()
	first.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/check?userid=alice", nil))
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("first handler after second handler's requests: status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
}

func TestConcurrentRequestsKeepUserAllowancesIndependent(t *testing.T) {
	const (
		userCount       = 8
		requestsPerUser = 12
		requestCount    = userCount * requestsPerUser
	)
	type result struct {
		user   int
		status int
	}

	handler := ratelimiter.NewHandler()
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
		if allowed[user] != 5 || denied[user] != requestsPerUser-5 {
			t.Errorf("user %d: allowed %d, denied %d; want 5 allowed and %d denied", user, allowed[user], denied[user], requestsPerUser-5)
		}
	}
}
