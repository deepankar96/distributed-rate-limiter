//go:build concurrency

package limiter

import (
	"sync"
	"testing"
)

// This opt-in regression test exercises the known shared-counter race.
// Run with: go test -race -tags=concurrency ./internal/ratelimiter/limiter
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
