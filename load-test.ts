import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 20 },
    { duration: '30s', target: 50 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<50'],
  },
};

export default function () {
  const payload = JSON.stringify({
    original_url: 'https://en.wikipedia.org/wiki/Load_testing',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const resPost = http.post(`${__ENV.BASE_DOMAIN}/urls`, payload, params);

  check(resPost, {
    'created status is 200': (r) => r.status === 200,
    'has short_url': (r) => r.body.includes('short_url'),
  });

  if (resPost.status === 200) {
    const body = JSON.parse(resPost.body);
    const shortUrl = body.short_url;

    const resGet = http.get(shortUrl, { redirects: 0 });

    check(resGet, {
      'redirect status is 302': (r) => r.status === 302,
      'location header is correct': (r) => r.headers['Location'] === 'https://en.wikipedia.org/wiki/Load_testing',
    });
  }

  sleep(1);
}