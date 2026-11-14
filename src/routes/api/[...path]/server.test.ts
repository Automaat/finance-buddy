// @vitest-environment node
import { describe, expect, it, vi } from 'vitest';
import { GET, POST } from './+server';

vi.mock('$app/env/private', () => ({ API_PROXY_TARGET: 'http://backend:8000' }));

type ProxyEvent = Parameters<typeof GET>[0];

function makeEvent(
	path: string,
	upstream: Response,
	cookieJar: Record<string, string> = {},
	method = 'GET'
) {
	const fetch = vi.fn(async (_url: string, _init?: RequestInit) => upstream);
	const url = new URL(`https://finance.example.test/api/${path}`);
	const event = {
		request: new Request(url, { method, body: method === 'GET' ? undefined : '{}' }),
		params: { path: path.split('?')[0] },
		cookies: { get: (name: string) => cookieJar[name] },
		url,
		fetch
	} as unknown as ProxyEvent;
	return { event, fetch };
}

describe('/api proxy — OIDC redirects', () => {
	it('passes a 302 with Location and every Set-Cookie through unchanged', async () => {
		const stateCleared = 'fb_oidc_state=; Path=/api/auth/oidc; Max-Age=0; HttpOnly; SameSite=Lax';
		const session = 'fb_token=jwt; Path=/; HttpOnly; Secure; SameSite=Lax';
		const upstream = new Response(null, {
			status: 302,
			headers: [
				['location', '/'],
				['set-cookie', stateCleared],
				['set-cookie', session]
			]
		});
		const { event, fetch } = makeEvent('auth/oidc/callback?code=c&state=s', upstream, {
			fb_oidc_state: 'payload.sig'
		});

		const res = await GET(event);

		expect(res.status).toBe(302);
		expect(res.headers.get('location')).toBe('/');
		expect(res.headers.getSetCookie()).toEqual([stateCleared, session]);
		const [calledURL, init] = fetch.mock.calls[0];
		expect(calledURL).toBe('http://backend:8000/api/auth/oidc/callback?code=c&state=s');
		expect(init?.redirect).toBe('manual');
		expect(new Headers(init?.headers).get('cookie')).toBe('fb_oidc_state=payload.sig');
	});

	it('passes the provider redirect from the login endpoint through', async () => {
		const authorize = 'https://auth.example.test/authorize?client_id=finance&state=s';
		const upstream = new Response(null, {
			status: 302,
			headers: [
				['location', authorize],
				['set-cookie', 'fb_oidc_state=p.s; Path=/api/auth/oidc; Max-Age=600; HttpOnly']
			]
		});
		const { event } = makeEvent('auth/oidc/login', upstream);

		const res = await GET(event);

		expect(res.status).toBe(302);
		expect(res.headers.get('location')).toBe(authorize);
		expect(res.headers.getSetCookie()).toHaveLength(1);
	});
});

describe('/api proxy — regular requests', () => {
	it('forwards the session as a Bearer token and returns the JSON body', async () => {
		const upstream = new Response('{"ok":true}', {
			status: 200,
			headers: { 'content-type': 'application/json' }
		});
		const { event, fetch } = makeEvent('accounts', upstream, { fb_token: 'jwt' });

		const res = await GET(event);

		expect(res.status).toBe(200);
		expect(await res.json()).toEqual({ ok: true });
		expect(res.headers.get('location')).toBeNull();
		expect(res.headers.getSetCookie()).toEqual([]);
		const headers = new Headers(fetch.mock.calls[0][1]?.headers);
		expect(headers.get('authorization')).toBe('Bearer jwt');
		expect(headers.get('cookie')).toBeNull();
	});

	it.each(['auth/oidc/../../../metrics', '../metrics', 'accounts/./1'])(
		'rejects dot segments without calling the backend: %s',
		async (path) => {
			const { event, fetch } = makeEvent(path, new Response('{}'));

			const res = await GET(event);

			expect(res.status).toBe(404);
			expect(fetch).not.toHaveBeenCalled();
		}
	);

	it('returns a null body for 204 responses', async () => {
		const { event } = makeEvent('accounts/1', new Response(null, { status: 204 }), {}, 'POST');

		const res = await POST(event);

		expect(res.status).toBe(204);
		expect(res.body).toBeNull();
	});
});
