package limiter

import "sync"

type Limiter struct {
	mu        sync.Mutex
	remaining int
}

func New(initial int) *Limiter {
	return &Limiter{remaining: initial}
}

type UserLimiters struct {
	mu    sync.Mutex
	users map[string]*Limiter
}

func NewUserLimiters() *UserLimiters {
	u := make(map[string]*Limiter)
	return &UserLimiters{users: u}
}

func (u *UserLimiters) ForUsers(userID string) *Limiter {
	u.mu.Lock()
	defer u.mu.Unlock()
	_, exists := u.users[userID]
	if !exists {
		u.users[userID] = New(5)
	}

	return u.users[userID]
}

func (l *Limiter) CheckLimit() (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.remaining > 0 {
		l.remaining -= 1
		return true, l.remaining
	} else {
		return false, l.remaining
	}
}
