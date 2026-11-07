# Running the learning project

Run commands from this `src` directory. The project uses Go 1.24.2 or later and only the standard library.

## Services

```sh
make run-ratelimiter
```

The rate limiter listens on port 3000:

```sh
curl -i 'http://localhost:3000/healthz'
curl -i 'http://localhost:3000/check?userid=1'
```

Each user gets five successful checks, then receives HTTP 429. The allowance does not reset with time; restarting the service resets all users. This is an in-memory learning example, not a distributed or time-based rate limiter. User IDs are caller-provided and are not authenticated.

```sh
make run-loadgenerator
```

The load generator currently serves **only a health endpoint** on port 8080. It does not generate requests:

```sh
curl -i 'http://localhost:8080/healthz'
```

Use `make run` to start both services together. Stop the foreground processes with Ctrl-C.

## Checks

```sh
make test
make test-race
make vet
```

Or run the Go commands directly:

```sh
go test ./...
go test -race ./...
go vet ./...
```

Concurrency tests now run as part of the normal suite; no build tag is required. They exercise the shared counter, concurrent user lookup/creation, and concurrent HTTP handler calls. Tests use in-process handlers and do not require running services.

For repeated concurrency checks:

```sh
go test -race -run Concurrent -count=10 ./internal/ratelimiter/...
```

## Synchronization to study

- `UserLimiters.mu` protects the map's entire lookup-or-create operation.
- Each `Limiter.mu` protects that user's check, decrement, and returned counter value.
- The collection lock is released before the individual limiter is checked, so different users do not share a counter lock.
- Pointer receivers avoid copying the structs containing mutexes. Do not copy a mutex after it has been used.
- Constructors initialize map fields; mutex fields need no explicit initialization.

The concurrency tests collect results through channels and inspect them after workers finish, so the tests do not introduce their own shared-counter races. Race detection covers only execution paths exercised during a run.
