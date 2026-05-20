import { test, expect } from '@playwright/test';

const apiUrl = process.env.E2E_API_URL ?? 'http://127.0.0.1:8000';

test.describe('backend API', () => {
	test('GET /health returns ok', async ({ request }) => {
		const res = await request.get(`${apiUrl}/health`);
		expect(res.status()).toBe(200);
		expect(await res.json()).toMatchObject({ status: 'ok' });
	});

	test('GET /api/dashboard returns net worth shape', async ({ request }) => {
		const res = await request.get(`${apiUrl}/api/dashboard`);
		expect(res.status()).toBe(200);
		const body = await res.json();
		expect(body).toHaveProperty('current_net_worth');
		expect(body).toHaveProperty('total_assets');
		expect(body).toHaveProperty('total_liabilities');
		expect(Array.isArray(body.net_worth_history)).toBe(true);
		expect(Array.isArray(body.allocation)).toBe(true);
	});

	test('GET /api/personas returns seeded personas', async ({ request }) => {
		const res = await request.get(`${apiUrl}/api/personas`);
		expect(res.status()).toBe(200);
		const personas = await res.json();
		expect(Array.isArray(personas)).toBe(true);
		expect(personas.length).toBeGreaterThan(0);
		const names = personas.map((p: { name: string }) => p.name);
		expect(names).toContain('Marcin');
	});

	test('GET /api/accounts returns asset and liability arrays', async ({ request }) => {
		const res = await request.get(`${apiUrl}/api/accounts`);
		expect(res.status()).toBe(200);
		const body = await res.json();
		expect(Array.isArray(body.assets)).toBe(true);
		expect(Array.isArray(body.liabilities)).toBe(true);
	});
});
