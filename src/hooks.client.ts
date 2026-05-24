import type { HandleClientError } from '@sveltejs/kit';
import { dev } from '$app/environment';
import { toast } from '$lib/stores/toast.svelte';

// SvelteKit error hook — fires on uncaught errors during load() and component
// rendering. The default reaches `+error.svelte`; we also surface a toast so
// the user gets a signal even when the broken view is not the active route.
export const handleError: HandleClientError = ({ error, status }) => {
	const message = error instanceof Error ? error.message : String(error);
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
