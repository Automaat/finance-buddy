<script lang="ts">
	import { onMount } from 'svelte';
	import { dev } from '$app/environment';
	import { toast } from '$lib/stores/toast.svelte';

	// Errors we have already toasted in the last few seconds — dedupes runaway
	// loops where the same error fires repeatedly (React-style infinite render
	// or fetch-storm), instead of spamming the screen.
	const recent = new Map<string, number>();
	const DEDUP_WINDOW_MS = 5000;

	function show(message: string): void {
		const now = Date.now();
		for (const [key, ts] of recent) {
			if (now - ts > DEDUP_WINDOW_MS) recent.delete(key);
		}
		if (recent.has(message)) return;
		recent.set(message, now);
		toast.error(message);
	}

	function describe(reason: unknown): string {
		if (reason instanceof Error) return reason.message || reason.name;
		if (typeof reason === 'string') return reason;
		try {
			return JSON.stringify(reason);
		} catch {
			return String(reason);
		}
	}

	onMount(() => {
		const onError = (event: ErrorEvent) => {
			const msg = event.error instanceof Error ? event.error.message : event.message;
			console.error('[window.onerror]', event.error ?? event.message);
			show(`Niespodziewany błąd: ${msg || 'unknown'}`);
		};
		const onRejection = (event: PromiseRejectionEvent) => {
			console.error('[unhandledrejection]', event.reason);
			show(`Niespodziewany błąd: ${describe(event.reason)}`);
		};
		window.addEventListener('error', onError);
		window.addEventListener('unhandledrejection', onRejection);
		if (dev) {
			console.info('[GlobalErrorCapture] window-level handlers attached');
		}
		return () => {
			window.removeEventListener('error', onError);
			window.removeEventListener('unhandledrejection', onRejection);
		};
	});
</script>
