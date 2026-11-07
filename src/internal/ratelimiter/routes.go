package ratelimiter

import (
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/handlers"
	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/limiter"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	users := limiter.NewUserLimiters()

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /check", handlers.CheckLimit(users))
	return mux
}
