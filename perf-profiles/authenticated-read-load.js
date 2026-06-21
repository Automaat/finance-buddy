import http from 'k6/http';
import { check, group } from 'k6';
import { Trend } from 'k6/metrics';

const BASE_URL = __ENV.BB_BASE_URL || 'http://127.0.0.1:18080';
const USERNAME = __ENV.BB_USERNAME || 'admin';
const PASSWORD = __ENV.BB_PASSWORD || 'admin';

const endpoints = [
	'/health',
	'/api/auth/me',
	'/api/dashboard',
	'/api/accounts',
	'/api/assets',
	'/api/snapshots',
	'/api/users',
	'/api/config',
	'/api/goals',
	'/api/transactions',
	'/api/transactions/counts',
	'/api/payments',
	'/api/payments/counts',
	'/api/debts',
	'/api/salaries',
	'/api/bonuses',
	'/api/equity-grants',
	'/api/company-valuations',
	'/api/holdings',
	'/api/holdings/securities',
	'/api/holdings/lots',
	'/api/holdings/dividends',
	'/api/bonds',
	'/api/bonds/maturity-ladder',
	'/api/investment/stock-stats',
	'/api/investment/bond-stats',
	'/api/investment/returns',
	'/api/retirement/stats',
	'/api/retirement/ppk-stats',
	'/api/cpi/series',
	'/api/zus/prefill',
	'/api/simulations/prefill',
	'/api/exposure/currency',
	'/api/pit38/realized',
	'/api/allocation/targets',
	'/api/allocation/drift'
];

function metricName(path) {
	return 'latency_' + path.replace(/[^a-zA-Z0-9]/g, '_').replace(/^_+|_+$/g, '');
}

const trends = Object.fromEntries(endpoints.map((path) => [path, new Trend(metricName(path))]));

export const options = {
	scenarios: {
		reads: {
			executor: 'constant-vus',
			vus: Number(__ENV.VUS || 10),
			duration: __ENV.DURATION || '30s',
			gracefulStop: '5s'
		}
	},
	thresholds: {
		checks: ['rate>0.99']
	},
	summaryTrendStats: ['avg', 'min', 'med', 'p(95)', 'p(99)', 'max']
};

export function setup() {
	const response = http.post(
		`${BASE_URL}/api/auth/login`,
		JSON.stringify({ username: USERNAME, password: PASSWORD, remember_me: true }),
		{ headers: { 'Content-Type': 'application/json' } }
	);
	check(response, {
		'login: 200': (r) => r.status === 200,
		'login: token present': (r) => Boolean(r.json('token'))
	});
	return { token: response.json('token') };
}

function hit(path, token) {
	const params = path === '/health' ? {} : { headers: { Authorization: `Bearer ${token}` } };
	const response = http.get(`${BASE_URL}${path}`, {
		...params,
		tags: { endpoint: path }
	});
	trends[path].add(response.timings.duration);
	check(response, {
		[`${path}: 2xx`]: (r) => r.status >= 200 && r.status < 300
	});
}

export default function (data) {
	for (const path of endpoints) {
		group(path, () => hit(path, data.token));
	}
}
