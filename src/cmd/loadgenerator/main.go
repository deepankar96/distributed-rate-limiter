package main

import (
	"log"
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/loadgenerator"
)

func main() {
	server := &http.Server{
		Addr:    ":8080",
		Handler: loadgenerator.NewHandler(),
	}

	log.Println("Load generator health server started on port 8080")
	log.Fatal(server.ListenAndServe())
}
