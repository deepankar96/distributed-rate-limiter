package ratelimiter

import (
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/handlers"
)

func NewHandler(users handlers.RequestLimiter) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /check", handlers.CheckLimit(users))
	return mux
}
