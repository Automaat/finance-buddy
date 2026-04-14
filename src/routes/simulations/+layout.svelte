<script lang="ts">
	import { page } from '$app/stores';

	const tabs = [
		{ href: '/simulations', label: 'Emerytalne' },
		{ href: '/simulations/mortgage', label: 'Hipoteka' },
		{ href: '/simulations/zus', label: 'Emerytura ZUS' },
		{ href: '/simulations/kalkulator', label: 'Kalkulator' }
	];

	let { children } = $props();
</script>

<div class="tabs">
	{#each tabs as tab}
		<a href={tab.href} class="tab" class:active={$page.url.pathname === tab.href}>
			{tab.label}
		</a>
	{/each}
</div>

{@render children?.()}

<style>
	.tabs {
		display: flex;
		gap: var(--size-1);
		border-bottom: 2px solid var(--surface-3);
		margin-bottom: var(--size-6);
		overflow-x: auto;
	}

	.tab {
		padding: var(--size-2) var(--size-4);
		font-size: var(--font-size-1);
		font-weight: 500;
		color: var(--color-text-3);
		text-decoration: none;
		border-bottom: 2px solid transparent;
		margin-bottom: -2px;
		white-space: nowrap;
		transition:
			color 0.15s,
			border-color 0.15s;
	}

	.tab:hover {
		color: var(--color-text-1);
	}

	.tab.active {
		color: var(--color-primary);
		border-bottom-color: var(--color-primary);
		font-weight: 600;
	}
</style>
