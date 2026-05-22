// Server-side proxy for the Go backend.
//
// The browser only ever talks to this SvelteKit origin; this route forwards
// /api/* to the backend and attaches the session token from the fb_token
// cookie as a Bearer header. Keeps the JWT out of client-side JavaScript.
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

function backendURL(path: string, search: string): string {
	const base = env.API_PROXY_TARGET ?? 'http://localhost:8000';
	return `${base}/api/${path}${search}`;
}

const proxy: RequestHandler = async ({ request, params, cookies, url, fetch }) => {
	const headers = new Headers();
	const contentType = request.headers.get('content-type');
	if (contentType) {
		headers.set('content-type', contentType);
	}
	const token = cookies.get('fb_token');
	if (token) {
		headers.set('authorization', `Bearer ${token}`);
	}

	const init: RequestInit = { method: request.method, headers };
	if (request.method !== 'GET' && request.method !== 'HEAD') {
		init.body = await request.text();
	}

	const upstream = await fetch(backendURL(params.path, url.search), init);
	const body = await upstream.arrayBuffer();
	// Only content-type is forwarded. Set-Cookie in particular is dropped on
	// purpose — the backend's own session cookie must never reach the browser;
	// frontend sessions are managed by the /login and /logout routes.
	const responseHeaders = new Headers();
	const responseType = upstream.headers.get('content-type');
	if (responseType) {
		responseHeaders.set('content-type', responseType);
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
