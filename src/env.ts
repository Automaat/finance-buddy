import { defineEnvVars } from '@sveltejs/kit/env';

// Read at startup, not inlined; unset stays undefined like $env/dynamic.
const optional = (input: string | undefined): string | undefined => input;

export const variables = defineEnvVars({
	PUBLIC_API_URL: { public: true, schema: optional },
	PUBLIC_API_URL_BROWSER: { public: true, schema: optional },
	API_PROXY_TARGET: { schema: optional },
	FB_COOKIE_SECURE: { schema: optional }
});
