package limiter

import (
	"strconv"
	"sync"
	"testing"
)

// Run with -race to check synchronization as well as the final allowance.
func TestCheckLimitConcurrent(t *testing.T) {
	const (
		initial           = 4096
		workers           = 64
		attemptsPerWorker = 128
	)
	l := New(initial)
	start := make(chan struct{})
	results := make(chan int, workers)
	var wg sync.WaitGroup

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			accepted := 0
			for attempt := 0; attempt < attemptsPerWorker; attempt++ {
				if allowed, _ := l.CheckLimit(); allowed {
					accepted++
				}
			}
			results <- accepted
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	accepted := 0
	for count := range results {
		accepted += count
	}
	if accepted != initial {
		t.Errorf("accepted %d calls, want %d", accepted, initial)
	}
	allowed, remaining := l.CheckLimit()
	if allowed || remaining != 0 {
		t.Errorf("after workers finish: got (%v, %d), want (false, 0)", allowed, remaining)
	}
}

func TestForUsersConcurrentLookupAndCreation(t *testing.T) {
	const (
		workers   = 128
		userCount = 8
	)
	type result struct {
		userID  string
		limiter *Limiter
	}

	users := NewUserLimiters()
	start := make(chan struct{})
	results := make(chan result, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()
			<-start
			results <- result{userID: userID, limiter: users.ForUsers(userID)}
		}(strconv.Itoa(worker % userCount))
	}
	close(start)
	wg.Wait()
	close(results)

	seen := make(map[string]*Limiter)
	for got := range results {
		if got.limiter == nil {
			t.Fatalf("user %q got a nil limiter", got.userID)
		}
		if previous, exists := seen[got.userID]; exists && previous != got.limiter {
			t.Fatalf("user %q received different limiter instances", got.userID)
		}
		seen[got.userID] = got.limiter
	}
	if len(seen) != userCount {
		t.Fatalf("got %d users, want %d", len(seen), userCount)
	}
	for userID, l := range seen {
		if l.remaining != 5 {
			t.Errorf("lookup consumed allowance for user %q: got %d, want 5", userID, l.remaining)
		}
		if got := users.ForUsers(userID); got != l {
			t.Errorf("stored limiter for user %q differs from the returned instance", userID)
		}
	}
}
