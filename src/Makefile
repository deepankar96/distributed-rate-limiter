.PHONY: run run-ratelimiter test test-race vet docker-up docker-down docker-stats load-test load-test-smoke

run: run-ratelimiter

run-ratelimiter:
	go run ./cmd/ratelimiter

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

docker-up:
	docker compose -f ../compose.yaml up --build -d

docker-down:
	docker compose -f ../compose.yaml down

docker-stats:
	docker compose -f ../compose.yaml stats

load-test:
	k6 run ../loadtests/token-bucket.js

load-test-smoke:
	USERS=10 DURATION=5s EXPECT_THROTTLING=false k6 run ../loadtests/token-bucket.js
