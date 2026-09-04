import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics to track during 10,000 RPS benchmark
export const orderSuccessRate = new Rate('order_success_rate');
export const productLatency = new Trend('product_fetch_latency');

export const options = {
  scenarios: {
    high_load_ramp: {
      executor: 'ramping-arrival-rate',
      startRate: 500,
      timeUnit: '1s',
      preAllocatedVUs: 500,
      maxVUs: 3000,
      stages: [
        { target: 1000, duration: '10s' },  // Warmup & cache prefill
        { target: 5000, duration: '15s' },  // Mid-scale test
        { target: 10000, duration: '30s' }, // Target 10,000 RPS stress phase!
        { target: 1000, duration: '10s' },  // Ramp-down
      ],
    },
  },
  thresholds: {
    // 95% of requests must complete within 35ms (thanks to L1/L2 cache + async Kafka)
    http_req_duration: ['p(95)<35', 'p(99)<100'],
    // Less than 0.5% failure rate under peak 10k RPS
    http_req_failed: ['rate<0.005'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const CATEGORIES = ['laptops', 'smartphones', 'audio', 'monitors', 'accessories'];

export default function () {
  const rand = Math.random();

  // 1. 80% of traffic: High-speed cached product detail (PDP)
  if (rand < 0.80) {
    const productId = Math.floor(Math.random() * 500) + 1;
    const res = http.get(`${BASE_URL}/api/v1/products/${productId}`, {
      tags: { name: 'GetProductByID' },
    });
    check(res, {
      'product status 200': (r) => r.status === 200,
    });
    productLatency.add(res.timings.duration);
  } 
  // 2. 10% of traffic: Catalog browsing by category
  else if (rand < 0.90) {
    const cat = CATEGORIES[Math.floor(Math.random() * CATEGORIES.length)];
    const res = http.get(`${BASE_URL}/api/v1/products?category=${cat}&limit=20`, {
      tags: { name: 'ListProductsByCategory' },
    });
    check(res, {
      'catalog status 200': (r) => r.status === 200,
    });
  } 
  // 3. 10% of traffic: High-concurrency Order placement (Write Path -> Kafka -> 202 Accepted)
  else {
    const payload = JSON.stringify({
      user_id: `user-${Math.floor(Math.random() * 10000)}`,
      items: [
        {
          product_id: Math.floor(Math.random() * 500) + 1,
          quantity: 1,
        },
      ],
    });

    const params = {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'CreateOrderAsync' },
    };

    const res = http.post(`${BASE_URL}/api/v1/orders`, payload, params);
    const success = check(res, {
      'order accepted 202': (r) => r.status === 202,
    });
    orderSuccessRate.add(success);
  }
}
