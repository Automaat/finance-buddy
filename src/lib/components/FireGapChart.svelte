<script lang="ts">
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import { projectFireGap, type FireGapInputs } from '$lib/utils/fireGap';
	import { formatPLN } from '$lib/utils/format';
	import { Scale } from 'lucide-svelte';

	let currentAge = $state(35);
	let retirementAge = $state(65);
	let lifeExpectancy = $state(85);
	let currentPortfolioPLN = $state(100000);
	let annualContributionPLN = $state(20000);
	let expectedReturnPct = $state(6);
	let inflationPct = $state(3);
	let withdrawalRatePct = $state(4);
	let monthlyPensionNetPLN = $state(3500);

	const inputs = $derived<FireGapInputs>({
		currentAge,
		retirementAge,
		lifeExpectancy,
		currentPortfolioPLN,
		annualContributionPLN,
		expectedReturnPct,
		inflationPct,
		withdrawalRatePct,
		monthlyPensionNetPLN
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
			return;
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
	});
</script>

<section class="card preset-filled-surface-100-900 p-5 space-y-4">
	<header class="space-y-1">
		<h2 class="h4 flex items-center gap-2">
			<Scale size={20} class="text-primary-500" />
			Luka FIRE vs ZUS
		</h2>
		<p class="text-sm text-surface-700-300">
			Roczna luka między prywatnym portfelem a szacunkową emeryturą ZUS — używa założeń kalkulatora
			ZUS.
		</p>
	</header>

	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
		<label class="space-y-1">
			<span class="text-xs font-semibold">Obecny wiek</span>
			<input type="number" min="18" max="100" class="input w-full" bind:value={currentAge} />
		</label>
		<label class="space-y-1">
			<span class="text-xs font-semibold">Wiek emerytalny</span>
			<input type="number" min="18" max="100" class="input w-full" bind:value={retirementAge} />
		</label>
		<label class="space-y-1">
			<span class="text-xs font-semibold">Oczekiwana długość życia</span>
			<input type="number" min="18" max="120" class="input w-full" bind:value={lifeExpectancy} />
		</label>
		<label class="space-y-1">
			<span class="text-xs font-semibold">Obecny portfel (PLN)</span>
			<input type="number" min="0" class="input w-full" bind:value={currentPortfolioPLN} />
		</label>
		<label class="space-y-1">
			<span class="text-xs font-semibold">Roczna wpłata (PLN)</span>
			<input type="number" min="0" class="input w-full" bind:value={annualContributionPLN} />
		</label>
		<label class="space-y-1">
			<span class="text-xs font-semibold">Oczekiwana stopa zwrotu (%)</span>
			<input type="number" step="0.1" class="input w-full" bind:value={expectedReturnPct} />
		</label>
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
					Średnia roczna luka po przejściu na emeryturę
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
