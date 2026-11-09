# Rate limiter learning project

A Go learning project focused on a standalone HTTP rate limiter service.

The current service uses a per-user token bucket with a capacity of 30 tokens and a refill rate of 0.5 tokens per second.

The learning focus is Go structs, pointers, interfaces, HTTP handlers, concurrency, and testing. Load testing will use **k6 as an external tool**, not a second application in this repository.

## Quick start

From the repository root:

```sh
make -C src run
```

In another terminal:

```sh
curl -i 'http://localhost:3000/healthz'
curl -i 'http://localhost:3000/check?userid=1'
```

## Docker and a 1,000-user load test

From the repository root, with Docker Compose and k6 installed:

```sh
docker compose up --build -d
curl -i 'http://localhost:3000/healthz'
USERS=10 DURATION=5s EXPECT_THROTTLING=false k6 run loadtests/token-bucket.js
k6 run loadtests/token-bucket.js
```

Stop any locally running service on port 3000 first. The container is limited to **2 CPUs and 256 MiB RAM**, with `GOMAXPROCS=2` and a **192 MiB soft Go memory limit**. Resource limits are in `compose.yaml`; a Dockerfile alone does not enforce them. These are initial experiment settings, not measured capacity requirements.

The default load test runs **1,000 distinct user IDs for 2 minutes**, targeting about one request per second per user. It measures allowed and throttled responses separately; legitimate HTTP 429 decisions are not transport errors. Actual request rate can fall if the server or local generator is overloaded. See [the load-test guide](loadtests/README.md) for thresholds, configuration, and limitations.

Watch resource usage in Docker Desktop or run `docker compose stats` in another terminal. Stop the container with `docker compose down`.

To run tests with race detection:

```sh
make -C src test-race
```

See [the service guide](src/README.md) for endpoints, implementation details, test commands, and load-testing considerations.

Despite the repository name, state is currently in memory and local to one process. This is a learning implementation, not a production-ready distributed limiter.
