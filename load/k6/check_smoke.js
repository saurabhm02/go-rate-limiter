import http from 'k6/http';
import { check } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
const apiKey = __ENV.API_KEY || 'rl_demo_abc123xyz';

export const options = {
  vus: 5,
  duration: '10s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(99)<500'],
    checks: ['rate>0.99'],
  },
};

export default function () {
  const res = http.post(
    `${baseURL}/v1/check`,
    JSON.stringify({ route: '/v1/check', cost: 1 }),
    {
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': apiKey,
      },
    },
  );

  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'has rate limit headers when limited': (r) =>
      r.status === 200 || r.headers['Retry-After'] !== undefined || r.headers['X-Ratelimit-Limit'] !== undefined,
  });
}
