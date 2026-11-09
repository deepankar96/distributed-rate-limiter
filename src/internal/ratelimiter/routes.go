package ratelimiter

import (
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/handlers"
	tokenbucketlimiter "github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/tokenBucketLimiter"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	users := tokenbucketlimiter.New(0.5, 30)

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /check", handlers.CheckLimit(users))
	return mux
}
