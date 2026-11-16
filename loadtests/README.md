# Token-bucket load test

Run k6 on the host against the already-running rate limiter. Test only targets you own or have permission to load test. This directory adds no services and does not configure Docker resource limits.

From the repository root, with k6 installed and the service reachable:

```sh
k6 run loadtests/token-bucket.js
```

The default ramps from **0 to 1,000 virtual users over 10 seconds**, holds 1,000 VUs for **2 measured minutes**, then allows **5 seconds** for iterations to finish. Each request has a **2-second timeout**. Stop early with Ctrl-C if needed. `DURATION` controls the hold period, not the ramp-up.

Each VU uses its own stable `userid`, prefixed by a fresh run ID generated once in `setup()`. Setup calls `/healthz` and requires HTTP 200; it does not consume quota. Load requests call `/check?userid=...`.

## Configuration

Set environment variables before `k6 run`:

| Variable | Default | Meaning |
| --- | --- | --- |
| `BASE_URL` | `http://localhost:3000` | Service origin, or base path without a final slash |
| `USERS` | `1000` | Positive safe integer; target concurrent VUs after ramp-up |
| `DURATION` | `2m` | Positive measured hold duration, such as `5s`, `2m`, or `1m30s`; excludes the fixed 10-second ramp-up |
| `REQUEST_INTERVAL_SECONDS` | `1` | Positive finite number; target start-to-start interval per VU |
| `RUN_ID` | Generated timestamp/random prefix | Optional reproducible prefix; reuse also reuses that run's bucket IDs |
| `EXPECT_THROTTLING` | `true` | Exactly `true` or `false`; whether to require at least one HTTP 429 |

Quick smoke test (too short to exhaust fresh buckets):

```sh
USERS=10 DURATION=5s EXPECT_THROTTLING=false k6 run loadtests/token-bucket.js
```

The final CLI summary shows **only measured hold traffic**: request count, request rate over the configured hold duration, allowed/throttled/unexpected counts, HTTP failure rate, status checks, latency, and threshold outcomes. No dashboard or HTML report is needed.

k6's built-in live progress and iteration counters still include ramp-up. Optional raw metric exports and dashboards also retain ramp-up traffic; the custom final summary is the filtered view. On interrupted runs, the request-rate denominator remains the configured hold duration, so it is not the achieved rate for the partial run.

## Load model and expected behavior

This is a **closed-loop** `ramping-vus` test with a constant-VU hold stage, not a guaranteed arrival-rate/RPS test. Each VU performs one synchronous request, then sleeps for the remainder of its configured interval. With fast responses, defaults target roughly 1,000 requests/second. Slow responses or an overloaded generator reduce that rate; there is no catch-up burst to recover missed requests.

Ramp-up requests consume tokens and create user buckets normally; excluding their metrics does not reset service state. VUs begin at different times during the ramp, so they enter the measured hold with different remaining allowances. A ramp-up is not a guarantee that every user has already sent a request or established a connection before measurement.

The current service starts each new user's bucket with 30 tokens and refills at 0.5 tokens/second, lazily on checks. At roughly one request/second per user, the initial allowance drains over about a minute. Afterwards, expect roughly 0.5 allowed requests/user/second, with the rest throttled. Timing, scheduling, and test boundaries affect totals; do not assert exact counts from these approximations. A short run, slower request cadence, or slow server may never produce a 429 even if the limiter is correct.

Fresh run IDs avoid starting with previously depleted buckets. However, user state currently remains in the service's in-memory map, so repeated runs grow that map. Restart the service between comparable tests to reset state and avoid accumulated-memory effects. A fixed `RUN_ID` deliberately gives up fresh buckets.

## Results and acceptance thresholds

Measurement is based on **request start time relative to the scenario start**: requests starting during the first 10 seconds are excluded, even if they finish during the hold. Requests starting during the hold are included if they finish during the 5-second graceful stop. There is no explicit ramp-down stage.

- `allowed_requests`: count of HTTP 200 measured responses.
- `throttled_requests`: count of HTTP 429 measured responses.
- `unexpected_responses`: count of other measured outcomes, including transport failures (status 0).
- Request rate: total measured responses divided by the configured hold duration in seconds, not by ramp-up plus hold.
- `http_req_failed{phase:load}`: transport/unexpected HTTP failure rate. HTTP 200 **and 429** are classified as expected responses, so legitimate throttling is not an HTTP failure.
- `checks{phase:load}`: each load response must be either 200 or 429.

The script exits unsuccessfully if any configured threshold fails:

| Threshold | Purpose |
| --- | --- |
| Measured HTTP duration p95 < 200 ms | Initial performance hypothesis, not a production guarantee |
| Measured HTTP failure rate < 1% | Separate transport/server errors from expected denials |
| Measured status checks = 100% | Reject any unexpected response, even below the failure-rate budget |
| Allowed count > 0 | Prevent an always-429 service from passing |
| Throttled count > 0, unless disabled | Confirm that this workload observed a denial |

Adjust the 200 ms hypothesis in the script to match the experiment and measured baseline. `http_req_duration` is HTTP request duration, not the full paced iteration or all connection-establishment time. Seeing both decisions is a smoke/behavior check, **not proof** of token arithmetic, per-user isolation, concurrency correctness, or architecture scalability; use the service's targeted tests for those properties.

Requests use constant `name` tags; `url` is excluded from `systemTags` so user IDs do not create high-cardinality metric series. Requests have `phase` tags (`setup`, `ramp`, `load`, or `unmeasured`); HTTP and check thresholds select only `phase:load`. Custom decision counters are recorded only for measured requests. Ramp-up errors are excluded from the reported thresholds too, while a failed setup health check still aborts the run.

## Resource-limited service experiments

If you run the service container with a **2 CPU / 256 MiB** budget, that budget applies to the service, not the host-side k6 process. This script neither sets nor verifies those limits; check your Docker configuration and runtime limits separately. Observe container CPU/memory alongside k6 throughput, errors, and latency (for example, using `docker stats`).

On macOS, Docker runs inside a VM. Check the VM's available CPU/memory too. Running 1,000 k6 VUs can itself consume substantial host resources and compete with Docker; generator saturation can look like server saturation. Monitor the host, close unrelated workloads, and record the machine, VM allocation, container limits, test settings, and service restart state when comparing runs. Start with the smoke test before the default load.
