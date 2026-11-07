package handlers

import (
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/limiter"
)

func CheckLimit(l *limiter.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if res, _ := l.CheckLimit(); res {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}
}
