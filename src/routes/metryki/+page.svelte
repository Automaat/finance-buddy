<script lang="ts">
	import { onMount } from 'svelte';
	import MetricCard from '$lib/components/MetricCard.svelte';
	import DateRangePicker from '$lib/components/DateRangePicker.svelte';
	import { TrendingUp, CheckCircle2, Lightbulb } from 'lucide-svelte';
	import {
		buildAllocationChartOption,
		buildWrapperChartOption,
		buildInvestmentTrendChartOption,
		buildWrapperTrendChartOption,
		buildYearlyRoiChartOption
	} from '$lib/utils/charts/metryki';
	import { buildCumulativeInflationChartOption } from '$lib/utils/charts/inflation';
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import ContributionAdjustedReturns from '$lib/components/ContributionAdjustedReturns.svelte';
	import CurrencyExposureWidget from '$lib/components/CurrencyExposureWidget.svelte';
	import RealYieldsTable from '$lib/components/RealYieldsTable.svelte';

	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const metricCards = $derived(data.metricCards);
	const allocationAnalysis = $derived(data.allocationAnalysis);
	const investmentTimeSeries = $derived(data.investmentTimeSeries);
	const wrapperTimeSeries = $derived(data.wrapperTimeSeries);
	const categoryTimeSeries = $derived(data.categoryTimeSeries);
	const realYieldAccounts = $derived(data.realYieldAccounts ?? []);
	const cpiSeries = $derived(data.cpiSeries);
	const hasCpiSeries = $derived((cpiSeries?.points?.length ?? 0) > 0);

	let allocationChart: HTMLDivElement;
	let inflationChart = $state<HTMLDivElement | undefined>(undefined);
	let wrapperChart: HTMLDivElement;
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
			allocation: createChart(allocationChart),
			wrapper: createChart(wrapperChart),
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
		handles.allocation.chart.setOption(buildAllocationChartOption(allocationAnalysis.by_category));
		handles.wrapper.chart.setOption(buildWrapperChartOption(allocationAnalysis.by_wrapper));
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
		inflationHandle.chart.setOption(buildCumulativeInflationChartOption(cpiSeries));
	});
</script>

<svelte:head>
	<title>Metryki - Finance Buddy</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="h1">Metryki</h1>

	<DateRangePicker />

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

	<h2 class="h2">Przegląd finansowy</h2>

	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
		<MetricCard
			label="Ile metrów mieszkania jest nasze"
			value={metricCards.property_sqm}
			decimals={2}
			suffix=" m²"
			color="blue"
		/>

		<MetricCard
			label="Ile miesięcy bez pracy"
			value={metricCards.emergency_fund_months}
			decimals={2}
			color="green"
		/>

		<MetricCard
			label="Pensja z odsetek"
			value={metricCards.retirement_income_monthly}
			decimals={2}
			suffix=" PLN"
			color="blue"
		/>

		<MetricCard
			label="Ile zostało do spłaty hipoteki"
			value={metricCards.mortgage_remaining}
			decimals={0}
			suffix=" PLN"
			color="red"
		/>

		<MetricCard
			label="Ile miesięcy do spłaty hipoteki"
			value={metricCards.mortgage_months_left}
			decimals={0}
			color="red"
		/>

		<MetricCard
			label="Ile lat do spłaty hipoteki"
			value={metricCards.mortgage_years_left}
			decimals={2}
			color="red"
		/>

		<MetricCard
			label="Ile oszczędności emerytalnych"
			value={metricCards.retirement_total}
			decimals={0}
			suffix=" PLN"
			color="green"
		/>

		<MetricCard
			label="Ile wpłaciliśmy na inwestycje"
			value={metricCards.investment_contributions}
			decimals={0}
			suffix=" PLN"
			color="blue"
		/>

		<MetricCard
			label="Ile zarobiliśmy na inwestycjach"
			value={metricCards.investment_returns}
			decimals={0}
			suffix=" PLN"
			color="green"
		/>

		{#if metricCards.savings_rate !== null}
			<MetricCard
				label="Ile oszczędzamy miesięcznie"
				value={metricCards.savings_rate}
				decimals={1}
				suffix="%"
				color="green"
			/>
		{/if}

		{#if metricCards.debt_to_income_ratio !== null}
			<MetricCard
				label="Stosunek długu do dochodu"
				value={metricCards.debt_to_income_ratio}
				decimals={1}
				suffix="%"
				color={metricCards.debt_to_income_ratio < 30
					? 'green'
					: metricCards.debt_to_income_ratio <= 36
						? 'blue'
						: 'red'}
			/>
		{/if}

		{#if metricCards.hour_of_work_cost !== null}
			<MetricCard
				label="Koszt godziny pracy"
				value={metricCards.hour_of_work_cost}
				decimals={2}
				suffix=" PLN"
				color="blue"
			/>
		{/if}

		{#if metricCards.hour_of_life_cost !== null}
			<MetricCard
				label="Koszt godziny życia"
				value={metricCards.hour_of_life_cost}
				decimals={2}
				suffix=" PLN"
				color="green"
			/>
		{/if}
	</div>

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

	<h2 class="h2">Zwroty skorygowane o wpłaty</h2>
	<div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
		<ContributionAdjustedReturns scope={{ type: 'all' }} title="Gospodarstwo" />
		<ContributionAdjustedReturns scope={{ type: 'category', value: 'stock' }} title="Akcje" />
		<ContributionAdjustedReturns scope={{ type: 'category', value: 'bond' }} title="Obligacje" />
	</div>

	<h2 class="h2">Realne zwroty (po inflacji)</h2>

	<RealYieldsTable accounts={realYieldAccounts} />

	{#if hasCpiSeries}
		<div class="card preset-filled-surface-100-900 p-4 mt-4">
			<div bind:this={inflationChart} class="w-full h-[320px] sm:h-[440px]"></div>
		</div>
	{/if}

	<h2 class="h2">Struktura portfela inwestycyjnego</h2>

	<div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
		<div class="card preset-filled-surface-100-900 p-4">
			<div bind:this={allocationChart} class="w-full h-[280px] sm:h-[400px]"></div>
		</div>

		<div class="card preset-filled-surface-100-900 p-4">
			<div bind:this={wrapperChart} class="w-full h-[280px] sm:h-[400px]"></div>
		</div>
	</div>

	<CurrencyExposureWidget />

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
</div>
