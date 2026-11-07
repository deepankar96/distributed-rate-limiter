.PHONY: run run-loadgenerator run-ratelimiter test test-race vet

run:
	$(MAKE) -j2 run-loadgenerator run-ratelimiter

run-loadgenerator:
	go run ./cmd/loadgenerator

run-ratelimiter:
	go run ./cmd/ratelimiter

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...
