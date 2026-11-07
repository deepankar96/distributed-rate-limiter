package handlers

import (
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/limiter"
)

func CheckLimit(users *limiter.UserLimiters) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID := r.URL.Query().Get("userid")
		l := users.ForUsers(userID)

		if allowed, _ := l.CheckLimit(); allowed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}
}
