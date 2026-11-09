.PHONY: run run-ratelimiter test test-race vet

run: run-ratelimiter

run-ratelimiter:
	go run ./cmd/ratelimiter

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...
