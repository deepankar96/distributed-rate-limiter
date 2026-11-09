package handlers

import (
	"net/http"
)

type RequestLimiter interface {
	Allow(userId string) bool
}

func CheckLimit(users RequestLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID := r.URL.Query().Get("userid")

		if users.Allow(userID) {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}
}
