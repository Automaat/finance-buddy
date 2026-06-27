<script lang="ts">
	import { resolveApiUrl } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';
	import { formatPLN, formatNumber, formatSignedPLN } from '$lib/utils/format';

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
		// Optional: older snapshots / other callers may omit it; treat missing as 0.
		dividends_received_net?: number | null;
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

	async function load(signal: AbortSignal) {
		loading = true;
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/investment/returns?${buildQuery()}`, { signal });
			if (!res.ok) throw new Error('Nie udało się pobrać zwrotów');
			const body = (await res.json()) as ReturnsResponse;
			// Guard the success path too: a response that arrives after this run
			// was superseded (signal aborted) must not overwrite the newer scope's
			// data, even if the fetch itself didn't reject on abort.
			if (signal.aborted) return;
			data = body;
		} catch (err) {
			if (signal.aborted) return;
			if (err instanceof Error) toast.error(err.message);
			data = null;
		} finally {
			if (!signal.aborted) loading = false;
		}
	}

	// Re-fetch on scope/period change; abort the prior request (and on destroy)
	// so a stale response can't overwrite the current scope's data.
	$effect(() => {
		void period;
		void scope.type;
		void scope.value;
		void scope.account_id;
		const controller = new AbortController();
		void load(controller.signal);
		return () => controller.abort();
	});

	function fmtPct(n: number): string {
		const sign = n > 0 ? '+' : '';
		return `${sign}${formatNumber(n, 2)}%`;
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
		<dl class="divide-y divide-surface-200-800">
			<div class="flex items-baseline justify-between gap-3 py-2">
				<dt class="text-xs text-surface-600-400 shrink-0">
					Wpłacono netto
					<span class="block text-[10px] opacity-70">
						+{formatPLN(data.deposits)} / −{formatPLN(data.withdrawals)}
					</span>
				</dt>
				<dd class="text-base font-semibold text-right tabular-nums">
					{formatPLN(data.net_contributed)}
				</dd>
			</div>
			<div class="flex items-baseline justify-between gap-3 py-2">
				<dt class="text-xs text-surface-600-400 shrink-0">
					Wartość obecna
					<span class="block text-[10px] opacity-70">na {data.as_of}</span>
				</dt>
				<dd class="text-base font-semibold text-right tabular-nums">
					{formatPLN(data.current_value)}
				</dd>
			</div>
			<div class="flex items-baseline justify-between gap-3 py-2">
				<dt class="text-xs text-surface-600-400 shrink-0">
					Zmiana wyceny
					<span class="block text-[10px] opacity-70">wartość − wpłaty</span>
				</dt>
				<dd
					class="text-base font-semibold text-right tabular-nums {data.valuation_change >= 0
						? 'text-success-500'
						: 'text-error-500'}"
				>
					{formatSignedPLN(data.valuation_change)}
				</dd>
			</div>
			{#if (data.dividends_received_net ?? 0) !== 0}
				<div class="flex items-baseline justify-between gap-3 py-2">
					<dt class="text-xs text-surface-600-400 shrink-0">
						Dywidendy (netto)
						<span class="block text-[10px] opacity-70">w okresie, po podatku</span>
					</dt>
					<dd class="text-base font-semibold text-right tabular-nums text-success-500">
						+{formatPLN(data.dividends_received_net ?? 0)}
					</dd>
				</div>
			{/if}
			<div class="flex items-baseline justify-between gap-3 py-2">
				<dt class="text-xs text-surface-600-400 shrink-0">
					Zwrot (XIRR)
					{#if data.money_weighted_pct !== null || data.convergence_failed}
						<span class="block text-[10px] opacity-70">
							prosty ROI: {fmtPct(data.simple_roi_pct)}
						</span>
					{/if}
				</dt>
				{#if data.money_weighted_pct !== null}
					<dd
						class="text-base font-semibold text-right tabular-nums {data.money_weighted_pct >= 0
							? 'text-success-500'
							: 'text-error-500'}"
					>
						{fmtPct(data.money_weighted_pct)}
					</dd>
				{:else if data.convergence_failed}
					<dd class="text-sm text-surface-700-300 text-right">Nie udało się obliczyć</dd>
				{:else}
					<dd class="text-sm text-surface-700-300 text-right">—</dd>
				{/if}
			</div>
		</dl>
	{/if}
</section>
