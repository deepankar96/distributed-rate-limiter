package limiter

type Limiter struct {
	remaining int
}

func New(initial int) *Limiter {
	return &Limiter{remaining: initial}
}

func (l *Limiter) CheckLimit() (bool, int) {
	if l.remaining > 0 {
		l.remaining -= 1
		return true, l.remaining
	} else {
		return false, l.remaining
	}
}
