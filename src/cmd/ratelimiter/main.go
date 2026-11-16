package main

import (
	"log"
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/config"
	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("load configuration: ", err)
	}

	server := &http.Server{
		Addr:         cfg.HTTP.Address,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		Handler:      ratelimiter.NewHandler(),
	}

	log.Println("Starting Rate Limiter on port 3000")
	log.Fatal(server.ListenAndServe())
}
