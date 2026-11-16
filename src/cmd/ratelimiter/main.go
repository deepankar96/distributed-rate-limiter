package main

import (
	"context"
	"log"
	"net/http"

	"github.com/deepankar96/distributed-rate-limiter/internal/config"
	"github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter"
	redislimiter "github.com/deepankar96/distributed-rate-limiter/internal/ratelimiter/redisLimiter"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("load configuration: ", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:                  cfg.Redis.Address,
		Username:              cfg.Redis.Username,
		Password:              cfg.Redis.Password,
		DB:                    cfg.Redis.DB,
		ContextTimeoutEnabled: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Redis.Timeout)
	err = client.Ping(ctx).Err()
	cancel()

	if err != nil {
		log.Fatal("Redis startup check failed ", err)
	}

	users, err := redislimiter.NewLimiter(client, 30, 0.5)
	if err != nil {
		log.Fatal("Initialized Redis Limiter ", err)
	}

	server := &http.Server{
		Addr:         cfg.HTTP.Address,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		Handler:      ratelimiter.NewHandler(users),
	}

	log.Println("Starting Rate Limiter on port ", cfg.HTTP.Address)
	log.Fatal(server.ListenAndServe())
}
