// Server-side proxy for the Go backend.
//
// The browser only ever talks to this SvelteKit origin; this route forwards
// /api/* to the backend and attaches the session token from the fb_token
// cookie as a Bearer header. Keeps the JWT out of client-side JavaScript.
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

/** Signed OIDC login state; the only browser cookie the backend needs to read. */
const OIDC_STATE_COOKIE = 'fb_oidc_state';

/** SvelteKit decodes `..%2F` in the rest param; a dot segment would escape /api on the backend. */
function hasDotSegment(path: string): boolean {
	return path.split('/').some((segment) => segment === '.' || segment === '..');
}

function backendURL(path: string, search: string): string {
	const base = env.API_PROXY_TARGET ?? 'http://localhost:8000';
	return `${base}/api/${path}${search}`;
}

const proxy: RequestHandler = async ({ request, params, cookies, url, fetch }) => {
	if (hasDotSegment(params.path)) {
		return Response.json({ detail: 'Not Found' }, { status: 404 });
	}
	const headers = new Headers();
	const contentType = request.headers.get('content-type');
	if (contentType) {
		headers.set('content-type', contentType);
	}
	const token = cookies.get('fb_token');
	if (token) {
		headers.set('authorization', `Bearer ${token}`);
	}
	const oidcState = cookies.get(OIDC_STATE_COOKIE);
	if (oidcState) {
		headers.set('cookie', `${OIDC_STATE_COOKIE}=${oidcState}`);
	}

	// Manual so OIDC login/callback 302s reach the browser instead of being followed here.
	const init: RequestInit = { method: request.method, headers, redirect: 'manual' };
	if (request.method !== 'GET' && request.method !== 'HEAD') {
		init.body = await request.text();
	}

	const upstream = await fetch(backendURL(params.path, url.search), init);
	const body = await upstream.arrayBuffer();
	// Set-Cookie and Location pass through unchanged: the OIDC callback issues
	// the session cookie on this origin via the backend's redirect response.
	const responseHeaders = new Headers();
	const responseType = upstream.headers.get('content-type');
	if (responseType) {
		responseHeaders.set('content-type', responseType);
	}
	const location = upstream.headers.get('location');
	if (location) {
		responseHeaders.set('location', location);
	}
	for (const cookie of upstream.headers.getSetCookie()) {
		responseHeaders.append('set-cookie', cookie);
	}
	// 204/304 are null-body statuses — Response throws if handed a body, even
	// an empty one (e.g. the backend's 204 on DELETE).
	const responseBody = body.byteLength > 0 ? body : null;
	return new Response(responseBody, { status: upstream.status, headers: responseHeaders });
};

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
