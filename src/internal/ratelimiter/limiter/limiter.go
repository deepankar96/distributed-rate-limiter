package limiter

type Limiter struct {
	remaining int
}

func New(initial int) *Limiter {
	return &Limiter{remaining: initial}
}

type UserLimiters struct {
	users map[string]*Limiter
}

func NewUserLimiters() *UserLimiters {
	u := make(map[string]*Limiter)
	return &UserLimiters{users: u}
}

func (u *UserLimiters) ForUsers(userID string) *Limiter {
	_, exists := u.users[userID]
	if !exists {
		u.users[userID] = New(5)
	}

	return u.users[userID]
}

func (l *Limiter) CheckLimit() (bool, int) {
	if l.remaining > 0 {
		l.remaining -= 1
		return true, l.remaining
	} else {
		return false, l.remaining
	}
}
