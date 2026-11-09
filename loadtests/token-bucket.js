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
      executor: 'constant-vus',
      vus: users,
      duration: __ENV.DURATION || '2m',
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
  const userID = `${data.runID}-vu-${exec.vu.idInTest}`;
  const response = http.get(`${baseURL}/check?userid=${encodeURIComponent(userID)}`, {
    timeout: '2s',
    redirects: 0,
    tags: { name: 'GET /check', phase: 'load' },
  });

  allowedRequests.add(response.status === 200 ? 1 : 0);
  throttledRequests.add(response.status === 429 ? 1 : 0);
  unexpectedResponses.add(response.status !== 200 && response.status !== 429 ? 1 : 0);
  check(response, {
    'allowed or throttled': (r) => r.status === 200 || r.status === 429,
  }, { phase: 'load' });

  // Start-to-start pacing; slow requests reduce throughput in this closed loop.
  sleep(Math.max(0, intervalSeconds - (Date.now() - startedAt) / 1000));
}
