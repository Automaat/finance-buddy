// @vitest-environment node
import { describe, expect, it, vi } from 'vitest';
import { handle } from './hooks.server';

vi.mock('$env/dynamic/public', () => ({ env: {} }));

type HandleInput = Parameters<typeof handle>[0];

const expiredToken = `x.${btoa(JSON.stringify({ username: 'u', is_admin: false, exp: 1 }))}.y`;

function runHandle(pathname: string, token?: string) {
	const cookies = {
		get: vi.fn((name: string) => (name === 'fb_token' ? token : undefined)),
		delete: vi.fn()
	};
	const resolve = vi.fn(async () => new Response('resolved'));
	const event = {
		url: new URL(`https://finance.example.test${pathname}`),
		cookies,
		locals: {}
	};
	const result = handle({ event, resolve } as unknown as HandleInput);
	return { result, resolve, cookies };
}

describe('handle — public OIDC endpoints', () => {
	it.each(['/api/auth/oidc/status', '/api/auth/oidc/login', '/api/auth/oidc/callback'])(
		'lets %s through without a session',
		async (path) => {
			const { result, resolve } = runHandle(path);
			expect(await (await result).text()).toBe('resolved');
			expect(resolve).toHaveBeenCalledOnce();
		}
	);

	it.each(['/api/auth/oidc/..%2F..%2F..%2Fmetrics', '/api/auth/oidc/other', '/api/auth/oidc'])(
		'rejects other paths under the OIDC prefix: %s',
		async (path) => {
			const { result, resolve } = runHandle(path);
			expect((await result).status).toBe(401);
			expect(resolve).not.toHaveBeenCalled();
		}
	);
});

describe('handle — stale session cookie', () => {
	it('clears an expired fb_token on other routes', async () => {
		const { result, cookies } = runHandle('/api/auth/oidc/status', expiredToken);
		await result;
		expect(cookies.delete).toHaveBeenCalledWith('fb_token', { path: '/' });
	});

	it('keeps it on the OIDC callback so the new session cookie is not overridden', async () => {
		const { result, cookies } = runHandle('/api/auth/oidc/callback', expiredToken);
		await result;
		expect(cookies.delete).not.toHaveBeenCalled();
	});
});
