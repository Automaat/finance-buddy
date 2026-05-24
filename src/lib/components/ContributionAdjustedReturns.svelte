<script lang="ts">
	import { resolveApiUrl } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';

	interface ScopeProp {
		type: 'all' | 'category' | 'wrapper' | 'account';
		value?: string;
		account_id?: number;
	}

	interface ReturnsResponse {
		scope: { type: string; value?: string; account_id?: number };
		period: string;
		since: string | null;
		as_of: string;
		deposits: number;
		withdrawals: number;
		net_contributed: number;
		current_value: number;
		valuation_change: number;
		simple_roi_pct: number;
		money_weighted_pct: number | null;
		has_snapshot: boolean;
		convergence_failed: boolean;
	}

	let { scope, title }: { scope: ScopeProp; title?: string } = $props();

	const PERIODS = [
		{ value: '1m', label: '1M' },
		{ value: '3m', label: '3M' },
		{ value: 'ytd', label: 'YTD' },
		{ value: '1y', label: '1Y' },
		{ value: 'all', label: 'Wszystko' }
	];

	let period = $state('all');
	let data = $state<ReturnsResponse | null>(null);
	let loading = $state(false);

	function buildQuery(): string {
		const params = new URLSearchParams({ scope: scope.type, period });
		if (scope.type === 'account' && scope.account_id !== undefined) {
			params.set('id', String(scope.account_id));
		}
		if ((scope.type === 'category' || scope.type === 'wrapper') && scope.value) {
			params.set('value', scope.value);
		}
		return params.toString();
	}

	async function load() {
		loading = true;
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/investment/returns?${buildQuery()}`);
			if (!res.ok) throw new Error('Nie udało się pobrać zwrotów');
			data = (await res.json()) as ReturnsResponse;
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
			data = null;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void period;
		void scope.type;
		void scope.value;
		void scope.account_id;
		load();
	});

	function fmt(n: number): string {
		return n.toLocaleString('pl-PL', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	function fmtPct(n: number): string {
		const sign = n >= 0 ? '+' : '';
		return `${sign}${n.toFixed(2)}%`;
	}
</script>

<section class="card preset-filled-surface-100-900 p-5 space-y-4">
	<header class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h3 class="h3">{title ?? 'Zwroty skorygowane o wpłaty'}</h3>
			<p class="text-xs text-surface-600-400">
				Money-weighted (XIRR) — pokazuje, czy inwestycje wzrosły, a nie tylko ile dodałeś środków.
			</p>
		</div>
		<div class="flex gap-1" role="tablist" aria-label="Okres">
			{#each PERIODS as p}
				<button
					type="button"
					role="tab"
					aria-selected={period === p.value}
					class="btn btn-sm {period === p.value
						? 'preset-filled-primary-500'
						: 'preset-tonal-surface'}"
					onclick={() => (period = p.value)}
				>
					{p.label}
				</button>
			{/each}
		</div>
	</header>

	{#if loading}
		<p class="text-sm text-surface-700-300">Ładowanie…</p>
	{:else if !data}
		<p class="text-sm text-surface-700-300">Brak danych do wyświetlenia.</p>
	{:else if !data.has_snapshot}
		<p class="text-sm text-surface-700-300">
			Brak snapshotów — utwórz snapshot, by zobaczyć zwroty.
		</p>
	{:else}
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
			<div class="card preset-tonal-surface p-3">
				<div class="text-xs text-surface-600-400">Wpłacono netto</div>
				<div class="text-lg font-semibold">{fmt(data.net_contributed)} PLN</div>
				<div class="text-xs text-surface-600-400">
					+{fmt(data.deposits)} / -{fmt(data.withdrawals)}
				</div>
			</div>
			<div class="card preset-tonal-surface p-3">
				<div class="text-xs text-surface-600-400">Wartość obecna</div>
				<div class="text-lg font-semibold">{fmt(data.current_value)} PLN</div>
				<div class="text-xs text-surface-600-400">na {data.as_of}</div>
			</div>
			<div class="card preset-tonal-surface p-3">
				<div class="text-xs text-surface-600-400">Zmiana wyceny</div>
				<div
					class="text-lg font-semibold {data.valuation_change >= 0
						? 'text-success-500'
						: 'text-error-500'}"
				>
					{data.valuation_change >= 0 ? '+' : ''}{fmt(data.valuation_change)} PLN
				</div>
				<div class="text-xs text-surface-600-400">wartość − wpłaty</div>
			</div>
			<div class="card preset-tonal-surface p-3">
				<div class="text-xs text-surface-600-400">Zwrot (XIRR)</div>
				{#if data.money_weighted_pct !== null}
					<div
						class="text-lg font-semibold {data.money_weighted_pct >= 0
							? 'text-success-500'
							: 'text-error-500'}"
					>
						{fmtPct(data.money_weighted_pct)}
					</div>
					<div class="text-xs text-surface-600-400">
						prosty ROI: {fmtPct(data.simple_roi_pct)}
					</div>
				{:else if data.convergence_failed}
					<div class="text-sm text-surface-700-300">Nie udało się obliczyć</div>
					<div class="text-xs text-surface-600-400">
						prosty ROI: {fmtPct(data.simple_roi_pct)}
					</div>
				{:else}
					<div class="text-sm text-surface-700-300">—</div>
				{/if}
			</div>
		</div>
	{/if}
</section>
