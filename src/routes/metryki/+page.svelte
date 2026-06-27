<script lang="ts">
	import MetricCard from '$lib/components/MetricCard.svelte';
	import { onMount } from 'svelte';
	import DateRangePicker from '$lib/components/DateRangePicker.svelte';
	import { TrendingUp, CheckCircle2, Lightbulb } from 'lucide-svelte';
	import {
		buildInvestmentTrendChartOption,
		buildWrapperTrendChartOption,
		buildYearlyRoiChartOption
	} from '$lib/utils/charts/metryki';
	import { buildCumulativeInflationChartOption } from '$lib/utils/charts/inflation';
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import { applyMobileChartTweaks } from '$lib/utils/charts/responsive';
	import { isMobile } from '$lib/utils/viewport';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import { formatDate } from '$lib/utils/format';
	import ContributionAdjustedReturns from '$lib/components/ContributionAdjustedReturns.svelte';
	import RealYieldsTable from '$lib/components/RealYieldsTable.svelte';

	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const allocationAnalysis = $derived(data.allocationAnalysis);
	const investmentTimeSeries = $derived(data.investmentTimeSeries);
	const wrapperTimeSeries = $derived(data.wrapperTimeSeries);
	const categoryTimeSeries = $derived(data.categoryTimeSeries);
	const realYieldAccounts = $derived(data.realYieldAccounts ?? []);
	const cpiSeries = $derived(data.cpiSeries);
	const hasCpiSeries = $derived((cpiSeries?.points?.length ?? 0) > 0);
	const ikzePitStats = $derived(data.ikzePitStats ?? []);
	const snapshotDate = $derived(data.snapshotDate ?? null);

	// The page is one long scroll across five themes. A sticky anchor nav lets
	// the user jump between them. Anchors (not tabs) keep every section — and
	// the always-rendered chart divs ECharts is bound to — in the DOM, so a
	// tab's display:none can't collapse a canvas to zero size.
	const sections = [
		{ id: 'dzialania', label: 'Działania' },
		{ id: 'konta', label: 'Konta' },
		{ id: 'zwroty', label: 'Zwroty' },
		{ id: 'wzrost', label: 'Wzrost' }
	];

	let inflationChart = $state<HTMLDivElement | undefined>(undefined);
	let investmentTrendChart: HTMLDivElement;
	let ikeChart: HTMLDivElement;
	let ikzeChart: HTMLDivElement;
	let ppkChart: HTMLDivElement;
	let stockChart: HTMLDivElement;
	let bondChart: HTMLDivElement;
	let yearlyRoiChart: HTMLDivElement;

	// Chart instances are created once (the divs are always rendered) and kept;
	// a date-range refresh only re-applies options via setOption, so ECharts
	// reflows the existing canvas instead of disposing + recreating the fleet.
	let handles: Record<string, ChartHandle> = {};
	let chartsReady = $state(false);
	let inflationHandle: ChartHandle | undefined;

	onMount(() => {
		handles = {
			investmentTrend: createChart(investmentTrendChart),
			ike: createChart(ikeChart),
			ikze: createChart(ikzeChart),
			ppk: createChart(ppkChart),
			stock: createChart(stockChart),
			bond: createChart(bondChart),
			yearlyRoi: createChart(yearlyRoiChart)
		};
		chartsReady = true;
		return () => {
			for (const handle of Object.values(handles)) handle.dispose();
			handles = {};
			chartsReady = false;
			// The inflation chart has its own effect-driven lifecycle, but its
			// ref doesn't reliably flip to undefined on whole-component teardown,
			// so dispose it here too to avoid leaking the instance + observer.
			inflationHandle?.dispose();
			inflationHandle = undefined;
		};
	});

	// Re-apply options whenever the underlying series change (reflow, no remount).
	$effect(() => {
		if (!chartsReady) return;
		handles.investmentTrend.chart.setOption(buildInvestmentTrendChartOption(investmentTimeSeries));
		handles.ike.chart.setOption(
			buildWrapperTrendChartOption('IKE w czasie', wrapperTimeSeries.ike)
		);
		handles.ikze.chart.setOption(
			buildWrapperTrendChartOption('IKZE w czasie', wrapperTimeSeries.ikze)
		);
		handles.ppk.chart.setOption(
			buildWrapperTrendChartOption('PPK w czasie', wrapperTimeSeries.ppk)
		);
		handles.stock.chart.setOption(
			buildWrapperTrendChartOption('Akcje w czasie', categoryTimeSeries.stock)
		);
		handles.bond.chart.setOption(
			buildWrapperTrendChartOption('Obligacje w czasie', categoryTimeSeries.bond)
		);
		handles.yearlyRoi.chart.setOption(
			buildYearlyRoiChartOption(
				categoryTimeSeries.stock,
				categoryTimeSeries.bond,
				wrapperTimeSeries.ppk
			)
		);
	});

	// The inflation chart's container is conditional, so it has its own
	// create-once / dispose-when-gone lifecycle keyed on the ref + data.
	$effect(() => {
		if (!hasCpiSeries || !inflationChart) {
			inflationHandle?.dispose();
			inflationHandle = undefined;
			return;
		}
		if (!inflationHandle) inflationHandle = createChart(inflationChart);
		inflationHandle.chart.setOption(
			applyMobileChartTweaks(buildCumulativeInflationChartOption(cpiSeries), $isMobile)
		);
	});
