# Rate limiter service guide

Run the commands below from this `src` directory. The service uses Go 1.24.2 or later and only the standard library. It is the only application currently in the project.

## Run

```sh
make run
```

`make run-ratelimiter` is an equivalent command. The server listens on port 3000. Stop it with Ctrl-C; restart it after changing source or configuration.

## Endpoints

```sh
curl -i 'http://localhost:3000/healthz'
curl -i 'http://localhost:3000/check?userid=1'
```

| Endpoint | Behavior |
| --- | --- |
| `GET /healthz` | Returns HTTP 200 without consuming allowance |
| `GET /check?userid=1` | Checks and consumes that user's allowance; returns HTTP 200 if allowed or HTTP 429 if denied |

The check endpoint currently returns a status code without a response body or remaining-token headers. User IDs are caller-provided, not authenticated, and currently not validated; use an explicit `userid` in requests.

## Current limiter policy

`internal/ratelimiter/routes.go` currently creates the token-bucket implementation with:

```go
tokenbucketlimiter.New(0.5, 30)
```

The constructor takes **refill rate first, capacity second**:

- Each new user starts with 30 tokens.
- Each allowed request consumes one token.
- The bucket refills at 0.5 tokens per second, up to capacity.
- Refilling is calculated on each check; no background refill goroutine runs.
- Different user IDs have independent buckets.
- State is in memory, is not shared across instances, and resets on restart. User entries are not currently removed.


## Structure

- `cmd/ratelimiter/main.go`: starts the server.
- `internal/ratelimiter/routes.go`: creates the selected implementation and registers routes.
- `internal/ratelimiter/handlers/`: HTTP input/output and the consumer-defined `RequestLimiter` interface.

- `internal/ratelimiter/tokenBucketLimiter/`: token-bucket implementation.

The token-bucket collection provides `Allow(userID string) bool`. The handler depends on the `RequestLimiter` interface rather than on the algorithm's concrete type, so another implementation can be added later without changing the handler.

## Tests and static checks

```sh
make test
make test-race
make vet
```

Equivalent Go commands:

```sh
go test ./...
go test -race ./...
go vet ./...
```

Concurrency tests are part of the normal suite; no build tag is required. Tests use in-process handlers and do not require the service to be running.

For repeated concurrency checks:

```sh
go test -race -run Concurrent -count=10 ./internal/ratelimiter/...
```

The tests cover the token-bucket algorithm, the handler interface, route wiring, independent user state, concurrent lookup/creation, and concurrent consumption. Refill arithmetic is tested with controlled timestamps; exact allowance tests use zero refill where needed to avoid wall-clock-sensitive expectations. No sleeps are required.

## Load testing with k6

Use k6 externally against the running service. The script at `../loadtests/token-bucket.js` defaults to 1,000 distinct users, each attempting approximately one request per second for two minutes. k6 is a separate tool and is not required for Go tests.

From this `src` directory:

```sh
make docker-up
curl -i 'http://localhost:3000/healthz'
make load-test-smoke
make load-test
```

Stop any existing host process using port 3000 before starting Docker. `make docker-up` builds a non-root, static service image and starts it in the background using `../compose.yaml`.

The initial container budget is **2 CPUs and 256 MiB RAM**. `GOMAXPROCS=2` makes Go's CPU parallelism explicit, and `GOMEMLIMIT=192MiB` leaves headroom below the hard container memory limit. Go's memory limit is soft and does not cover all process memory; it does not guarantee protection from an OOM kill. The builder image is pinned to the project's Go 1.24.2 version; review and update the toolchain image as the project evolves.

Inspect resource usage with Docker Desktop or `make docker-stats` (Ctrl-C exits the live view). You can also verify the configured hard limits from the repository root:

```sh
docker inspect --format '{{.HostConfig.NanoCpus}} {{.HostConfig.Memory}} {{.HostConfig.MemorySwap}}' distributed-rate-limiter-ratelimiter-1
```

With the default Compose project name, expected values are `2000000000 268435456 268435456`. A custom project name changes the container name. Memory+swap is set equal to memory to disable extra swap allowance where supported.

Use `make docker-down` to stop and remove the container. Stopping or restarting the service discards its in-memory buckets. To reset before a comparable test, run `docker compose -f ../compose.yaml restart ratelimiter`, then check `/healthz` before starting k6.

See [the load-test guide](../loadtests/README.md) for parameters, the local web dashboard, acceptance thresholds, and interpretation. The test generator runs on the host, outside the service's resource budget.

When designing a k6 test:

- Use the same user ID to exercise contention on one bucket, or multiple IDs to exercise independent buckets and the shared user map.
- Account for the initial 30-token burst and the refill rate; the expected result depends on request timing, not just total request count.
- Treat HTTP 429 as an expected rate-limit decision, separate from connection errors or server failures. Configure k6's response classification and assertions accordingly rather than treating every non-200 response as a limiter bug.
- Restart the service when a test needs fresh in-memory state, or use a fresh set of user IDs.
- Generate traffic only against targets you own or have permission to test.

## Synchronization to study

- The collection mutex protects the entire map lookup-or-create operation.
- Each bucket mutex protects its refill, check, and state update.
- The collection lock is released before the individual bucket is checked, so different users do not share a bucket lock.
- The token bucket captures current time after acquiring its lock.
- Pointer receivers avoid copying structs containing mutexes. Do not copy a mutex after it has been used.
- Constructors initialize map fields; mutex fields need no explicit initialization.

The concurrency tests aggregate results through channels after workers finish. Race detection covers only execution paths exercised during a run; a passing run is not proof of race freedom for every possible execution.
