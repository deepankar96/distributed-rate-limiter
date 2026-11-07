package main

import (
	"log"
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/gateway"
)

func main() {
	server := &http.Server{
		Addr:    ":8080",
		Handler: gateway.NewHandler(),
	}

	log.Println("Gateway started on port 8080")
	log.Fatal(server.ListenAndServe())
}