</script>

<svelte:head>
	<title>Metryki - Finance Buddy</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="h1">Metryki</h1>

	<nav
		class="sticky top-14 md:top-0 z-10 -mx-2 px-2 py-2 flex gap-2 overflow-x-auto whitespace-nowrap bg-surface-50-950/80 backdrop-blur"
		aria-label="Sekcje strony"
	>
		{#each sections as section}
			<a href="#{section.id}" class="btn btn-sm preset-tonal-surface shrink-0">{section.label}</a>
		{/each}
	</nav>

	<DateRangePicker />

	<section id="dzialania" class="scroll-mt-24 md:scroll-mt-16 space-y-6">
		<h2 class="h2">Jak inwestować nowe pieniądze</h2>

		{#if allocationAnalysis.rebalancing.length > 0}
			<div class="card preset-filled-surface-100-900 p-5">
				<p class="mb-4 text-surface-600-400">
					Aby osiągnąć docelową alokację portfela, wpłać nowe środki w następujący sposób:
				</p>

				<div class="flex flex-col gap-3 mb-4">
					{#each allocationAnalysis.rebalancing as suggestion}
						<div
							class="flex items-center gap-4 p-3 rounded-container border border-success-500 bg-success-500/10"
						>
							<span class="font-bold min-w-[120px] text-success-500 inline-flex items-center gap-1"
								><TrendingUp size={14} /> KUP</span
							>
							<span class="flex-1 capitalize">{suggestion.category}</span>
							<span class="font-bold">
								{suggestion.amount.toLocaleString('pl-PL', {
									minimumFractionDigits: 0,
									maximumFractionDigits: 0
								})} PLN
							</span>
						</div>
					{/each}
				</div>

				<p class="italic text-surface-600-400 mt-4 inline-flex items-center gap-1">
					<Lightbulb size={14} /> Całkowita wartość portfela inwestycyjnego: {allocationAnalysis.total_investment_value.toLocaleString(
						'pl-PL',
						{ minimumFractionDigits: 0, maximumFractionDigits: 0 }
					)} PLN
				</p>
			</div>
		{:else}
			<div
				class="card preset-filled-surface-100-900 p-4 flex items-center gap-2 text-success-500 font-semibold"
			>
				<CheckCircle2 size={16} /> Portfel jest zgodny z docelową alokacją (różnice mniejsze niż 1%)
			</div>
		{/if}
	</section>

	<section id="konta" class="scroll-mt-24 md:scroll-mt-16 space-y-6">
		<!-- PPK Stats Section -->
		{#if data.ppkStats && data.ppkStats.length > 0}
			<h2 class="h2">Podsumowanie PPK</h2>
			{#each data.ppkStats as ppkStat}
				<h3 class="h4 font-semibold mt-4 mb-3">
					{ownerName((data.owners ?? []) as OwnerOption[], ppkStat.owner_user_id)}
				</h3>
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					<MetricCard
						label="PPK - Wartość całkowita"
						value={ppkStat.total_value}
						decimals={0}
						suffix=" PLN"
						color="green"
					/>

					<MetricCard
						label="PPK - Wpłaty pracownika"
						value={ppkStat.employee_contributed}
						decimals={0}
						suffix=" PLN"
						color="blue"
					/>

					<MetricCard
						label="PPK - Wpłaty pracodawcy"
						value={ppkStat.employer_contributed}
						decimals={0}
						suffix=" PLN"
						color="blue"
					/>

					<MetricCard
						label="PPK - Dopłaty państwa"
						value={ppkStat.government_contributed}
						decimals={0}
						suffix=" PLN"
						color="blue"
					/>

					<MetricCard
						label="PPK - Łącznie wpłacone"
						value={ppkStat.total_contributed}
						decimals={0}
						suffix=" PLN"
						color="blue"
					/>

					<MetricCard
						label="PPK - Zyski z inwestycji"
						value={ppkStat.returns}
						decimals={0}
						suffix=" PLN"
						color={ppkStat.returns >= 0 ? 'green' : 'red'}
					/>

					<MetricCard
						label="PPK - ROI"
						value={ppkStat.roi_percentage}
						decimals={2}
						suffix="%"
						color={ppkStat.roi_percentage >= 0 ? 'green' : 'red'}
					/>
				</div>
			{/each}
		{/if}

		<!-- IKZE PIT Savings Section -->
		{#if ikzePitStats.length > 0}
			<h2 class="h2">Korzyść podatkowa IKZE ({ikzePitStats[0].year})</h2>
			<p class="text-sm text-surface-600-400">
				Wpłaty na IKZE odliczasz od podstawy opodatkowania. Szacunek na podstawie krańcowej stawki
				PIT z ostatniej pensji.
			</p>
			{#each ikzePitStats as pit}
				<h3 class="h4 font-semibold mt-4 mb-3">
					{ownerName((data.owners ?? []) as OwnerOption[], pit.owner_user_id)}
				</h3>
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					<MetricCard
						label="IKZE - Wpłacono w tym roku"
						value={pit.total_contributed}
						decimals={0}
						suffix=" PLN"
						color="blue"
					/>

					<MetricCard
						label="IKZE - Krańcowa stawka PIT"
						value={pit.marginal_tax_rate == null ? null : pit.marginal_tax_rate * 100}
						decimals={0}
						suffix="%"
						color="blue"
					/>

					<MetricCard
						label="IKZE - Szacowana ulga PIT"
						value={pit.pit_savings}
						decimals={0}
						suffix=" PLN"
						color="green"
					/>
				</div>
			{/each}
		{/if}

		<!-- Stock Stats Section -->
		{#if data.stockStats}
			<h2 class="h2">Podsumowanie Akcji</h2>
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				<MetricCard
					label="Akcje - Wartość całkowita"
					value={data.stockStats.total_value}
					decimals={0}
					suffix=" PLN"
					color="green"
				/>

				<MetricCard
					label="Akcje - Łącznie wpłacone"
					value={data.stockStats.total_contributed}
					decimals={0}
					suffix=" PLN"
					color="blue"
				/>

				<MetricCard
					label="Akcje - Zyski z inwestycji"
					value={data.stockStats.returns}
					decimals={0}
					suffix=" PLN"
					color={data.stockStats.returns >= 0 ? 'green' : 'red'}
				/>

				<MetricCard
					label="Akcje - ROI"
					value={data.stockStats.roi_percentage}
					decimals={2}
					suffix="%"
					color={data.stockStats.roi_percentage >= 0 ? 'green' : 'red'}
				/>
			</div>
		{/if}

		<!-- Bond Stats Section -->
		{#if data.bondStats}
			<h2 class="h2">Podsumowanie Obligacji</h2>
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				<MetricCard
					label="Obligacje - Wartość całkowita"
					value={data.bondStats.total_value}
					decimals={0}
					suffix=" PLN"
					color="green"
				/>

				<MetricCard
					label="Obligacje - Łącznie wpłacone"
					value={data.bondStats.total_contributed}
					decimals={0}
					suffix=" PLN"
					color="blue"
				/>

				<MetricCard
					label="Obligacje - Zyski z inwestycji"
					value={data.bondStats.returns}
					decimals={0}
					suffix=" PLN"
					color={data.bondStats.returns >= 0 ? 'green' : 'red'}
				/>

				<MetricCard
					label="Obligacje - ROI"
					value={data.bondStats.roi_percentage}
					decimals={2}
					suffix="%"
					color={data.bondStats.roi_percentage >= 0 ? 'green' : 'red'}
				/>
			</div>
		{/if}
	</section>

	<section id="zwroty" class="scroll-mt-24 md:scroll-mt-16 space-y-6">
		<h2 class="h2">Zwroty skorygowane o wpłaty</h2>
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
			<ContributionAdjustedReturns scope={{ type: 'all' }} title="Gospodarstwo" />
			<ContributionAdjustedReturns scope={{ type: 'category', value: 'stock' }} title="Akcje" />
			<ContributionAdjustedReturns scope={{ type: 'category', value: 'bond' }} title="Obligacje" />
			<ContributionAdjustedReturns scope={{ type: 'wrapper', value: 'IKE' }} title="IKE" />
			<ContributionAdjustedReturns scope={{ type: 'wrapper', value: 'IKZE' }} title="IKZE" />
			<ContributionAdjustedReturns scope={{ type: 'wrapper', value: 'PPK' }} title="PPK" />
		</div>

		<h2 class="h2">Realne zwroty (po inflacji)</h2>

		<RealYieldsTable accounts={realYieldAccounts} />

		{#if hasCpiSeries}
			<div class="card preset-filled-surface-100-900 p-4 mt-4">
				<div bind:this={inflationChart} class="w-full h-[320px] sm:h-[440px]"></div>
			</div>
		{/if}

		<div
			class="card preset-tonal-surface p-4 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4"
		>
			<div>
				<h3 class="font-bold">Ekspozycja walutowa</h3>
				<p class="text-sm text-surface-600-400">
					Zobacz szczegółową analizę alokacji walutowej i geograficznej portfela.
				</p>
			</div>
			<a href="/ekspozycja" class="btn preset-filled-primary-500 btn-sm whitespace-nowrap"
				>Otwórz ekspozycję</a
			>
		</div>
	</section>

	<section id="wzrost" class="scroll-mt-24 md:scroll-mt-16 space-y-6">
		<h2 class="h2">Wzrost inwestycji w czasie</h2>

		<div class="card preset-filled-surface-100-900 p-4 mb-8">
			<div bind:this={investmentTrendChart} class="w-full h-[320px] sm:h-[500px]"></div>
		</div>

		<h2 class="h2">Wzrost według typu konta</h2>

		<div class="grid grid-cols-1 gap-4 mb-8">
			<div class="card preset-filled-surface-100-900 p-4">
				<div bind:this={ikeChart} class="w-full h-[280px] sm:h-[400px]"></div>
			</div>
			<div class="card preset-filled-surface-100-900 p-4">
				<div bind:this={ikzeChart} class="w-full h-[280px] sm:h-[400px]"></div>
			</div>
			<div class="card preset-filled-surface-100-900 p-4">
				<div bind:this={ppkChart} class="w-full h-[280px] sm:h-[400px]"></div>
			</div>
		</div>

		<h2 class="h2">Wzrost według typu inwestycji</h2>

		<div class="grid grid-cols-1 gap-4 mb-8">
			<div class="card preset-filled-surface-100-900 p-4">
				<div bind:this={stockChart} class="w-full h-[280px] sm:h-[400px]"></div>
			</div>
			<div class="card preset-filled-surface-100-900 p-4">
				<div bind:this={bondChart} class="w-full h-[280px] sm:h-[400px]"></div>
			</div>
		</div>

		<h2 class="h2">Roczny ROI według klasy aktywów</h2>

		<div class="card preset-filled-surface-100-900 p-4 mb-8">
			<div bind:this={yearlyRoiChart} class="w-full h-[320px] sm:h-[500px]"></div>
		</div>
	</section>
</div>
