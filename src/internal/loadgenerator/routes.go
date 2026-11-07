package loadgenerator

import (
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/loadgenerator/handlers"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handlers.Health)

	return mux
}
