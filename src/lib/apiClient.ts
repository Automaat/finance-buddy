import { resolveApiUrl } from '$lib/api';

// ApiError carries the HTTP status plus a human message extracted from the
// backend's error body, so callers can branch on `status` (e.g. 404 vs 422)
// while still having a ready-to-show `message`.
export class ApiError extends Error {
	readonly status: number;
	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

type FetchFn = typeof globalThis.fetch;

export interface ApiOptions {
	// Pass SvelteKit's `fetch` from a load() so SSR cookies/relative URLs work;
	// defaults to the global fetch for browser-side mutations.
	fetch?: FetchFn;
	// Query params appended to the path.
	query?: Record<string, string | number | undefined | null>;
	signal?: AbortSignal;
}

// extractDetail turns the backend's error body into a single message. The Go
// backend emits either {"detail": "msg"} or a Pydantic-shaped
// {"detail": [{"msg": "..."}]}; fall back to the status text otherwise.
function extractDetail(body: unknown, fallback: string): string {
	if (body && typeof body === 'object' && 'detail' in body) {
		const detail = (body as { detail: unknown }).detail;
		if (typeof detail === 'string' && detail) return detail;
		if (Array.isArray(detail)) {
			const msgs = detail
				.map((d) =>
					d && typeof d === 'object' && 'msg' in d ? String((d as { msg: unknown }).msg) : ''
				)
				.filter(Boolean);
			if (msgs.length > 0) return msgs.join('; ');
		}
	}
	return fallback;
}

function buildUrl(path: string, query?: ApiOptions['query']): string {
	const base = resolveApiUrl();
	if (!query) return `${base}${path}`;
	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(query)) {
		if (value !== undefined && value !== null && value !== '') {
			params.set(key, String(value));
		}
	}
	const qs = params.toString();
	return qs ? `${base}${path}?${qs}` : `${base}${path}`;
}

async function request<T>(
	method: string,
	path: string,
	body: unknown,
	opts: ApiOptions = {}
): Promise<T> {
	const doFetch = opts.fetch ?? globalThis.fetch;
	const init: RequestInit = { method, signal: opts.signal };
	if (body !== undefined) {
		init.headers = { 'Content-Type': 'application/json' };
		init.body = JSON.stringify(body);
	}
	const res = await doFetch(buildUrl(path, opts.query), init);
	if (!res.ok) {
		const errBody = await res.json().catch(() => null);
		throw new ApiError(res.status, extractDetail(errBody, res.statusText));
	}
	// 204 No Content (and empty bodies) have nothing to parse.
	if (res.status === 204) return undefined as T;
	const text = await res.text();
	return (text ? JSON.parse(text) : undefined) as T;
}

export const api = {
	get: <T>(path: string, opts?: ApiOptions) => request<T>('GET', path, undefined, opts),
	post: <T>(path: string, body?: unknown, opts?: ApiOptions) =>
		request<T>('POST', path, body, opts),
	put: <T>(path: string, body?: unknown, opts?: ApiOptions) => request<T>('PUT', path, body, opts),
	del: <T = void>(path: string, opts?: ApiOptions) => request<T>('DELETE', path, undefined, opts)
};
