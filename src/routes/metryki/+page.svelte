<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import MetricCard from '$lib/components/MetricCard.svelte';
	import { TrendingUp, CheckCircle2, Lightbulb } from 'lucide-svelte';
	import {
		buildAllocationChartOption,
		buildWrapperChartOption,
		buildInvestmentTrendChartOption,
		buildWrapperTrendChartOption,
		buildYearlyRoiChartOption
	} from '$lib/utils/charts/metryki';

	export let data;

	const {
		metricCards,
		allocationAnalysis,
		investmentTimeSeries,
		wrapperTimeSeries,
		categoryTimeSeries
	} = data;

	let allocationChart: HTMLDivElement;
	let wrapperChart: HTMLDivElement;
	let investmentTrendChart: HTMLDivElement;
	let ikeChart: HTMLDivElement;
	let ikzeChart: HTMLDivElement;
	let ppkChart: HTMLDivElement;
	let stockChart: HTMLDivElement;
	let bondChart: HTMLDivElement;
	let yearlyRoiChart: HTMLDivElement;

	onMount(() => {
		const allocationChartInstance = echarts.init(allocationChart);
		allocationChartInstance.setOption(buildAllocationChartOption(allocationAnalysis.by_category));

		const wrapperChartInstance = echarts.init(wrapperChart);
		wrapperChartInstance.setOption(buildWrapperChartOption(allocationAnalysis.by_wrapper));

		const investmentTrendChartInstance = echarts.init(investmentTrendChart);
		investmentTrendChartInstance.setOption(buildInvestmentTrendChartOption(investmentTimeSeries));

		const createWrapperChart = (
			chartElement: HTMLDivElement,
			title: string,
			series: Parameters<typeof buildWrapperTrendChartOption>[1]
		) => {
			const chartInstance = echarts.init(chartElement);
			chartInstance.setOption(buildWrapperTrendChartOption(title, series));
			return chartInstance;
		};

		const ikeChartInstance = createWrapperChart(ikeChart, 'IKE w czasie', wrapperTimeSeries.ike);
		const ikzeChartInstance = createWrapperChart(
			ikzeChart,
			'IKZE w czasie',
			wrapperTimeSeries.ikze
		);
		const ppkChartInstance = createWrapperChart(ppkChart, 'PPK w czasie', wrapperTimeSeries.ppk);
		const stockChartInstance = createWrapperChart(
			stockChart,
			'Akcje w czasie',
			categoryTimeSeries.stock
		);
		const bondChartInstance = createWrapperChart(
			bondChart,
			'Obligacje w czasie',
			categoryTimeSeries.bond
		);

		const yearlyRoiChartInstance = echarts.init(yearlyRoiChart);
		yearlyRoiChartInstance.setOption(
			buildYearlyRoiChartOption(
				categoryTimeSeries.stock,
				categoryTimeSeries.bond,
				wrapperTimeSeries.ppk
			)
		);

		return () => {
			allocationChartInstance.dispose();
			wrapperChartInstance.dispose();
			investmentTrendChartInstance.dispose();
			ikeChartInstance.dispose();
			ikzeChartInstance.dispose();
			ppkChartInstance.dispose();
			stockChartInstance.dispose();
			bondChartInstance.dispose();
			yearlyRoiChartInstance.dispose();
		};
	});
</script>

<svelte:head>
	<title>Metryki - Finance Buddy</title>
</svelte:head>

