# Rate limiter learning project

A Go learning project focused on a standalone HTTP rate limiter service.

The current service uses a per-user token bucket with a capacity of 30 tokens and a refill rate of 0.5 tokens per second. An earlier five-call counter implementation remains available for learning and comparison, but is not the implementation selected by the HTTP routes.

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

To run tests with race detection:

```sh
make -C src test-race
```

See [the service guide](src/README.md) for endpoints, implementation details, test commands, and load-testing considerations.

Despite the repository name, state is currently in memory and local to one process. This is a learning implementation, not a production-ready distributed limiter.
