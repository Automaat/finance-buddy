<script lang="ts">
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import { projectFireGap, type FireGapInputs, type PensionPriceBasis } from '$lib/utils/fireGap';
	import { formatPLN } from '$lib/utils/format';
	import { Scale } from 'lucide-svelte';

	// All numeric props are $bindable so the retirement page can share its
	// existing Monte Carlo inputs with this chart instead of asking the user
	// to type them twice.
	let {
		currentAge = $bindable(35),
		retirementAge = $bindable(65),
		lifeExpectancy = $bindable(85),
		currentPortfolioPLN = $bindable(100000),
		annualContributionPLN = $bindable(20000),
		expectedReturnPct = $bindable(6),
		inflationPct = $bindable(3),
		withdrawalRatePct = $bindable(4),
		monthlyPensionNetPLN = $bindable(3500),
		pensionBasis = $bindable<PensionPriceBasis>('today')
	}: {
		currentAge?: number;
		retirementAge?: number;
		lifeExpectancy?: number;
		currentPortfolioPLN?: number;
		annualContributionPLN?: number;
		expectedReturnPct?: number;
		inflationPct?: number;
		withdrawalRatePct?: number;
		monthlyPensionNetPLN?: number;
		pensionBasis?: PensionPriceBasis;
	} = $props();

	const inputs = $derived<FireGapInputs>({
		currentAge,
		retirementAge,
		lifeExpectancy,
		currentPortfolioPLN,
		annualContributionPLN,
		expectedReturnPct,
		inflationPct,
		withdrawalRatePct,
		monthlyPensionNetPLN,
		pensionBasis
	});

	const rows = $derived(projectFireGap(inputs));

	const summary = $derived.by(() => {
		const post = rows.filter((r) => r.afterRetirement);
		if (post.length === 0) return null;
		const totalGap = post.reduce((sum, r) => sum + r.gapPLN, 0);
		const avgGap = totalGap / post.length;
		const firstYearGap = post[0].gapPLN;
		return { avgGap, firstYearGap };
	});

	let container: HTMLDivElement | undefined = $state();
	let handle: ChartHandle | null = null;

	$effect(() => {
		if (!container) {
			handle?.dispose();
			handle = null;
			return undefined;
		}
		if (!handle) handle = createChart(container);
		handle.chart.setOption({
			tooltip: { trigger: 'axis' },
			legend: {
				top: 0,
				data: ['Prywatny portfel', 'Emerytura ZUS', 'Luka miesięczna']
			},
			grid: { left: 60, right: 30, top: 40, bottom: 30 },
			xAxis: {
				type: 'category',
				data: rows.map((r) => r.year)
			},
			yAxis: {
				type: 'value',
				axisLabel: {
					formatter: (v: number) => formatPLN(v)
				}
			},
			series: [
				{
					name: 'Prywatny portfel',
					type: 'line',
					smooth: true,
					data: rows.map((r) => Math.round(r.privateMonthlyIncomePLN)),
					itemStyle: { color: '#10b981' }
				},
				{
					name: 'Emerytura ZUS',
					type: 'line',
					smooth: true,
					data: rows.map((r) => Math.round(r.zusMonthlyIncomePLN)),
					itemStyle: { color: '#3b82f6' }
				},
				{
					name: 'Luka miesięczna',
					type: 'bar',
					barWidth: '40%',
					data: rows.map((r) => (r.afterRetirement ? Math.round(r.gapPLN) : 0)),
					itemStyle: { color: '#f59e0b', opacity: 0.65 }
				}
			]
		});
		// Dispose the chart + ResizeObserver when the component is destroyed
		// or the container ref is detached, so this widget can be re-mounted
		// without leaking either.
		return () => {
			handle?.dispose();
			handle = null;
		};
	});
</script>

<section class="card preset-filled-surface-100-900 p-5 space-y-4">
	<header class="space-y-1">
		<h2 class="h4 flex items-center gap-2">
			<Scale size={20} class="text-primary-500" />
			Luka FIRE vs ZUS
		</h2>
		<p class="text-sm text-surface-700-300">
			Miesięczna luka między prywatnym portfelem a szacunkową emeryturą ZUS — używa założeń
			kalkulatora ZUS.
		</p>
	</header>

	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
		<label class="space-y-1">
			<span class="text-xs font-semibold">Inflacja (%)</span>
			<input type="number" step="0.1" min="0" class="input w-full" bind:value={inflationPct} />
		</label>
		<label class="space-y-1">
			<span class="text-xs font-semibold">Wskaźnik wypłaty (%)</span>
			<input type="number" step="0.1" min="0" class="input w-full" bind:value={withdrawalRatePct} />
		</label>
		<label class="space-y-1">
			<span class="text-xs font-semibold">Mies. emerytura ZUS (PLN)</span>
			<input type="number" min="0" class="input w-full" bind:value={monthlyPensionNetPLN} />
		</label>
		<label class="space-y-1">
			<span class="text-xs font-semibold">Denominacja emerytury</span>
			<select bind:value={pensionBasis} class="input w-full">
				<option value="today">Dzisiejsze PLN</option>
				<option value="retirement">PLN w roku emerytury (kalkulator ZUS)</option>
			</select>
		</label>
	</div>

	{#if summary}
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
			<div class="card preset-tonal-surface p-3">
				<div class="text-xs text-surface-600-400">Luka w pierwszym roku emerytury</div>
				<div
					class="text-2xl font-bold {summary.firstYearGap >= 0
						? 'text-success-600-400'
						: 'text-error-600-400'}"
				>
					{summary.firstYearGap >= 0 ? '+' : ''}{formatPLN(summary.firstYearGap)}
				</div>
			</div>
			<div class="card preset-tonal-surface p-3">
				<div class="text-xs text-surface-600-400">
					Średnia miesięczna luka po przejściu na emeryturę
				</div>
				<div
					class="text-2xl font-bold {summary.avgGap >= 0
						? 'text-success-600-400'
						: 'text-error-600-400'}"
				>
					{summary.avgGap >= 0 ? '+' : ''}{formatPLN(summary.avgGap)}
				</div>
			</div>
		</div>
	{/if}

	<div bind:this={container} class="w-full h-[360px]"></div>

	<p class="text-xs text-surface-600-400">
		Słupek = miesięczna luka między prywatną wypłatą (portfel × stopa wypłaty / 12) a indeksowaną
		emeryturą ZUS. Dodatni słupek = prywatny portfel pokrywa więcej niż państwowa emerytura; ujemny
		= luka do zasypania.
	</p>
</section>
