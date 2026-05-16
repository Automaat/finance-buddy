<script lang="ts">
	import { page } from '$app/stores';
	import { invalidateAll } from '$app/navigation';
	import { TriangleAlert, RefreshCw } from 'lucide-svelte';

	let retrying = $state(false);

	async function retry() {
		retrying = true;
		try {
			await invalidateAll();
		} finally {
			retrying = false;
		}
	}
</script>

<div class="error-boundary">
	<TriangleAlert class="error-icon" size={48} />
	<h1>Coś poszło nie tak</h1>
	<p class="error-detail">
		{$page.error?.message || 'Nie udało się załadować danych.'}
	</p>
	<p class="error-status">Kod błędu: {$page.status}</p>
	<div class="error-actions">
		<button type="button" onclick={retry} disabled={retrying}>
			<RefreshCw size={18} />
			{retrying ? 'Ponawianie…' : 'Spróbuj ponownie'}
		</button>
		<a href="/" class="home-link">Wróć do pulpitu</a>
	</div>
</div>

<style>
	.error-boundary {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: var(--size-3);
		padding: var(--size-8) var(--size-4);
		max-width: 32rem;
		margin: 0 auto;
	}

	.error-boundary :global(.error-icon) {
		color: var(--color-error);
	}

	h1 {
		font-size: var(--font-size-5);
		font-weight: var(--font-weight-7);
		color: var(--color-text-1);
		margin: 0;
	}

	.error-detail {
		font-size: var(--font-size-2);
		color: var(--color-text-2);
		margin: 0;
	}

	.error-status {
		font-size: var(--font-size-0);
		color: var(--color-text-muted);
		margin: 0;
	}

	.error-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: var(--size-3);
		margin-top: var(--size-3);
	}

	button {
		display: inline-flex;
		align-items: center;
		gap: var(--size-2);
		padding: var(--size-2) var(--size-4);
		min-height: var(--tap-target-min);
		font-size: var(--font-size-1);
		font-weight: var(--font-weight-6);
		color: white;
		background: var(--color-primary);
		border: none;
		border-radius: var(--radius-2);
		cursor: pointer;
	}

	button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.home-link {
		display: inline-flex;
		align-items: center;
		padding: var(--size-2) var(--size-4);
		min-height: var(--tap-target-min);
		font-size: var(--font-size-1);
		font-weight: var(--font-weight-6);
		color: var(--color-text-1);
		background: var(--surface-3);
		border-radius: var(--radius-2);
		text-decoration: none;
	}
</style>
