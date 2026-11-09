package tokenbucketlimiter

import (
	"sync"
	"time"
)

type bucket struct {
	token       float64
	lastUpdated time.Time
	mu          sync.Mutex
}

func NewBucket(maxLimit float64) *bucket {
	return &bucket{token: maxLimit, lastUpdated: time.Now()}
}

func (b *bucket) checkLimit(refilRate, maxCapacity float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	newToken := b.calculateRefil(refilRate, now) + b.token
	b.token = min(newToken, maxCapacity)
	if b.token < 1 {
		return false
	} else {
		b.token -= 1
		return true
	}

}

func (b *bucket) calculateRefil(refilRate float64, now time.Time) float64 {
	elapsedTime := now.Sub(b.lastUpdated).Seconds()
	b.lastUpdated = now
	refilAmount := elapsedTime * refilRate

	return refilAmount
}
