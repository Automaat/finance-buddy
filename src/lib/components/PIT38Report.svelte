<script lang="ts">
	import { onMount } from 'svelte';
	import { resolveApiUrl } from '$lib/api';
	import { formatPLN } from '$lib/utils/format';
	import { FileSpreadsheet, Download } from 'lucide-svelte';

	interface SaleRow {
		date: string;
		symbol: string;
		currency: string;
		quantity: string;
		proceeds: string;
		cost_basis: string;
		fees: string;
		realized_gain: string;
		fx_rate: string;
		has_fx: boolean;
		proceeds_pln: string;
		cost_basis_pln: string;
		fees_pln: string;
		realized_pln: string;
	}

	interface Totals {
		proceeds_pln: string;
		cost_basis_pln: string;
		fees_pln: string;
		realized_pln: string;
	}

	interface Report {
		year: number;
		rows: SaleRow[];
		totals: Totals;
	}

	// Polish PIT-38 filings are due by April 30 for the previous year, so
	// default to last year before May 1 and to this year afterwards. Months
	// are 0-indexed: April = 3, May = 4.
	const now = new Date();
	let year = $state(now.getMonth() < 4 ? now.getFullYear() - 1 : now.getFullYear());
	let report = $state<Report | null>(null);
	let loading = $state(false);
	let error = $state('');

	async function load() {
		loading = true;
		error = '';
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/pit38/realized?year=${year}`);
			if (!res.ok) {
				const body = (await res.json().catch(() => null)) as { detail?: string } | null;
				throw new Error(body?.detail ?? `Pobranie raportu nieudane: ${res.statusText}`);
			}
			report = await res.json();
		} catch (err) {
			if (err instanceof Error) error = err.message;
		} finally {
			loading = false;
		}
	}

	function csvUrl(): string {
		return `${resolveApiUrl()}/api/pit38/realized?year=${year}&format=csv`;
	}

	onMount(load);

	function fmt(n: string): string {
		const parsed = Number.parseFloat(n);
		return Number.isFinite(parsed) ? formatPLN(parsed) : n;
	}
</script>

<section class="card preset-filled-surface-100-900 p-4 space-y-3">
	<header class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
		<div>
			<h2 class="h3 flex items-center gap-2">
				<FileSpreadsheet size={20} class="text-primary-500" />
				PIT-38 — zrealizowane zyski
			</h2>
			<p class="text-sm text-surface-700-300">
				Pomocnik do rozliczenia rocznego: sprzedaże, koszt, prowizje, zysk; kursy NBP dla walut
				obcych.
			</p>
		</div>
		<div class="flex items-center gap-2">
			<label class="label">
				<span class="text-xs font-semibold">Rok</span>
				<input
					type="number"
					min="2000"
					max="2100"
					bind:value={year}
					class="input w-24"
					onchange={load}
				/>
			</label>
			<a class="btn preset-filled-primary-500 gap-2" href={csvUrl()} download>
				<Download size={16} /> CSV
			</a>
		</div>
	</header>

	{#if loading}
		<p class="text-sm text-surface-700-300">Ładowanie…</p>
	{:else if error}
		<div class="card preset-tonal-error p-3 text-sm">{error}</div>
	{:else if report && report.rows.length === 0}
		<p class="text-sm text-surface-700-300">Brak sprzedaży w roku {year}.</p>
	{:else if report}
		<div class="overflow-x-auto">
			<table class="table caption-bottom">
				<thead>
					<tr>
						<th>Data</th>
						<th>Symbol</th>
						<th>Waluta</th>
						<th class="text-right">Ilość</th>
						<th class="text-right">Przychód</th>
						<th class="text-right">Koszt</th>
						<th class="text-right">Prowizja</th>
						<th class="text-right">Zysk/strata</th>
						<th class="text-right">Kurs</th>
						<th class="text-right">Zysk PLN</th>
					</tr>
				</thead>
				<tbody>
					{#each report.rows as row (`${row.date}-${row.symbol}`)}
						<tr>
							<td>{row.date}</td>
							<td>{row.symbol}</td>
							<td>{row.currency}</td>
							<td class="text-right">{row.quantity}</td>
							<td class="text-right">{row.proceeds}</td>
							<td class="text-right">{row.cost_basis}</td>
							<td class="text-right">{row.fees}</td>
							<td class="text-right">{row.realized_gain}</td>
							<td class="text-right">{row.has_fx ? row.fx_rate : '—'}</td>
							<td class="text-right font-bold">{fmt(row.realized_pln)}</td>
						</tr>
					{/each}
				</tbody>
				<tfoot>
					<tr class="font-bold">
						<td colspan="4">Razem</td>
						<td class="text-right">{fmt(report.totals.proceeds_pln)}</td>
						<td class="text-right">{fmt(report.totals.cost_basis_pln)}</td>
						<td class="text-right">{fmt(report.totals.fees_pln)}</td>
						<td colspan="2"></td>
						<td class="text-right">{fmt(report.totals.realized_pln)}</td>
					</tr>
				</tfoot>
			</table>
		</div>
	{/if}
</section>
