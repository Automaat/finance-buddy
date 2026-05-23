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

async function freshImport() {
	vi.resetModules();
	return await import('./api');
}

describe('resolveApiUrl', () => {
	beforeEach(() => {
		browserState.value = false;
		envState.PUBLIC_API_URL = undefined;
		envState.PUBLIC_API_URL_BROWSER = undefined;
	});

	it('returns PUBLIC_API_URL on the server', async () => {
		browserState.value = false;
		envState.PUBLIC_API_URL = 'http://backend-go:8000';
		envState.PUBLIC_API_URL_BROWSER = 'http://localhost:5174';
		const { resolveApiUrl } = await freshImport();
		expect(resolveApiUrl()).toBe('http://backend-go:8000');
	});

	it('returns PUBLIC_API_URL_BROWSER in the browser', async () => {
		browserState.value = true;
		envState.PUBLIC_API_URL = 'http://backend-go:8000';
		envState.PUBLIC_API_URL_BROWSER = 'http://localhost:5174';
		const { resolveApiUrl } = await freshImport();
		expect(resolveApiUrl()).toBe('http://localhost:5174');
	});

	it('throws when the server variable is missing', async () => {
		browserState.value = false;
		envState.PUBLIC_API_URL_BROWSER = 'http://localhost:5174';
		const { resolveApiUrl } = await freshImport();
		expect(() => resolveApiUrl()).toThrowError(/PUBLIC_API_URL/);
	});

	it('throws when the browser variable is missing', async () => {
		browserState.value = true;
		envState.PUBLIC_API_URL = 'http://backend-go:8000';
		const { resolveApiUrl } = await freshImport();
		expect(() => resolveApiUrl()).toThrowError(/PUBLIC_API_URL_BROWSER/);
	});
});
