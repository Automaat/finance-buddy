import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';

// resolveApiUrl returns the API base URL for the current execution context.
//
// In the browser, returns PUBLIC_API_URL_BROWSER (the SvelteKit origin —
// calls hit the /api proxy in routes/api/[...path]/+server.ts). On the
// server, e.g. during SSR for a universal load, returns PUBLIC_API_URL
// (the backend's reachable hostname inside the Docker network).
//
// Throws when the relevant variable is not configured. Callers in load
// functions can let it propagate — SvelteKit converts an uncaught throw
// into a 500.
export function resolveApiUrl(): string {
	const url = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
	if (!url) {
		const which = browser ? 'PUBLIC_API_URL_BROWSER' : 'PUBLIC_API_URL';
		throw new Error(`${which} is not configured`);
	}
	return url;
}
