package tokenbucketlimiter

import (
	"sync"
)

type buckets struct {
	buckets     map[string]*bucket
	maxCapacity float64
	refilRate   float64
	mu          sync.Mutex
}

func New(refilRate, maxCapacity float64) *buckets {
	b := make(map[string]*bucket)
	return &buckets{buckets: b, maxCapacity: maxCapacity, refilRate: refilRate}
}

func (b *buckets) forBucket(userID string) *bucket {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.buckets[userID]; !exists {
		b.buckets[userID] = NewBucket(b.maxCapacity)
	}

	return b.buckets[userID]
}

func (b *buckets) Allow(userID string) bool {
	l := b.forBucket(userID)

	allowed := l.checkLimit(b.refilRate, b.maxCapacity)
	return allowed
}
