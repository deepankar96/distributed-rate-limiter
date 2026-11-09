# Token-bucket load test

Run k6 on the host against the already-running rate limiter. Test only targets you own or have permission to load test. This directory adds no services and does not configure Docker resource limits.

From the repository root, with k6 installed and the service reachable:

```sh
k6 run loadtests/token-bucket.js
```

The default starts **1,000 virtual users immediately** (no ramp), runs for **2 minutes**, then allows **5 seconds** for iterations to finish. Each request has a **2-second timeout**. Stop early with Ctrl-C if needed.

Each VU uses its own stable `userid`, prefixed by a fresh run ID generated once in `setup()`. Setup calls `/healthz` and requires HTTP 200; it does not consume quota. Load requests call `/check?userid=...`.

## Configuration

Set environment variables before `k6 run`:

| Variable | Default | Meaning |
| --- | --- | --- |
| `BASE_URL` | `http://localhost:3000` | Service origin, or base path without a final slash |
| `USERS` | `1000` | Positive safe integer; constant concurrent VUs |
| `DURATION` | `2m` | Scenario duration, validated by k6 |
| `REQUEST_INTERVAL_SECONDS` | `1` | Positive finite number; target start-to-start interval per VU |
| `RUN_ID` | Generated timestamp/random prefix | Optional reproducible prefix; reuse also reuses that run's bucket IDs |
| `EXPECT_THROTTLING` | `true` | Exactly `true` or `false`; whether to require at least one HTTP 429 |

Quick smoke test (too short to exhaust fresh buckets):

```sh
USERS=10 DURATION=5s EXPECT_THROTTLING=false k6 run loadtests/token-bucket.js
```

Optional local live dashboard, on k6 versions supporting the web dashboard:

```sh
K6_WEB_DASHBOARD=true k6 run loadtests/token-bucket.js
```

Open the dashboard address printed by k6 (normally `http://localhost:5665`). No Grafana stack is needed for this test.

## Load model and expected behavior

This is a **closed-loop** `constant-vus` test, not a guaranteed arrival-rate/RPS test. Each VU performs one synchronous request, then sleeps for the remainder of its configured interval. With fast responses, defaults target roughly 1,000 requests/second. Slow responses or an overloaded generator reduce that rate; there is no catch-up burst to recover missed requests.

The current service starts each new user's bucket with 30 tokens and refills at 0.5 tokens/second, lazily on checks. At roughly one request/second per user, the initial allowance drains over about a minute. Afterwards, expect roughly 0.5 allowed requests/user/second, with the rest throttled. Timing, scheduling, and test boundaries affect totals; do not assert exact counts from these approximations. A short run, slower request cadence, or slow server may never produce a 429 even if the limiter is correct.

Fresh run IDs avoid starting with previously depleted buckets. However, user state currently remains in the service's in-memory map, so repeated runs grow that map. Restart the service between comparable tests to reset state and avoid accumulated-memory effects. A fixed `RUN_ID` deliberately gives up fresh buckets.

## Results and acceptance thresholds

- `allowed_requests`: count of HTTP 200 load responses.
- `throttled_requests`: count of HTTP 429 load responses.
- `unexpected_responses`: count of other load outcomes, including transport failures (status 0).
- `http_req_failed{phase:load}`: transport/unexpected HTTP failure rate. HTTP 200 **and 429** are classified as expected responses, so legitimate throttling is not an HTTP failure.
- `checks{phase:load}`: each load response must be either 200 or 429.

The script exits unsuccessfully if any configured threshold fails:

| Threshold | Purpose |
| --- | --- |
| Load HTTP duration p95 < 200 ms | Initial performance hypothesis, not a production guarantee |
| Load HTTP failure rate < 1% | Separate transport/server errors from expected denials |
| Load status checks = 100% | Reject any unexpected response, even below the failure-rate budget |
| Allowed count > 0 | Prevent an always-429 service from passing |
| Throttled count > 0, unless disabled | Confirm that this workload observed a denial |

Adjust the 200 ms hypothesis in the script to match the experiment and measured baseline. `http_req_duration` is HTTP request duration, not the full paced iteration or all connection-establishment time. Seeing both decisions is a smoke/behavior check, **not proof** of token arithmetic, per-user isolation, concurrency correctness, or architecture scalability; use the service's targeted tests for those properties.

Requests use constant `name` tags; `url` is excluded from `systemTags` so user IDs do not create high-cardinality metric series. Setup and load requests have separate `phase` tags; load HTTP thresholds exclude the health check.

## Resource-limited service experiments

If you run the service container with a **2 CPU / 256 MiB** budget, that budget applies to the service, not the host-side k6 process. This script neither sets nor verifies those limits; check your Docker configuration and runtime limits separately. Observe container CPU/memory alongside k6 throughput, errors, and latency (for example, using `docker stats`).

On macOS, Docker runs inside a VM. Check the VM's available CPU/memory too. Running 1,000 k6 VUs can itself consume substantial host resources and compete with Docker; generator saturation can look like server saturation. Monitor the host, close unrelated workloads, and record the machine, VM allocation, container limits, test settings, and service restart state when comparing runs. Start with the smoke test before the default load.
