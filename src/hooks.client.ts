import type { HandleClientError } from '@sveltejs/kit/hooks';
import { dev } from '$app/env';
import { toast } from '#lib/stores/toast.svelte.ts';

// SvelteKit error hook — fires on uncaught errors during load() and component
// rendering. The default reaches `+error.svelte`; we also surface a toast so
// the user gets a signal even when the broken view is not the active route.
export const handleError: HandleClientError = ({ kind, error }) => {
	// error() throws keep their own body; before SvelteKit 3 they skipped this hook.
	if (kind === 'app') return;
	const status = kind === 'framework' ? error.status : 500;
	const message =
		kind === 'framework' ? error.message : error instanceof Error ? error.message : String(error);
	if (dev) {
		console.error('[handleError]', error);
	} else {
		console.error('[handleError]', message);
	}
	if (status !== 404) {
		toast.error(`Coś poszło nie tak: ${message}`);
	}
	return {
		message,
		stack: dev && error instanceof Error ? error.stack : undefined
	};
};
