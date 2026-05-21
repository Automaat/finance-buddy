// k6 baseline for the Python FastAPI backend.
//
// Goal: capture a per-endpoint p50/p95/p99 reference against a seeded DB so we
// can detect regressions after the Go cutover.
//
// Light load (10 VUs × 30 s) — this is a baseline, not a stress test.
//
// One iteration walks the full endpoint list, so each VU spreads its time
// across every route. Per-iteration timing therefore reflects the slowest
// endpoint (currently /api/dashboard); the per-route Trend metrics below are
// what you read for actual latencies.
//
// Each request is tagged with `endpoint` so the JSON summary can be sliced
// per route; see migration/perf/baseline.json after a run.

import http from 'k6/http';
import { check, group } from 'k6';
import { Trend } from 'k6/metrics';

const BASE_URL = __ENV.BB_BASE_URL || 'http://127.0.0.1:8000';

// Per-endpoint trends so the summary captures p50/p95/p99 for each route.
// Metric names must be [A-Za-z0-9_], so we slug the path.
const endpoints = [
	'/health',
	'/api/dashboard',
	'/api/accounts',
	'/api/assets',
	'/api/snapshots',
	'/api/personas',
	'/api/config',
	'/api/goals',
	'/api/transactions',
	'/api/payments',
	'/api/debts',
	'/api/salaries',
	'/api/bonuses',
	'/api/equity-grants',
	'/api/company-valuations',
	'/api/investment/stock-stats',
	'/api/investment/bond-stats',
	'/api/retirement/stats',
	'/api/retirement/ppk-stats',
	'/api/cpi/series',
	'/api/zus/prefill',
	'/api/simulations/prefill'
];

function metricName(path) {
	return 'latency_' + path.replace(/[^a-zA-Z0-9]/g, '_').replace(/^_+|_+$/g, '');
}

const trends = Object.fromEntries(endpoints.map((path) => [path, new Trend(metricName(path))]));

export const options = {
	scenarios: {
		reads: {
			executor: 'constant-vus',
			vus: 10,
			duration: '30s',
			gracefulStop: '5s'
		}
	},
	thresholds: {
		// The only hard gate: every per-endpoint 2xx check must pass. If a
		// request errors or returns 4xx/5xx, the seed is wrong or the backend
		// isn't healthy, and the captured numbers are meaningless. Using the
		// `checks` metric (rather than http_req_failed) gives a clearer
		// passes/fails breakdown in the exported JSON summary.
		// Latency thresholds intentionally omitted — this is a *baseline*
		// capture, not a CI gate. Compare new runs against baseline.json.
		checks: ['rate>0.99']
	},
	summaryTrendStats: ['avg', 'min', 'med', 'p(95)', 'p(99)', 'max']
};

function hit(path) {
	const response = http.get(`${BASE_URL}${path}`, {
		tags: { endpoint: path }
	});
	trends[path].add(response.timings.duration);
	check(response, {
		[`${path}: 2xx`]: (r) => r.status >= 200 && r.status < 300
	});
}

export default function () {
	for (const path of endpoints) {
		group(path, () => hit(path));
	}
}
