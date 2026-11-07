.PHONY:run run-gateway run-ratelimiter

run:
	$(MAKE) -j2 run-gateway run-ratelimiter

run-gateway:
	go run ./cmd/gateway

run-ratelimiter:
	go run ./cmd/ratelimiter
