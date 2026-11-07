package main

import (
	"log"
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter"
)

func main() {
	// limit := 100
	// for i := 0; i < 101; i++ {
	// 	fmt.Println(limiter.CheckLimit(&limit))
	// }

	server := &http.Server{
		Addr:    ":3000",
		Handler: ratelimiter.NewHandler(),
	}

	log.Println("Starting Rate Limiter on port 3000")
	log.Fatal(server.ListenAndServe())
}
