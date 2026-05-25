<script lang="ts">
	import { onMount } from 'svelte';
	import { resolveApiUrl } from '$lib/api';
	import { formatPLN, formatPercent } from '$lib/utils/format';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import { Calculator, Wallet } from 'lucide-svelte';
	import { limitFromRules, optimizeIKZE, type IKZELimitKind } from '$lib/utils/ikzeOptimizer';

	interface YearlyStat {
		year: number;
		account_wrapper: string;
		owner_user_id: number | null;
		limit_amount: number | null;
		total_contributed: number;
		marginal_tax_rate: number | null;
	}

	let stats = $state<YearlyStat[]>([]);
	let owners = $state<OwnerOption[]>([]);
	let loading = $state(true);
	let error = $state('');
	let limitKindByOwner = $state<Record<string, IKZELimitKind>>({});

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
			// Seed each row to the limit kind that matches the configured limit
			// — if the user already set the B2B amount, default the toggle there.
			const seeds: Record<string, IKZELimitKind> = {};
			const b2bLimit = limitFromRules(currentYear, 'b2b');
			for (const row of ikzeRows) {
				const key = String(row.owner_user_id ?? 'shared');
				seeds[key] =
					row.limit_amount != null && b2bLimit != null && row.limit_amount >= b2bLimit
						? 'b2b'
						: 'employee';
			}
			limitKindByOwner = seeds;
		} catch (err) {
			if (err instanceof Error) error = err.message;
		} finally {
			loading = false;
		}
	});

	function setKind(key: string, kind: IKZELimitKind) {
		limitKindByOwner = { ...limitKindByOwner, [key]: kind };
	}

	function recommendationFor(row: YearlyStat) {
		const key = String(row.owner_user_id ?? 'shared');
		const kind = limitKindByOwner[key] ?? 'employee';
		const ruleLimit = limitFromRules(currentYear, kind);
		// Fall back to the configured limit when rules table lacks the year
		// (e.g. historical snapshots) — keeps the optimizer useful in tests
		// and seeded fixtures.
		const limitOverride =
			ruleLimit == null && row.limit_amount != null ? row.limit_amount : undefined;
		return optimizeIKZE({
			year: currentYear,
			limitKind: kind,
			alreadyContributed: row.total_contributed,
			marginalTaxRate: row.marginal_tax_rate ?? 0,
			limitOverride
		});
	}

	function refundLabel(rate: number | null | undefined): string {
		if (rate == null) return 'Szac. zwrot PIT (—)';
		return `Szac. zwrot PIT (${formatPercent(rate * 100)})`;
	}

	function refundValue(
		rate: number | null | undefined,
		amount: number
	): { text: string; hasRate: boolean } {
		if (rate == null) return { text: '—', hasRate: false };
		return { text: formatPLN(amount), hasRate: true };
	}
</script>

<section class="card preset-filled-surface-100-900 p-5 space-y-4">
	<header class="space-y-1">
		<h2 class="h4 flex items-center gap-2">
			<Calculator size={20} class="text-primary-500" />
			Optymalizator wpłat IKZE
		</h2>
		<p class="text-sm text-surface-700-300">
			Ile dopłacić w {currentYear}, aby maksymalnie wykorzystać limit IKZE i ulgę PIT.
		</p>
	</header>

	{#if loading}
		<p class="text-sm text-surface-700-300">Ładowanie…</p>
	{:else if error}
		<div class="card preset-tonal-error p-3 text-sm">{error}</div>
	{:else if ikzeRows.length === 0}
		<p class="text-sm text-surface-700-300">
			Brak kont IKZE w bieżącym roku — dodaj konto i transakcje, aby zobaczyć rekomendację.
		</p>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each ikzeRows as row (row.owner_user_id ?? 'shared')}
				{@const label = ownerName(owners, row.owner_user_id)}
				{@const key = String(row.owner_user_id ?? 'shared')}
				{@const kind = limitKindByOwner[key] ?? 'employee'}
				{@const rec = recommendationFor(row)}
				{@const refund = refundValue(row.marginal_tax_rate, rec.refundEstimate)}
				<article class="card preset-tonal-surface p-4 space-y-3">
					<div class="flex items-center justify-between gap-2">
						<div class="font-semibold flex items-center gap-2">
							<Wallet size={16} class="text-primary-500" />
							{label}
						</div>
						<div class="flex gap-1" role="group" aria-label="Wariant limitu IKZE">
							<button
								type="button"
								class="btn btn-sm {kind === 'employee'
									? 'preset-filled-primary-500'
									: 'preset-tonal-surface'}"
								aria-pressed={kind === 'employee'}
								onclick={() => setKind(key, 'employee')}
							>
								Pracownik
							</button>
							<button
								type="button"
								class="btn btn-sm {kind === 'b2b'
									? 'preset-filled-primary-500'
									: 'preset-tonal-surface'}"
								aria-pressed={kind === 'b2b'}
								onclick={() => setKind(key, 'b2b')}
							>
								B2B
							</button>
						</div>
					</div>

					<dl class="grid grid-cols-2 gap-2 text-sm">
						<div>
							<dt class="text-xs text-surface-600-400">Cel roczny</dt>
							<dd class="font-bold">{formatPLN(rec.annualTarget)}</dd>
						</div>
						<div>
							<dt class="text-xs text-surface-600-400">Wpłacono</dt>
							<dd class="font-bold">{formatPLN(row.total_contributed)}</dd>
						</div>
						<div>
							<dt class="text-xs text-surface-600-400">Cel miesięczny</dt>
							<dd class="font-bold">{formatPLN(rec.monthlyTarget)}</dd>
						</div>
						<div>
							<dt class="text-xs text-surface-600-400">{refundLabel(row.marginal_tax_rate)}</dt>
							<dd class="font-bold {refund.hasRate ? 'text-success-600-400' : ''}">
								{refund.text}
							</dd>
						</div>
					</dl>

					<p class="text-xs text-surface-600-400">
						{#if rec.monthsLeft === 0}
							Rok zakończony — zaktualizuj wartości lub przejdź do nowego roku.
						{:else if rec.remaining === 0}
							Limit wyczerpany — gratulacje, ulga w pełni wykorzystana.
						{:else}
							Pozostało {rec.monthsLeft} mies. × {formatPLN(rec.monthlyTarget)} = {formatPLN(
								rec.remaining
							)}.
						{/if}
					</p>
				</article>
			{/each}
		</div>
	{/if}
</section>
