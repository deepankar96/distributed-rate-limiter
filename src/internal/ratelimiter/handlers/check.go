package handlers

import (
	"context"
	"net/http"
)

type RequestLimiter interface {
	Allow(ctx context.Context, userId string) (bool, error)
}

func CheckLimit(users RequestLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID := r.URL.Query().Get("userid")
		allowed, err := users.Allow(r.Context(), userID)
		if err != nil {
			http.Error(w, "rate limiter unavailable", http.StatusServiceUnavailable)
			return
		}

		if allowed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}
}
