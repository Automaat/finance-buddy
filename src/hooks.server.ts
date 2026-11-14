import { redirect, type Handle, type HandleFetch } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';

/**
 * The OIDC callback sets a fresh fb_token. The stale-token delete in `handle`
 * would be sent after it and override it, so it is skipped on this path.
 */
const OIDC_CALLBACK_PATH = '/api/auth/oidc/callback';

/** Exact matches only: a prefix would let `..%2F` segments reach other backend paths. */
const PUBLIC_API_PATHS = new Set([
	'/api/auth/login',
	'/api/auth/logout',
	'/api/auth/oidc/status',
	'/api/auth/oidc/login',
	OIDC_CALLBACK_PATH
]);

interface SessionUser {
	username: string;
	name: string;
	isAdmin: boolean;
}

// decodeUser reads the JWT payload without verifying the signature (the
// backend does that on every API call) — enough to know who is logged in and
// to drop a token whose `exp` has already passed.
function decodeUser(token: string): SessionUser | null {
	try {
		const payload = token.split('.')[1];
		if (!payload) {
			return null;
		}
		const pad = payload.length % 4 === 0 ? '' : '='.repeat(4 - (payload.length % 4));
		const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/') + pad);
		const claims = JSON.parse(json) as {
			username?: unknown;
			name?: unknown;
			is_admin?: unknown;
			exp?: unknown;
		};
		// Reject malformed claims. The backend still verifies the signature on
		// every API call — this only stops a bogus token from looking
		// logged-in to the page-level gating.
		if (typeof claims.exp !== 'number' || claims.exp * 1000 < Date.now()) {
			return null;
		}
		if (typeof claims.username !== 'string' || claims.username === '') {
			return null;
		}
		if (typeof claims.is_admin !== 'boolean') {
			return null;
		}
		// name is optional — tokens issued before the persona merge lack it.
		return {
			username: claims.username,
			name: typeof claims.name === 'string' ? claims.name : '',
			isAdmin: claims.is_admin
		};
	} catch {
		return null;
	}
}

export const handle: Handle = async ({ event, resolve }) => {
	const { pathname } = event.url;
	const token = event.cookies.get('fb_token');
	const user = token ? decodeUser(token) : null;
	event.locals.user = user;

	if (user) {
		return resolve(event);
	}

	// A lingering expired/invalid token — clear it so the browser stops sending it.
	if (token && pathname !== OIDC_CALLBACK_PATH) {
		event.cookies.delete('fb_token', { path: '/' });
	}

	const isAsset = pathname.startsWith('/_app/') || /\.\w+$/.test(pathname);
	const isLogin = pathname === '/login';
	const isPublicApi = PUBLIC_API_PATHS.has(pathname);
	if (isAsset || isLogin || isPublicApi) {
		return resolve(event);
	}

	// API calls get a JSON 401; page navigations get sent to the login screen.
	if (pathname.startsWith('/api/')) {
		return new Response(JSON.stringify({ detail: 'Not authenticated' }), {
			status: 401,
			headers: { 'content-type': 'application/json' }
		});
	}
	redirect(303, '/login');
};

// handleFetch attaches the session token to server-side load() calls that go
// straight to the backend (PUBLIC_API_URL). Browser-side calls instead route
// through the /api proxy, which adds the token itself.
export const handleFetch: HandleFetch = async ({ event, request, fetch }) => {
	const backend = env.PUBLIC_API_URL;
	if (backend && request.url.startsWith(backend)) {
		const token = event.cookies.get('fb_token');
		if (token) {
			request.headers.set('authorization', `Bearer ${token}`);
		}
	}
	return fetch(request);
};
