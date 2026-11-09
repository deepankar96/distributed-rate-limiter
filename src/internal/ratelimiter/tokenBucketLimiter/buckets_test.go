package tokenbucketlimiter

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNewCollection(t *testing.T) {
	collection := New(0.5, 3.5)
	if collection.refilRate != 0.5 || collection.maxCapacity != 3.5 {
		t.Fatalf("configuration = (rate %v, capacity %v), want (0.5, 3.5)", collection.refilRate, collection.maxCapacity)
	}
	if collection.buckets == nil || len(collection.buckets) != 0 {
		t.Fatalf("initial map = %v, want non-nil empty map", collection.buckets)
	}
}

func TestForBucketCreatesAndReusesState(t *testing.T) {
	collection := New(0, 3.5)
	first := collection.forBucket("alice")
	if first == nil {
		t.Fatal("forBucket returned nil")
	}
	if first.token != 3.5 || first.lastUpdated.IsZero() {
		t.Fatalf("new bucket state = (%v, %v), want full balance and initialized timestamp", first.token, first.lastUpdated)
	}
	if !first.checkLimit(0, 3.5) {
		t.Fatal("initial request denied")
	}
	updated := first.lastUpdated
	if again := collection.forBucket("alice"); again != first {
		t.Fatal("same user received a different bucket")
	}
	if first.token != 2.5 || !first.lastUpdated.Equal(updated) {
		t.Fatal("lookup reset existing balance or timestamp")
	}
	if len(collection.buckets) != 1 || collection.buckets["alice"] != first {
		t.Fatal("map does not retain the user's bucket")
	}
	if !collection.Allow("alice") || first.token != 1.5 {
		t.Fatal("Allow did not consume the same state returned by forBucket")
	}
}

func TestAllowIndependentUsersAndCollections(t *testing.T) {
	first := New(0, 1)
	second := New(0, 1)
	if !first.Allow("alice") || first.Allow("alice") {
		t.Fatal("first collection should allow alice exactly once")
	}
	if !first.Allow("bob") || first.Allow("bob") {
		t.Fatal("bob should have a balance independent of alice")
	}
	if !second.Allow("alice") || second.Allow("alice") {
		t.Fatal("second collection should have an independent balance for alice")
	}
	if first.forBucket("alice") == first.forBucket("bob") {
		t.Fatal("different users share a bucket")
	}
	if first.forBucket("alice") == second.forBucket("alice") {
		t.Fatal("different collections share a bucket")
	}
}

func TestAllowUsesConfiguredCapacity(t *testing.T) {
	collection := New(0, 2.5)
	for i := 0; i < 2; i++ {
		if !collection.Allow("alice") {
			t.Fatalf("request %d denied before configured capacity exhausted", i+1)
		}
	}
	if collection.Allow("alice") {
		t.Fatal("request allowed with less than one token")
	}
	if got := collection.forBucket("alice").token; got != 0.5 {
		t.Fatalf("remaining tokens = %v, want 0.5", got)
	}
}

func TestAllowUsesConfiguredRefillAndCap(t *testing.T) {
	collection := New(0.5, 2.5)
	b := collection.forBucket("alice")
	b.token = 0
	b.lastUpdated = time.Now().Add(-time.Hour)
	if !collection.Allow("alice") {
		t.Fatal("request denied after sufficient time for configured refill")
	}
	if b.token != 1.5 {
		t.Fatalf("remaining tokens = %v, want configured capacity minus one (1.5)", b.token)
	}
}

func TestForBucketConcurrentSameUser(t *testing.T) {
	const workers = 100
	collection := New(0, 7)
	start := make(chan struct{})
	results := make(chan *bucket, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			results <- collection.forBucket("alice")
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	stored := collection.buckets["alice"]
	if stored == nil || len(collection.buckets) != 1 {
		t.Fatal("concurrent same-user lookup did not create exactly one bucket")
	}
	for got := range results {
		if got != stored {
			t.Fatal("concurrent same-user lookup returned a different pointer")
		}
	}
	if stored.token != 7 {
		t.Fatalf("lookup consumed tokens: got %v, want 7", stored.token)
	}
}

func TestForBucketConcurrentDifferentUsers(t *testing.T) {
	const workers = 100
	collection := New(0, 7)
	type result struct {
		user string
		b    *bucket
	}
	start := make(chan struct{})
	results := make(chan result, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		user := strconv.Itoa(i)
		go func(user string) {
			defer wg.Done()
			<-start
			results <- result{user: user, b: collection.forBucket(user)}
		}(user)
	}
	close(start)
	wg.Wait()
	close(results)

	if len(collection.buckets) != workers {
		t.Fatalf("stored buckets = %d, want %d", len(collection.buckets), workers)
	}
	seen := make(map[*bucket]bool)
	for got := range results {
		if got.b == nil || got.b != collection.buckets[got.user] {
			t.Fatalf("incorrect stored bucket for user %q", got.user)
		}
		if seen[got.b] {
			t.Fatalf("user %q shares another user's bucket", got.user)
		}
		seen[got.b] = true
		if got.b.token != 7 {
			t.Fatalf("user %q balance = %v, want 7", got.user, got.b.token)
		}
	}
}

func TestAllowConcurrentExactCapacity(t *testing.T) {
	const capacity = 23
	collection := New(0, capacity)
	allowed := concurrentAllowed(100, func() bool {
		return collection.Allow("alice")
	})
	if allowed != capacity {
		t.Fatalf("allowed requests = %d, want %d", allowed, capacity)
	}
	if len(collection.buckets) != 1 {
		t.Fatalf("stored buckets = %d, want 1", len(collection.buckets))
	}
	if got := collection.forBucket("alice").token; got != 0 {
		t.Fatalf("remaining tokens = %v, want 0", got)
	}
}
