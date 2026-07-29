import http from 'k6/http';
import { check } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
const apiKey = __ENV.API_KEY || 'rl_demo_abc123xyz';
const route = __ENV.ROUTE || '/v1/check';

// Target: sustain 1000 checks/sec for 30s against POST /v1/check.
// Interpret results with: k6 handles status 200 (allowed) and 429 (denied) as success.
export const options = {
  scenarios: {
    check_sustained: {
      executor: 'constant-arrival-rate',
      rate: Number(__ENV.RATE || 1000),
      timeUnit: '1s',
      duration: __ENV.DURATION || '30s',
      preAllocatedVUs: 100,
      maxVUs: 500,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    // Stretch goal p99 < 5ms on warm local stack; threshold kept practical for CI laptops.
    http_req_duration: ['p(95)<50', 'p(99)<100'],
    checks: ['rate>0.99'],
  },
};

export default function () {
  const res = http.post(
    `${baseURL}/v1/check`,
    JSON.stringify({ route, cost: 1 }),
    {
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': apiKey,
      },
      tags: { name: 'check' },
    },
  );

  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });
}
