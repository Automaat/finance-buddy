<script lang="ts">
	import { onMount } from 'svelte';
	import { resolveApiUrl } from '$lib/api';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import { ShieldCheck, Receipt } from 'lucide-svelte';

	interface YearlyStat {
		year: number;
		account_wrapper: string;
		owner_user_id: number | null;
		limit_amount: number | null;
		total_contributed: number;
		remaining: number | null;
		percentage_used: number | null;
		marginal_tax_rate: number | null;
		pit_savings: number | null;
		is_warning: boolean;
	}

	let stats = $state<YearlyStat[]>([]);
	let owners = $state<OwnerOption[]>([]);
	let loading = $state(true);
	let error = $state('');

	const currentYear = new Date().getFullYear();
	const ikzeRows = $derived(stats.filter((s) => s.account_wrapper === 'IKZE'));

	onMount(async () => {
		try {
			const apiUrl = resolveApiUrl();
			const [statsRes, ownersRes] = await Promise.all([
				fetch(`${apiUrl}/api/retirement/stats?year=${currentYear}`),
				fetch(`${apiUrl}/api/users`)
			]);
			if (!statsRes.ok) throw new Error(`Stats failed: ${statsRes.statusText}`);
			stats = await statsRes.json();
			owners = ownersRes.ok ? await ownersRes.json() : [];
		} catch (err) {
			if (err instanceof Error) error = err.message;
		} finally {
			loading = false;
		}
	});

	function fmtPLN(value: number | null | undefined): string {
		if (value === null || value === undefined) return '—';
		return value.toLocaleString('pl-PL', { maximumFractionDigits: 0 }) + ' PLN';
	}

	function fmtPct(value: number | null | undefined): string {
		if (value === null || value === undefined) return '—';
		return `${value.toFixed(1)}%`;
	}

	function fmtRate(value: number | null | undefined): string {
		if (value === null || value === undefined) return '—';
		return `${(value * 100).toFixed(0)}%`;
	}

	function progressClass(pct: number | null | undefined): string {
		if (!pct) return 'bg-surface-300-700';
		// Order matters: ≥100 must check first or the ≥90 branch shadows it.
		if (pct >= 100) return 'bg-success-500';
		if (pct >= 90) return 'bg-warning-500';
		return 'bg-primary-500';
	}
</script>

<section class="card preset-filled-surface-100-900 p-5 space-y-4">
	<header class="space-y-1">
		<h2 class="h4 flex items-center gap-2">
			<Receipt size={20} class="text-primary-500" />
			IKZE — oszczędność na PIT
		</h2>
		<p class="text-sm text-surface-700-300">
			Wpłaty na IKZE odliczasz od dochodu — szacunek oszczędności podatkowej w {currentYear}.
		</p>
	</header>

	{#if loading}
		<p class="text-sm text-surface-700-300">Ładowanie…</p>
	{:else if error}
		<div class="card preset-tonal-error p-3 text-sm">{error}</div>
	{:else if ikzeRows.length === 0}
		<p class="text-sm text-surface-700-300">
			Brak kont IKZE w bieżącym roku — dodaj konto i transakcje, aby zobaczyć oszczędność na PIT.
		</p>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each ikzeRows as row (row.owner_user_id ?? 'shared')}
				{@const label = row.owner_user_id ? ownerName(owners, row.owner_user_id) : 'Wspólne'}
				{@const pct = row.percentage_used ?? 0}
				<article class="card preset-tonal-surface p-4 space-y-3">
					<div class="flex items-center justify-between gap-2">
						<div class="font-semibold flex items-center gap-2">
							<ShieldCheck size={16} class="text-primary-500" />
							{label}
						</div>
						{#if row.is_warning}
							<span class="badge preset-filled-warning-500 text-xs">Limit blisko</span>
						{/if}
					</div>

					<div class="space-y-1">
						<div class="flex justify-between text-sm">
							<span class="text-surface-700-300">Wpłacono</span>
							<span class="font-semibold">
								{fmtPLN(row.total_contributed)} / {fmtPLN(row.limit_amount)}
							</span>
						</div>
						<div class="h-2 w-full bg-surface-300-700 rounded-full overflow-hidden">
							<div class="h-full {progressClass(pct)}" style="width: {Math.min(pct, 100)}%"></div>
						</div>
						<div class="flex justify-between text-xs text-surface-600-400">
							<span>{fmtPct(row.percentage_used)} limitu</span>
							<span>Pozostało {fmtPLN(row.remaining)}</span>
						</div>
					</div>

					<div class="grid grid-cols-2 gap-2 pt-2 border-t border-surface-200-800">
						<div>
							<div class="text-xs text-surface-600-400">Stawka krańcowa</div>
							<div class="font-bold">{fmtRate(row.marginal_tax_rate)}</div>
						</div>
						<div>
							<div class="text-xs text-surface-600-400">Szac. oszczędność PIT</div>
							<div class="font-bold text-success-600-400">{fmtPLN(row.pit_savings)}</div>
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</section>
