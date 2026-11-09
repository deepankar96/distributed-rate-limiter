package tokenbucketlimiter

import (
	"sync"
	"testing"
	"time"
)

func TestNewBucket(t *testing.T) {
	before := time.Now()
	b := NewBucket(3.5)
	after := time.Now()

	if b.token != 3.5 {
		t.Fatalf("initial tokens = %v, want 3.5", b.token)
	}
	if b.lastUpdated.Before(before) || b.lastUpdated.After(after) {
		t.Fatalf("initial timestamp %v not between %v and %v", b.lastUpdated, before, after)
	}
}

func TestCheckLimitBalancesWithoutRefill(t *testing.T) {
	for _, tc := range []struct {
		name    string
		tokens  float64
		allowed bool
		want    float64
	}{
		{name: "empty", tokens: 0, allowed: false, want: 0},
		{name: "fractional", tokens: 0.75, allowed: false, want: 0.75},
		{name: "exactly one", tokens: 1, allowed: true, want: 0},
		{name: "fractional remainder", tokens: 1.5, allowed: true, want: 0.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBucket(tc.tokens)
			b.lastUpdated = time.Now().Add(-time.Hour)
			before := time.Now()
			if got := b.checkLimit(0, 5); got != tc.allowed {
				t.Fatalf("checkLimit() = %v, want %v", got, tc.allowed)
			}
			after := time.Now()
			if b.token != tc.want {
				t.Errorf("remaining tokens = %v, want %v", b.token, tc.want)
			}
			if b.lastUpdated.Before(before) || b.lastUpdated.After(after) {
				t.Errorf("timestamp %v not updated to check time between %v and %v", b.lastUpdated, before, after)
			}
		})
	}
}

func TestCheckLimitBurstAndExhaustion(t *testing.T) {
	b := NewBucket(3.5)
	for i := 0; i < 3; i++ {
		if !b.checkLimit(0, 3.5) {
			t.Fatalf("burst request %d denied", i+1)
		}
	}
	for i := 0; i < 3; i++ {
		if b.checkLimit(0, 3.5) {
			t.Fatalf("exhausted request %d allowed", i+1)
		}
		if b.token != 0.5 {
			t.Fatalf("denied request changed fractional balance: got %v, want 0.5", b.token)
		}
	}
}

func TestCheckLimitCapsBeforeConsumption(t *testing.T) {
	b := NewBucket(20)
	if !b.checkLimit(0, 3.5) {
		t.Fatal("request with excess tokens denied")
	}
	if b.token != 2.5 {
		t.Fatalf("remaining tokens = %v, want 2.5 (capacity minus one)", b.token)
	}
}

func TestCheckLimitPositiveRefillSaturates(t *testing.T) {
	b := NewBucket(0)
	b.lastUpdated = time.Now().Add(-time.Hour)
	before := time.Now()

	// checkLimit owns its clock. Saturation avoids a threshold-sensitive assertion
	// about the exact elapsed time; exact refill arithmetic is tested separately.
	if !b.checkLimit(0.5, 3.5) {
		t.Fatal("request after sufficient refill denied")
	}
	after := time.Now()
	if b.token != 2.5 {
		t.Fatalf("remaining tokens = %v, want 2.5", b.token)
	}
	if b.lastUpdated.Before(before) || b.lastUpdated.After(after) {
		t.Fatalf("timestamp %v not updated to check time between %v and %v", b.lastUpdated, before, after)
	}
}

func TestCalculateRefil(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		rate    float64
		elapsed time.Duration
		want    float64
	}{
		{name: "no elapsed time", rate: 2, elapsed: 0, want: 0},
		{name: "zero rate", rate: 0, elapsed: 3 * time.Second, want: 0},
		{name: "whole seconds", rate: 3, elapsed: 2 * time.Second, want: 6},
		{name: "fractional rate and seconds", rate: 0.5, elapsed: 1500 * time.Millisecond, want: 0.75},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &bucket{token: 0.25, lastUpdated: start}
			now := start.Add(tc.elapsed)
			if got := b.calculateRefil(tc.rate, now); got != tc.want {
				t.Errorf("calculateRefil() = %v, want %v", got, tc.want)
			}
			if !b.lastUpdated.Equal(now) {
				t.Errorf("timestamp = %v, want %v", b.lastUpdated, now)
			}
			if b.token != 0.25 {
				t.Errorf("refill calculation changed stored balance to %v", b.token)
			}
		})
	}
}

func TestCalculateRefilSuccessiveTimeCredits(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	b := &bucket{lastUpdated: start}
	// Sub-token credits matter even when a request cannot yet be allowed.
	// Each calculation must credit only the time since the previous update.
	for _, tc := range []struct {
		elapsed time.Duration
		want    float64
	}{
		{elapsed: 500 * time.Millisecond, want: 0.25},
		{elapsed: time.Second, want: 0.25},
		{elapsed: time.Second, want: 0},
		{elapsed: 2500 * time.Millisecond, want: 0.75},
	} {
		now := start.Add(tc.elapsed)
		if got := b.calculateRefil(0.5, now); got != tc.want {
			t.Fatalf("credit at %v = %v, want %v", tc.elapsed, got, tc.want)
		}
		if !b.lastUpdated.Equal(now) {
			t.Fatalf("timestamp = %v, want %v", b.lastUpdated, now)
		}
	}
}

func TestCheckLimitConcurrentExactCapacity(t *testing.T) {
	const capacity = 17
	b := NewBucket(capacity)
	allowed := concurrentAllowed(100, func() bool {
		return b.checkLimit(0, capacity)
	})
	if allowed != capacity {
		t.Fatalf("allowed requests = %d, want %d", allowed, capacity)
	}
	if b.token != 0 {
		t.Fatalf("remaining tokens = %v, want 0", b.token)
	}
}

// Results are aggregated only after all workers finish; the start channel releases
// workers together without sleeps or concurrent writes to a test-side counter.
func concurrentAllowed(workers int, allow func() bool) int {
	start := make(chan struct{})
	results := make(chan bool, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			results <- allow()
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	allowed := 0
	for result := range results {
		if result {
			allowed++
		}
	}
	return allowed
}