<div class="container">
	<h1>Metryki</h1>

	<h2>Jak inwestować nowe pieniądze</h2>

	{#if allocationAnalysis.rebalancing.length > 0}
		<div class="rebalancing-container">
			<p class="rebalancing-intro">
				Aby osiągnąć docelową alokację portfela, wpłać nowe środki w następujący sposób:
			</p>

			<div class="rebalancing-list">
				{#each allocationAnalysis.rebalancing as suggestion}
					<div class="rebalancing-item buy">
						<span class="action-label inline-flex items-center gap-1"
							><TrendingUp size={14} /> KUP</span
						>
						<span class="category-name">{suggestion.category}</span>
						<span class="amount">
							{suggestion.amount.toLocaleString('pl-PL', {
								minimumFractionDigits: 0,
								maximumFractionDigits: 0
							})} PLN
						</span>
					</div>
				{/each}
			</div>

			<p class="rebalancing-note inline-flex items-center gap-1">
				<Lightbulb size={14} /> Całkowita wartość portfela inwestycyjnego: {allocationAnalysis.total_investment_value.toLocaleString(
					'pl-PL',
					{ minimumFractionDigits: 0, maximumFractionDigits: 0 }
				)} PLN
			</p>
		</div>
	{:else}
		<div class="no-rebalancing inline-flex items-center gap-2">
			<CheckCircle2 size={16} /> Portfel jest zgodny z docelową alokacją (różnice mniejsze niż 1%)
		</div>
	{/if}

	<h2>Przegląd finansowy</h2>

	<div class="metrics-grid">
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
		<h2>Podsumowanie PPK</h2>
		{#each data.ppkStats as ppkStat}
			<h3 class="ppk-owner-title">{ppkStat.owner}</h3>
			<div class="metrics-grid">
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
		<h2>Podsumowanie Akcji</h2>
		<div class="metrics-grid">
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
		<h2>Podsumowanie Obligacji</h2>
		<div class="metrics-grid">
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

	<h2>Struktura portfela inwestycyjnego</h2>

	<div class="charts-grid">
		<div class="chart-container">
			<div bind:this={allocationChart} class="chart"></div>
		</div>

		<div class="chart-container">
			<div bind:this={wrapperChart} class="chart"></div>
		</div>
	</div>

	<h2>Wzrost inwestycji w czasie</h2>

	<div class="chart-container-wide">
		<div bind:this={investmentTrendChart} class="chart-wide"></div>
	</div>

	<h2>Wzrost według typu konta</h2>

	<div class="wrapper-charts-grid">
		<div class="chart-container">
			<div bind:this={ikeChart} class="chart"></div>
		</div>
		<div class="chart-container">
			<div bind:this={ikzeChart} class="chart"></div>
		</div>
		<div class="chart-container">
			<div bind:this={ppkChart} class="chart"></div>
		</div>
	</div>

	<h2>Wzrost według typu inwestycji</h2>

	<div class="wrapper-charts-grid">
		<div class="chart-container">
			<div bind:this={stockChart} class="chart"></div>
		</div>
		<div class="chart-container">
			<div bind:this={bondChart} class="chart"></div>
		</div>
	</div>

	<h2>Roczny ROI według klasy aktywów</h2>

	<div class="chart-container-wide">
		<div bind:this={yearlyRoiChart} class="chart-wide"></div>
	</div>
</div>

<style>
	.container {
		padding: var(--size-5);
		max-width: 1400px;
		margin: 0 auto;
	}

	h1 {
		margin-bottom: var(--size-6);
		color: var(--color-text);
	}

	h2 {
		margin-top: var(--size-8);
		margin-bottom: var(--size-4);
		color: var(--color-text);
		font-size: var(--font-size-4);
	}

	.metrics-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: var(--size-4);
	}

	.charts-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: var(--size-6);
		margin-bottom: var(--size-8);
	}

	.wrapper-charts-grid {
		display: grid;
		grid-template-columns: 1fr;
		gap: var(--size-4);
		margin-bottom: var(--size-8);
	}

	.chart-container {
		background: var(--surface-2);
		border-radius: var(--radius-2);
		padding: var(--size-4);
		border: 1px solid var(--surface-3);
	}

	.chart {
		width: 100%;
		height: 400px;
	}

	.chart-container-wide {
		background: var(--surface-2);
		border-radius: var(--radius-2);
		padding: var(--size-4);
		border: 1px solid var(--surface-3);
		margin-bottom: var(--size-8);
	}

	.chart-wide {
		width: 100%;
		height: 500px;
	}

	.rebalancing-container {
		background: var(--surface-2);
		border-radius: var(--radius-2);
		padding: var(--size-5);
		border: 1px solid var(--surface-3);
	}

	.rebalancing-intro {
		margin-bottom: var(--size-4);
		color: var(--text-2);
	}

	.rebalancing-list {
		display: flex;
		flex-direction: column;
		gap: var(--size-3);
		margin-bottom: var(--size-4);
	}

	.rebalancing-item {
		display: flex;
		align-items: center;
		gap: var(--size-4);
		padding: var(--size-3);
		border-radius: var(--radius-2);
		border: 1px solid;
	}

	.rebalancing-item.buy {
		background: rgba(163, 190, 140, 0.1);
		border-color: var(--green-6);
	}

	.action-label {
		font-weight: var(--font-weight-7);
		min-width: 120px;
	}

	.rebalancing-item.buy .action-label {
		color: var(--green-6);
	}

	.category-name {
		flex: 1;
		color: var(--color-text);
		text-transform: capitalize;
	}

	.amount {
		font-weight: var(--font-weight-7);
		color: var(--color-text);
		font-size: var(--font-size-3);
	}

	.rebalancing-note {
		color: var(--text-2);
		font-style: italic;
		margin-top: var(--size-4);
	}

	.no-rebalancing {
		background: rgba(163, 190, 140, 0.15);
		border: 1px solid var(--green-6);
		border-radius: var(--radius-2);
		padding: var(--size-4);
		color: var(--green-6);
		text-align: center;
		font-weight: var(--font-weight-6);
	}

	.ppk-owner-title {
		font-size: var(--font-size-3);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		margin-top: var(--size-4);
		margin-bottom: var(--size-3);
	}

	@media (max-width: 1024px) {
		.metrics-grid {
			grid-template-columns: repeat(2, 1fr);
		}

		.charts-grid {
			grid-template-columns: 1fr;
		}
	}

	@media (max-width: 640px) {
		.container {
			padding: var(--size-3);
		}

		.metrics-grid {
			grid-template-columns: 1fr;
		}

		.chart {
			height: 280px;
		}

		.chart-wide {
			height: 320px;
		}

		.chart-container,
		.chart-container-wide,
		.rebalancing-container {
			padding: var(--size-3);
		}

		h1 {
			font-size: var(--font-size-5);
		}

		h2 {
			font-size: var(--font-size-3);
			margin-top: var(--size-6);
		}

		.rebalancing-item {
			flex-direction: column;
			align-items: flex-start;
			gap: var(--size-2);
		}

		.action-label {
			min-width: auto;
		}
	}
</style>
