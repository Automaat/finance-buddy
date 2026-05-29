<script lang="ts">
	import { formatPercent, formatSignedPercent } from '$lib/utils/format';
	import { categoryLabel } from '$lib/utils/categories';

	// Minimal shape of the /api/accounts rows this table needs. real_yield_pct
	// is populated server-side only for interest-bearing accounts (post-Belka,
	// post-CPI); rows without a nominal rate are filtered out.
	export interface RealYieldAccount {
		id: number;
		name: string;
		category: string;
		account_wrapper: string | null;
		interest_rate_pct: number | null;
		cpi_yoy_pct: number | null;
		real_yield_pct: number | null;
	}

	interface Props {
		accounts: RealYieldAccount[];
	}

	let { accounts }: Props = $props();

	const rows = $derived(
		accounts
			.filter((a) => a.interest_rate_pct != null)
			.sort((a, b) => (b.real_yield_pct ?? -Infinity) - (a.real_yield_pct ?? -Infinity))
	);

	// Color buckets match the /accounts real-yield UI (issue #573): green > 1%,
	// amber 0–1% ("not yet comfortably beating inflation"), red < 0%.
	const realClass = (value: number | null): string => {
		if (value == null) return 'text-surface-600-400';
		if (value < 0) return 'text-error-600-400';
		if (value <= 1) return 'text-warning-600-400';
		return 'text-success-600-400';
	};
</script>

{#if rows.length > 0}
	<div class="card preset-filled-surface-100-900 p-4 overflow-x-auto">
		<table class="w-full text-sm">
			<thead>
				<tr class="text-left opacity-75 border-b border-surface-300-700">
					<th class="py-2 pr-3">Konto</th>
					<th class="py-2 px-3">Kategoria</th>
					<th class="py-2 px-3 text-right">Nominalnie</th>
					<th class="py-2 px-3 text-right">Inflacja r/r</th>
					<th class="py-2 pl-3 text-right">Realnie (po podatku)</th>
				</tr>
			</thead>
			<tbody>
				{#each rows as row (row.id)}
					<tr class="border-b border-surface-200-800 last:border-0">
						<td class="py-2 pr-3 font-medium">
							{row.name}
							{#if row.account_wrapper}
								<span class="ml-1 text-xs opacity-60">({row.account_wrapper})</span>
							{/if}
						</td>
						<td class="py-2 px-3">{categoryLabel(row.category)}</td>
						<td class="py-2 px-3 text-right">{formatPercent(row.interest_rate_pct)}</td>
						<td class="py-2 px-3 text-right text-surface-600-400">
							{row.cpi_yoy_pct == null ? '—' : formatSignedPercent(-row.cpi_yoy_pct)}
						</td>
						<td class="py-2 pl-3 text-right font-bold {realClass(row.real_yield_pct)}">
							{formatSignedPercent(row.real_yield_pct)}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
		<p class="mt-3 text-xs opacity-60">
			Realny zwrot = oprocentowanie po podatku Belki (konta IKE/IKZE zwolnione) minus inflacja r/r.
		</p>
	</div>
{:else}
	<div class="card preset-filled-surface-100-900 p-4 text-surface-600-400">
		Brak kont z oprocentowaniem do porównania.
	</div>
{/if}
