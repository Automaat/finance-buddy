<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { browser } from '$app/environment';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import type { SalaryRecord } from '$lib/types/salaries';
	import {
		type OfferInput,
		type OfferBreakdown,
		ContractType,
		B2BTaxForm,
		ZUSTier,
		calculateOffer,
		findBreakEvenAmount
	} from '$lib/utils/compensation';

	export let data: { latestSalaries: SalaryRecord[] };

	function mapContractType(dbType: string): ContractType | null {
		if (dbType === 'B2B') return ContractType.B2B_MONTHLY;
		if (dbType === 'UOP') return ContractType.UOP;
		return null; // UZ, UoD not supported in calculator
	}

	function emptyOffer(name: string): OfferInput {
		return {
			name,
			contractType: ContractType.UOP,
			grossMonthly: 0,
			ppkEnabled: false,
			netInvoice: 0,
			hourlyRate: 0,
			hoursPerMonth: 160,
			taxForm: B2BTaxForm.LINIOWY,
			zusTier: ZUSTier.PELNY,
			accountingCost: 0,
			rsuAnnual: 0,
			isCurrentJob: false
		};
	}

	function toggleCurrentJob(index: number) {
		const offer = offers[index];
		if (offer.isCurrentJob) {
			// Unchecked → clear
			offer.isCurrentJob = false;
		} else {
			// Uncheck any other current job
			for (const o of offers) o.isCurrentJob = false;
			offer.isCurrentJob = true;

			// Populate from latest salary (pick first available)
			const salary = data.latestSalaries[0];
			if (salary) {
				const contractType = mapContractType(salary.contract_type);
				if (contractType !== null) {
					offer.name = 'Aktualna praca';
					offer.contractType = contractType;
					if (offer.contractType === ContractType.UOP) {
						offer.grossMonthly = salary.gross_amount;
					} else {
						offer.netInvoice = salary.gross_amount;
					}
				}
			}
		}
		offers = [...offers];
	}

	let offers: OfferInput[] = [emptyOffer('Oferta 1'), emptyOffer('Oferta 2')];
	let results: OfferBreakdown[] | null = null;
	let winner: OfferBreakdown | null = null;
	let breakEvens: Map<number, number> = new Map();

	let chartContainer: HTMLDivElement;
	let chart: echarts.ECharts | null = null;

	function addOffer() {
		offers = [...offers, emptyOffer(`Oferta ${offers.length + 1}`)];
	}

	function removeOffer(index: number) {
		offers = offers.filter((_, i) => i !== index);
	}

	function isB2B(type: ContractType): boolean {
		return type === ContractType.B2B_MONTHLY || type === ContractType.B2B_HOURLY;
	}

	function breakEvenLabel(type: ContractType): string {
		switch (type) {
			case ContractType.UOP:
				return 'brutto';
			case ContractType.B2B_MONTHLY:
				return 'na fakturze';
			case ContractType.B2B_HOURLY:
				return 'PLN/h';
		}
	}

	async function compare() {
		results = offers.map((o) => calculateOffer(o));
		winner = results.reduce((best, r) => (r.netMonthly > best.netMonthly ? r : best));

		breakEvens = new Map();
		const base = results.find((r) => r.isCurrentJob);
		if (base) {
			for (let i = 0; i < offers.length; i++) {
				if (!results[i].isCurrentJob) {
					breakEvens.set(i, findBreakEvenAmount(offers[i], base.netMonthly));
				}
			}
		}

		await tick();
		renderChart();
	}

	type RowDef = {
		label: string;
		value: (b: OfferBreakdown) => string;
		delta?: (b: OfferBreakdown, base: OfferBreakdown) => number;
		bold?: boolean;
	};

	const tableRows: RowDef[] = [
		{
			label: 'Przychód brutto',
			value: (b) => fmt(b.grossMonthly),
			delta: (b, base) => b.grossMonthly - base.grossMonthly
		},
		{ label: 'ZUS', value: (b) => fmt(b.zusEmployee) },
		{ label: 'Składka zdrowotna', value: (b) => fmt(b.healthInsurance) },
		{ label: 'Podatek PIT', value: (b) => fmt(b.pit) },
		{ label: 'PPK pracownik', value: (b) => fmt(b.ppkEmployee) },
		{ label: 'Koszty dodatkowe', value: (b) => fmt(b.accountingCost) },
		{
			label: 'Netto miesięcznie',
			value: (b) => fmt(b.netMonthly),
			delta: (b, base) => b.netMonthly - base.netMonthly,
			bold: true
		},
		{
			label: 'Netto rocznie',
			value: (b) => fmt(b.netAnnual),
			delta: (b, base) => b.netAnnual - base.netAnnual
		},
		{ label: 'RSU po podatku (rocznie)', value: (b) => fmt(b.rsuAfterTax) },
		{
			label: 'Łączne roczne',
			value: (b) => fmt(b.totalAnnual),
			delta: (b, base) => b.totalAnnual - base.totalAnnual,
			bold: true
		},
		{ label: 'Koszt pracodawcy', value: (b) => fmt(b.employerCost) },
		{ label: 'Efektywna stawka podatkowa', value: (b) => `${b.effectiveTaxRate.toFixed(1)}%` },
		{ label: 'Ekwiwalent urlopowy', value: (b) => fmt(b.vacationEquivalent) }
	];

	$: baseline = results?.find((r) => r.isCurrentJob) ?? null;

	function fmt(v: number): string {
		return v.toLocaleString('pl-PL', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	function contractLabel(type: ContractType): string {
		switch (type) {
			case ContractType.UOP:
				return 'UoP';
			case ContractType.B2B_MONTHLY:
				return 'B2B';
			case ContractType.B2B_HOURLY:
				return 'B2B/h';
		}
	}

	function escapeHtml(str: string): string {
		return str
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/"/g, '&quot;');
	}

	function renderChart() {
		if (!results || !chartContainer) return;
		if (!chart) chart = echarts.init(chartContainer);

		const option: EChartsOption = {
			tooltip: {
				trigger: 'axis',
				formatter: (params: unknown) => {
					const items = params as { name: string; value: number; seriesName: string }[];
					let html = `<strong>${escapeHtml(items[0].name)}</strong><br/>`;
					for (const p of items) {
						html += `${escapeHtml(p.seriesName)}: ${fmt(p.value)} PLN<br/>`;
					}
					return html;
				}
			},
			xAxis: { type: 'category', data: results.map((r) => r.name) },
			yAxis: {
				type: 'value',
				name: 'PLN',
				axisLabel: {
					formatter: (v: number) => `${(v / 1000).toFixed(0)}k`
				}
			},
			series: [
				{
					name: 'Netto miesięcznie',
					type: 'bar',
					data: results.map((r) => Math.round(r.netMonthly)),
					itemStyle: { color: '#5E81AC' },
					label: { show: true, position: 'top', formatter: '{c}' }
				}
			]
		};
		chart.setOption(option, true);
	}

	onMount(() => {
		const handleResize = () => chart?.resize();
		if (browser) window.addEventListener('resize', handleResize);
		return () => {
			if (browser) window.removeEventListener('resize', handleResize);
			chart?.dispose();
		};
	});
</script>

<svelte:head>
	<title>Kalkulator wynagrodzeń</title>
</svelte:head>

<div class="calculator-page">
	<h1>Kalkulator wynagrodzeń</h1>

	<div class="content">
		<div class="form-section">
			<h2>Oferty</h2>

			{#each offers as offer, i}
				<div class="offer-card">
					<div class="offer-header">
						<input
							type="text"
							class="name-input"
							bind:value={offer.name}
							placeholder="Nazwa oferty"
						/>
						{#if offers.length > 1}
							<button class="remove-btn" on:click={() => removeOffer(i)}>✕</button>
						{/if}
					</div>

					<div class="form-group">
						<label>
							Typ umowy
							<select bind:value={offer.contractType}>
								<option value={ContractType.UOP}>UoP (umowa o pracę)</option>
								<option value={ContractType.B2B_MONTHLY}>B2B miesięcznie</option>
								<option value={ContractType.B2B_HOURLY}>B2B godzinowo</option>
							</select>
						</label>

						{#if offer.contractType === ContractType.UOP}
							<label>
								Brutto miesięcznie (PLN)
								<input type="number" bind:value={offer.grossMonthly} min="0" step="500" />
							</label>
							<label class="checkbox-label">
								<input type="checkbox" bind:checked={offer.ppkEnabled} />
								PPK (2% pracownik + 1.5% pracodawca)
							</label>
						{:else if offer.contractType === ContractType.B2B_MONTHLY}
							<label>
								Kwota na fakturze / przychód (brutto, PLN)
								<input type="number" bind:value={offer.netInvoice} min="0" step="500" />
							</label>
						{:else}
							<label>
								Stawka godzinowa (PLN)
								<input type="number" bind:value={offer.hourlyRate} min="0" step="10" />
							</label>
							<label>
								Godziny / miesiąc
								<input type="number" bind:value={offer.hoursPerMonth} min="1" step="1" />
							</label>
						{/if}

						{#if isB2B(offer.contractType)}
							<label>
								Forma opodatkowania
								<select bind:value={offer.taxForm}>
									<option value={B2BTaxForm.LINIOWY}>Liniowy (19%)</option>
									<option value={B2BTaxForm.RYCZALT}>Ryczałt IT (12%)</option>
									<option value={B2BTaxForm.SKALA}>Skala podatkowa (12/32%)</option>
								</select>
							</label>
							<label>
								ZUS
								<select bind:value={offer.zusTier}>
									<option value={ZUSTier.ULGA}>Ulga na start (0 PLN)</option>
									<option value={ZUSTier.PREFERENCYJNY}>Preferencyjny (456 PLN)</option>
									<option value={ZUSTier.PELNY}>Pełny (1 927 PLN)</option>
								</select>
							</label>
							<label>
								Koszt księgowości (PLN/mies.)
								<input type="number" bind:value={offer.accountingCost} min="0" step="50" />
							</label>
						{/if}

						<label>
							RSU roczne (PLN, opcjonalnie)
							<input type="number" bind:value={offer.rsuAnnual} min="0" step="1000" />
						</label>

						<label class="checkbox-label">
							<input
								type="checkbox"
								checked={offer.isCurrentJob}
								on:change={() => toggleCurrentJob(i)}
							/>
							Obecna praca
							{#if data.latestSalaries.length === 0}
								<small class="hint">brak danych o wynagrodzeniu</small>
							{/if}
						</label>
						{#if offer.isCurrentJob && data.latestSalaries.length > 1}
							<label>
								Źródło danych
								<select
									on:change={(e) => {
										const sal = data.latestSalaries[Number(e.currentTarget.value)];
										const contractType = mapContractType(sal.contract_type);
										if (contractType === null) return;
										offer.name = 'Aktualna praca';
										offer.contractType = contractType;
										if (offer.contractType === ContractType.UOP) {
											offer.grossMonthly = sal.gross_amount;
										} else {
											offer.netInvoice = sal.gross_amount;
										}
										offers = [...offers];
									}}
								>
									{#each data.latestSalaries as sal, si}
										<option value={si}>{sal.owner} — {sal.company}</option>
									{/each}
								</select>
							</label>
						{/if}
					</div>
				</div>
			{/each}

			<div class="form-actions">
				<button class="secondary-button" on:click={addOffer}>+ Dodaj ofertę</button>
				<button class="primary-button" on:click={compare}>Porównaj</button>
			</div>
		</div>

		{#if results && winner}
			<div class="results-section">
				<h2>Wyniki</h2>

				<div class="winner-banner">
					🏆 {winner.name}
					<span class="winner-net">{fmt(winner.netMonthly)} PLN netto/mies.</span>
				</div>

				<div class="chart-container" bind:this={chartContainer}></div>

				<div class="table-wrapper">
					<table>
						<thead>
							<tr>
								<th></th>
								{#each results as r}
									<th class:current={r.isCurrentJob}>
										{r.name}
										<small>{contractLabel(r.contractType)}</small>
									</th>
								{/each}
							</tr>
						</thead>
						<tbody>
							{#each tableRows as row}
								<tr class:bold-row={row.bold}>
									<td class="row-label">{row.label}</td>
									{#each results as r}
										<td class:highlight={row.bold && r === winner}>
											{row.value(r)}
											{#if baseline && row.delta && !r.isCurrentJob}
												{@const d = row.delta(r, baseline)}
												{#if d !== 0}
													<span class="delta" class:positive={d > 0} class:negative={d < 0}>
														{d > 0 ? '+' : ''}{fmt(d)}
													</span>
												{/if}
											{/if}
										</td>
									{/each}
								</tr>
							{/each}
							{#if breakEvens.size > 0}
								<tr class="break-even-row">
									<td class="row-label">Min. kwota do wyrównania</td>
									{#each results as r, i}
										<td>
											{#if r.isCurrentJob}
												—
											{:else if breakEvens.has(i)}
												{fmt(breakEvens.get(i) ?? 0)}
												<small class="break-even-hint">{breakEvenLabel(r.contractType)}</small>
											{/if}
										</td>
									{/each}
								</tr>
							{/if}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.calculator-page {
		padding: var(--size-4);
		max-width: 1400px;
		margin: 0 auto;
	}

	h1 {
		margin-bottom: var(--size-6);
		color: var(--color-text-1);
	}

	.content {
		display: grid;
		grid-template-columns: 400px 1fr;
		gap: var(--size-6);
		align-items: start;
	}

	.form-section,
	.results-section {
		background: var(--surface-2);
		padding: var(--size-5);
		border-radius: var(--radius-2);
	}

	h2 {
		margin-top: 0;
		margin-bottom: var(--size-4);
		color: var(--color-text-2);
	}

	.offer-card {
		background: var(--surface-1);
		padding: var(--size-4);
		border-radius: var(--radius-2);
		margin-bottom: var(--size-3);
		border: 1px solid var(--surface-4);
	}

	.offer-header {
		display: flex;
		gap: var(--size-2);
		align-items: center;
		margin-bottom: var(--size-3);
	}

	.name-input {
		flex: 1;
		padding: var(--size-2);
		border: 1px solid var(--surface-4);
		border-radius: var(--radius-2);
		background: var(--surface-2);
		color: var(--color-text-1);
		font-size: var(--font-size-2);
		font-weight: 600;
	}

	.remove-btn {
		padding: var(--size-1) var(--size-2);
		background: none;
		border: 1px solid var(--surface-4);
		border-radius: var(--radius-2);
		color: var(--color-text-3);
		cursor: pointer;
		font-size: var(--font-size-1);
	}

	.remove-btn:hover {
		color: var(--color-error);
		border-color: var(--color-error);
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: var(--size-3);
	}

	label {
		display: flex;
		flex-direction: column;
		gap: var(--size-1);
		font-size: var(--font-size-1);
		color: var(--color-text-2);
	}

	input[type='number'],
	select {
		padding: var(--size-2);
		border: 1px solid var(--surface-4);
		border-radius: var(--radius-2);
		background: var(--surface-1);
		color: var(--color-text-1);
		font-size: var(--font-size-1);
	}

	.checkbox-label {
		flex-direction: row;
		align-items: center;
		gap: var(--size-2);
		font-weight: 600;
	}

	.checkbox-label input[type='checkbox'] {
		width: 1rem;
		height: 1rem;
		cursor: pointer;
	}

	.form-actions {
		display: flex;
		gap: var(--size-3);
		margin-top: var(--size-3);
	}

	.primary-button {
		flex: 2;
		padding: var(--size-3);
		background: var(--color-primary);
		color: white;
		border: none;
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
		font-weight: 600;
		cursor: pointer;
	}

	.primary-button:hover {
		background: var(--color-primary-hover);
	}

	.secondary-button {
		flex: 1;
		padding: var(--size-3);
		background: var(--surface-3);
		color: var(--color-text-2);
		border: 1px solid var(--surface-4);
		border-radius: var(--radius-2);
		font-size: var(--font-size-1);
		font-weight: 600;
		cursor: pointer;
	}

	.secondary-button:hover {
		background: var(--surface-4);
	}

	.winner-banner {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: var(--size-4);
		border-radius: var(--radius-2);
		margin-bottom: var(--size-4);
		font-weight: 700;
		font-size: var(--font-size-3);
		background: hsl(92 30% 80%);
		color: var(--color-text-1);
	}

	.winner-net {
		font-size: var(--font-size-1);
		font-weight: 400;
	}

	.chart-container {
		width: 100%;
		height: 300px;
		margin-bottom: var(--size-5);
	}

	.table-wrapper {
		overflow-x: auto;
	}

	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--font-size-0);
	}

	th,
	td {
		padding: var(--size-2);
		text-align: right;
		border-bottom: 1px solid var(--surface-4);
		white-space: nowrap;
	}

	th {
		background: var(--surface-4);
		font-weight: 600;
		color: var(--color-text-2);
	}

	th small {
		display: block;
		font-weight: 400;
		font-size: var(--font-size-0);
		color: var(--color-text-3);
	}

	th.current {
		border-bottom: 2px solid var(--color-primary);
	}

	td:first-child,
	th:first-child {
		text-align: left;
	}

	.row-label {
		font-weight: 500;
		color: var(--color-text-2);
	}

	.bold-row td {
		font-weight: 700;
		color: var(--color-text-1);
	}

	.highlight {
		background: hsl(92 30% 90%);
	}

	.break-even-row td {
		border-top: 2px solid var(--color-primary);
		font-weight: 600;
		color: var(--color-primary);
	}

	.break-even-hint {
		display: block;
		font-weight: 400;
		font-size: var(--font-size-0);
		color: var(--color-text-3);
	}

	.hint {
		font-weight: 400;
		color: var(--color-text-3);
	}

	.delta {
		display: block;
		font-size: var(--font-size-0);
		font-weight: 400;
	}

	.delta.positive {
		color: hsl(130 50% 35%);
	}

	.delta.negative {
		color: hsl(0 60% 45%);
	}

	@media (max-width: 1100px) {
		.content {
			grid-template-columns: 1fr;
		}
	}
</style>
