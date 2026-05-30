<script lang="ts">
	import { page } from '$app/state';
	import { Briefcase, Coins, PiggyBank, TrendingUp } from 'lucide-svelte';

	let { children } = $props();

	const tabs = [
		{ href: '/investments/holdings', label: 'Akcje / ETF', icon: Briefcase },
		{ href: '/investments/bonds', label: 'Obligacje', icon: Coins },
		{ href: '/investments/ppk', label: 'PPK', icon: PiggyBank },
		{ href: '/investments/returns', label: 'Zwroty', icon: TrendingUp }
	];

	const activeTab = $derived(page.url.pathname);
</script>

<div class="space-y-4">
	<nav class="flex gap-1 border-b border-surface-200-800" aria-label="Inwestycje">
		{#each tabs as tab (tab.href)}
			{@const SvelteComponent = tab.icon}
			<a
				href={tab.href}
				class="flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors -mb-px {activeTab ===
				tab.href
					? 'border-primary-500 text-primary-700-300'
					: 'border-transparent text-surface-700-300 hover:text-surface-900-100 hover:bg-surface-100-900/40'}"
				aria-current={activeTab === tab.href ? 'page' : undefined}
			>
				<SvelteComponent size={16} />
				<span>{tab.label}</span>
			</a>
		{/each}
	</nav>

	{@render children()}
</div>
