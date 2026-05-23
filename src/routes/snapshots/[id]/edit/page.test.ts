import { describe, it, expect, vi, beforeEach } from 'vitest';

const browserState = { value: false };
const envState: { PUBLIC_API_URL?: string; PUBLIC_API_URL_BROWSER?: string } = {};

vi.mock('$app/environment', () => ({
	get browser() {
		return browserState.value;
	}
}));

vi.mock('$env/dynamic/public', () => ({
	get env() {
		return envState;
	}
}));

async function freshLoad() {
	vi.resetModules();
	return (await import('./+page')).load;
}

describe('snapshots/[id]/edit load', () => {
	beforeEach(() => {
		browserState.value = false;
		envState.PUBLIC_API_URL = 'http://backend-go:8000';
		envState.PUBLIC_API_URL_BROWSER = 'http://localhost:5174';
	});

	it('hits PUBLIC_API_URL during SSR (regression for #395)', async () => {
		const load = await freshLoad();
		const fetch = vi.fn(async () =>
			Response.json({ assets: [], liabilities: [] })
		) as unknown as typeof globalThis.fetch;
		await load({ params: { id: '7' }, fetch } as Parameters<typeof load>[0]);

		const urls = (fetch as unknown as { mock: { calls: [string][] } }).mock.calls.map((c) => c[0]);
		for (const u of urls) {
			expect(u).toMatch(/^http:\/\/backend-go:8000\//);
		}
		expect(urls).toContain('http://backend-go:8000/api/snapshots/7');
		expect(urls).toContain('http://backend-go:8000/api/accounts');
		expect(urls).toContain('http://backend-go:8000/api/assets');
		expect(urls).toContain('http://backend-go:8000/api/users');
	});
});
