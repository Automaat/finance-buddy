import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';
import { error } from '@sveltejs/kit';

// resolveApiUrl returns the API base URL for the current execution context.
//
// In the browser, returns PUBLIC_API_URL_BROWSER (the SvelteKit origin —
// calls hit the /api proxy in routes/api/[...path]/+server.ts). On the
// server, e.g. during SSR for a universal load, returns PUBLIC_API_URL
// (the backend's reachable hostname inside the Docker network).
//
// Throws a SvelteKit 500 when the relevant variable is unset. Load
// functions that catch errors and rethrow only those with a `status`
// preserve this actionable message instead of swallowing it as a
// generic "Failed to load …".
export function resolveApiUrl(): string {
	const url = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
	if (!url) {
		const which = browser ? 'PUBLIC_API_URL_BROWSER' : 'PUBLIC_API_URL';
		error(500, `${which} is not configured`);
	}
	return url;
}
