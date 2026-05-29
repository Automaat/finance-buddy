import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('$env/dynamic/public', () => ({
	env: { PUBLIC_API_URL_BROWSER: 'http://localhost:8000', PUBLIC_API_URL: 'http://localhost:8000' }
}));
vi.mock('$app/environment', () => ({ browser: true }));

import { api, ApiError } from './apiClient';

describe('apiClient', () => {
	let fetchSpy: ReturnType<typeof vi.spyOn>;

	beforeEach(() => {
		fetchSpy = vi.spyOn(globalThis, 'fetch');
	});
	afterEach(() => {
		fetchSpy.mockRestore();
	});

	it('GET parses JSON and hits the resolved base URL', async () => {
		fetchSpy.mockResolvedValue(new Response(JSON.stringify({ ok: 1 }), { status: 200 }));
		const out = await api.get<{ ok: number }>('/api/thing');
		expect(out).toEqual({ ok: 1 });
		expect(String(fetchSpy.mock.calls[0][0])).toBe('http://localhost:8000/api/thing');
	});

	it('appends query params, skipping null/undefined/empty', async () => {
		fetchSpy.mockResolvedValue(new Response('[]', { status: 200 }));
		await api.get('/api/x', { query: { a: 1, b: undefined, c: null, d: '', e: 'y' } });
		const url = String(fetchSpy.mock.calls[0][0]);
		expect(url).toContain('a=1');
		expect(url).toContain('e=y');
		expect(url).not.toContain('b=');
		expect(url).not.toContain('c=');
		expect(url).not.toContain('d=');
	});

	it('POST sends a JSON body + content-type', async () => {
		fetchSpy.mockResolvedValue(new Response(JSON.stringify({ id: 1 }), { status: 201 }));
		await api.post('/api/x', { name: 'a' });
		const init = fetchSpy.mock.calls[0][1] as RequestInit;
		expect(init.method).toBe('POST');
		expect(init.body).toBe(JSON.stringify({ name: 'a' }));
		expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json');
	});

	it('DELETE returns undefined on 204 (no body parse)', async () => {
		fetchSpy.mockResolvedValue(new Response(null, { status: 204 }));
		await expect(api.del('/api/x/1')).resolves.toBeUndefined();
	});

	it('throws ApiError with string detail', async () => {
		fetchSpy.mockResolvedValue(
			new Response(JSON.stringify({ detail: 'Not found' }), { status: 404 })
		);
		await expect(api.get('/api/x')).rejects.toMatchObject({ status: 404, message: 'Not found' });
		await expect(api.get('/api/x')).rejects.toBeInstanceOf(ApiError);
	});

	it('joins Pydantic-array detail messages', async () => {
		fetchSpy.mockResolvedValue(
			new Response(JSON.stringify({ detail: [{ msg: 'a bad' }, { msg: 'b bad' }] }), {
				status: 422
			})
		);
		await expect(api.post('/api/x', {})).rejects.toMatchObject({
			status: 422,
			message: 'a bad; b bad'
		});
	});

	it('falls back to status text on a non-JSON error body', async () => {
		fetchSpy.mockResolvedValue(
			new Response('boom', { status: 500, statusText: 'Internal Server Error' })
		);
		await expect(api.get('/api/x')).rejects.toMatchObject({ status: 500 });
	});
});
