.PHONY: run down run-ratelimiter redis-up test test-race vet docker-up docker-down docker-stats load-test load-test-smoke

run: run-ratelimiter

run-ratelimiter: redis-up
	go run ./cmd/ratelimiter

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

redis-up:
	docker compose -f ../compose.yaml up -d --wait redis

docker-up:
	docker compose -f ../compose.yaml up --build -d

down: docker-down

docker-down:
	docker compose -f ../compose.yaml down

docker-stats:
	docker compose -f ../compose.yaml stats

load-test:
	k6 run ../loadtests/token-bucket.js

load-test-smoke:
	USERS=10 DURATION=5s EXPECT_THROTTLING=false k6 run ../loadtests/token-bucket.js
