import http from 'k6/http';
import { check, fail, sleep } from 'k6';
import exec from 'k6/execution';
import { Counter } from 'k6/metrics';

function positiveNumber(name, fallback, integer = false) {
  const value = Number(__ENV[name] === undefined ? fallback : __ENV[name]);
  if (!Number.isFinite(value) || value <= 0 || (integer && !Number.isSafeInteger(value))) {
    throw new Error(`${name} must be a positive ${integer ? 'safe integer' : 'number'}`);
  }
  return value;
}

function durationMilliseconds(value) {
  const units = { h: 3600000, m: 60000, s: 1000, ms: 1, us: 0.001, 'µs': 0.001, 'μs': 0.001, ns: 0.000001 };
  const parts = value.match(/(?:\d+(?:\.\d*)?|\.\d+)(?:ns|us|µs|μs|ms|s|m|h)/g) || [];
  const milliseconds = parts.reduce((total, part) => {
    const unit = part.match(/[a-zµμ]+$/)[0];
    return total + parseFloat(part) * units[unit];
  }, 0);
  if (parts.join('') !== value || !Number.isFinite(milliseconds) || milliseconds <= 0) {
    throw new Error('DURATION must be a positive k6 duration, such as 5s, 2m, or 1m30s');
  }
  return milliseconds;
}

const rampUpMilliseconds = 10000;
const duration = __ENV.DURATION || '2m';
const measuredMilliseconds = durationMilliseconds(duration);
const baseURL = (__ENV.BASE_URL || 'http://localhost:3000').replace(/\/+$/, '');
const users = positiveNumber('USERS', 1000, true);
const intervalSeconds = positiveNumber('REQUEST_INTERVAL_SECONDS', 1);
const expectThrottling = __ENV.EXPECT_THROTTLING === undefined ? 'true' : __ENV.EXPECT_THROTTLING;
if (expectThrottling !== 'true' && expectThrottling !== 'false') {
  throw new Error('EXPECT_THROTTLING must be true or false');
}

const allowedRequests = new Counter('allowed_requests');
const throttledRequests = new Counter('throttled_requests');
const unexpectedResponses = new Counter('unexpected_responses');

http.setResponseCallback(http.expectedStatuses(200, 429));

export const options = {
  scenarios: {
    users: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: users },
        { duration, target: users },
      ],
      gracefulStop: '5s',
    },
  },
  discardResponseBodies: true,
  // Exclude URL: each virtual user has a distinct query parameter.
  systemTags: ['status', 'method', 'name', 'scenario', 'expected_response', 'error_code'],
  thresholds: {
    'http_req_duration{phase:load}': ['p(95)<200'],
    'http_req_failed{phase:load}': ['rate<0.01'],
    'checks{phase:load}': ['rate==1'],
    allowed_requests: ['count>0'],
    ...(expectThrottling === 'true' ? { throttled_requests: ['count>0'] } : {}),
  },
};

export function setup() {
  const response = http.get(`${baseURL}/healthz`, {
    timeout: '2s',
    redirects: 0,
    tags: { name: 'GET /healthz', phase: 'setup' },
  });
  if (response.status !== 200) {
    fail(`Health check failed: expected HTTP 200, got ${response.status}; ${response.error || 'check BASE_URL and service logs'}`);
  }

  // setup runs once; all VUs share this prefix, not the same user ID.
  return {
    runID: __ENV.RUN_ID || `k6-${Date.now()}-${Math.random().toString(36).slice(2)}`,
  };
}

export default function (data) {
  const startedAt = Date.now();
  const elapsed = startedAt - exec.scenario.startTime;
  // Tag by request start: a ramp-up request finishing during the hold stays excluded.
  let phase = 'unmeasured';
  if (elapsed < rampUpMilliseconds) {
    phase = 'ramp';
  } else if (elapsed < rampUpMilliseconds + measuredMilliseconds) {
    phase = 'load';
  }
  const userID = `${data.runID}-vu-${exec.vu.idInTest}`;
  const response = http.get(`${baseURL}/check?userid=${encodeURIComponent(userID)}`, {
    timeout: '2s',
    redirects: 0,
    tags: { name: 'GET /check', phase },
  });

  if (phase === 'load') {
    allowedRequests.add(response.status === 200 ? 1 : 0);
    throttledRequests.add(response.status === 429 ? 1 : 0);
    unexpectedResponses.add(response.status !== 200 && response.status !== 429 ? 1 : 0);
  }
  check(response, {
    'allowed or throttled': (r) => r.status === 200 || r.status === 429,
  }, { phase });

  // Start-to-start pacing; slow requests reduce throughput in this closed loop.
  sleep(Math.max(0, intervalSeconds - (Date.now() - startedAt) / 1000));
}

export function handleSummary(data) {
  const values = (name) => data.metrics[name] ? data.metrics[name].values : {};
  const count = (name) => values(name).count || 0;
  const milliseconds = (value) => value === undefined ? 'n/a' : `${value.toFixed(2)} ms`;
  const percentage = (value) => value === undefined ? 'n/a' : `${(value * 100).toFixed(2)}%`;
  const allowed = count('allowed_requests');
  const throttled = count('throttled_requests');
  const unexpected = count('unexpected_responses');
  const requests = allowed + throttled + unexpected;
  const latency = values('http_req_duration{phase:load}');
  const lines = [
    '',
    'STEADY-LOAD RESULTS (10s ramp-up and health check excluded)',
    `Target: ${users} VUs for ${duration}`,
    `Requests: ${requests}`,
    `Request rate over configured hold: ${(requests / (measuredMilliseconds / 1000)).toFixed(2)}/s`,
    `Allowed: ${allowed} | Throttled: ${throttled} | Unexpected: ${unexpected}`,
    `HTTP failure rate: ${percentage(values('http_req_failed{phase:load}').rate)}`,
    `Status checks passed: ${percentage(values('checks{phase:load}').rate)}`,
    `Latency: avg=${milliseconds(latency.avg)} | median=${milliseconds(latency.med)} | p95=${milliseconds(latency['p(95)'])} | max=${milliseconds(latency.max)}`,
    '',
    'THRESHOLDS (steady load only)',
  ];
  if (requests === 0) {
    lines.push('No measured requests were recorded. Check setup errors or early interruption.');
  }
  for (const [name, metric] of Object.entries(data.metrics)) {
    for (const [expression, result] of Object.entries(metric.thresholds || {})) {
      lines.push(`${result.ok ? 'PASS' : 'FAIL'} ${name}: ${expression}`);
    }
  }
  return { stdout: `${lines.join('\n')}\n` };
}
