<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import { MetricCard } from '@mskalski/home-ui';

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
		// Allocation by category chart (current vs target)
		const allocationChartInstance = echarts.init(allocationChart);
		const categories = allocationAnalysis.by_category.map((item: any) => item.category);
		const currentValues = allocationAnalysis.by_category.map((item: any) =>
			parseFloat(item.current_percentage.toFixed(1))
		);
		const targetValues = allocationAnalysis.by_category.map((item: any) =>
			parseFloat(item.target_percentage.toFixed(1))
		);

		allocationChartInstance.setOption({
			backgroundColor: 'transparent',
			title: {
				text: 'Alokacja inwestycyjna: Obecna vs Docelowa',
				left: 'center',
				top: 10,
				textStyle: { color: '#2e3440', fontSize: 16, fontWeight: 'bold' }
			},
			tooltip: {
				trigger: 'axis',
				axisPointer: { type: 'shadow' },
				backgroundColor: 'rgba(255, 255, 255, 0.95)',
				borderColor: '#d8dee9',
				textStyle: { color: '#2e3440' },
				formatter: '{b}<br/>{a0}: {c0}%<br/>{a1}: {c1}%'
			},
			legend: {
				data: ['Obecna', 'Docelowa'],
				bottom: 10,
				textStyle: { color: '#2e3440', fontSize: 14 }
			},
			grid: {
				left: 60,
				right: 40,
				bottom: 60,
				top: 80,
				containLabel: false
			},
			xAxis: {
				type: 'category',
				data: categories,
				axisLabel: {
					color: '#2e3440',
					fontSize: 14,
					fontWeight: 'bold',
					formatter: function (value: string) {
						return value.charAt(0).toUpperCase() + value.slice(1);
					}
				},
				axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
				axisTick: { show: false }
			},
			yAxis: {
				type: 'value',
				name: 'Procent (%)',
				nameLocation: 'middle',
				nameGap: 50,
				nameTextStyle: { color: '#2e3440', fontSize: 14, fontWeight: 'bold' },
				max: 100,
				axisLabel: {
					color: '#2e3440',
					fontSize: 13,
					fontWeight: 'bold',
					formatter: '{value}%'
				},
				axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
				splitLine: { lineStyle: { color: '#e5e9f0', type: 'dashed' } }
			},
			series: [
				{
					name: 'Obecna',
					type: 'bar',
					data: currentValues,
					barWidth: '35%',
					itemStyle: { color: '#88c0d0', borderRadius: [4, 4, 0, 0] },
					label: {
						show: true,
						position: 'top',
						distance: 5,
						formatter: '{c}%',
						color: '#2e3440',
						fontSize: 14,
						fontWeight: 'bold'
					}
				},
				{
					name: 'Docelowa',
					type: 'bar',
					data: targetValues,
					barWidth: '35%',
					itemStyle: { color: '#81a1c1', borderRadius: [4, 4, 0, 0] },
					label: {
						show: true,
						position: 'top',
						distance: 5,
						formatter: '{c}%',
						color: '#2e3440',
						fontSize: 14,
						fontWeight: 'bold'
					}
				}
			]
		});

		// Wrapper breakdown chart
		const wrapperChartInstance = echarts.init(wrapperChart);
		const wrapperData = allocationAnalysis.by_wrapper.map((item: any) => ({
			name: item.wrapper,
			value: item.value
		}));

		wrapperChartInstance.setOption({
			backgroundColor: 'transparent',
			title: {
				text: 'Podział według kont (IKE/IKZE/PPK)',
				left: 'center',
				top: 10,
				textStyle: { color: '#2e3440', fontSize: 16, fontWeight: 'bold' }
			},
			tooltip: {
				trigger: 'item',
				backgroundColor: 'rgba(255, 255, 255, 0.95)',
				borderColor: '#d8dee9',
				textStyle: { color: '#2e3440' },
				formatter: function (params: any) {
					const value = params.value.toLocaleString('pl-PL', {
						minimumFractionDigits: 0,
						maximumFractionDigits: 0
					});
					return `${params.name}: ${value} PLN (${params.percent.toFixed(1)}%)`;
				}
			},
			legend: {
				orient: 'vertical',
				left: 20,
				top: 'middle',
				textStyle: { color: '#2e3440', fontSize: 14, fontWeight: 'bold' },
				itemGap: 15
			},
			series: [
				{
					type: 'pie',
					radius: ['40%', '65%'],
					center: ['50%', '50%'],
					avoidLabelOverlap: true,
					minShowLabelAngle: 1,
					data: wrapperData,
					color: ['#88c0d0', '#81a1c1', '#5e81ac', '#b48ead'],
					emphasis: {
						itemStyle: {
							shadowBlur: 10,
							shadowOffsetX: 0,
							shadowColor: 'rgba(0, 0, 0, 0.5)'
						},
						label: {
							show: true,
							fontSize: 18,
							fontWeight: 'bold'
						}
					},
					label: {
						show: true,
						position: 'outside',
						alignTo: 'edge',
						margin: 20,
						edgeDistance: '15%',
						color: '#000000',
						fontSize: 15,
						fontWeight: 'bold',
						formatter: function (params: any) {
							return `${params.name}\n${params.percent.toFixed(1)}%`;
						},
						overflow: 'none'
					},
					labelLine: {
						show: true,
						length: 25,
						length2: 20,
						smooth: 0.2,
						lineStyle: {
							color: '#2e3440',
							width: 2
						}
					},
					labelLayout: {
						hideOverlap: false,
						moveOverlap: 'shiftY'
					}
				}
			]
		});

		// Investment trend chart (value, contributions, returns over time)
		const investmentTrendChartInstance = echarts.init(investmentTrendChart);
		const dates = investmentTimeSeries.map((item: any) => item.date);
		const values = investmentTimeSeries.map((item: any) => item.value);
		const contributions = investmentTimeSeries.map((item: any) => item.contributions);

		investmentTrendChartInstance.setOption({
			backgroundColor: 'transparent',
			title: {
				text: 'Inwestycje w czasie',
				left: 'center',
				top: 10,
				textStyle: { color: '#2e3440', fontSize: 16, fontWeight: 'bold' }
			},
			tooltip: {
				trigger: 'axis',
				backgroundColor: 'rgba(255, 255, 255, 0.95)',
				borderColor: '#d8dee9',
				textStyle: { color: '#2e3440' },
				formatter: function (params: any) {
					const date = params[0].axisValue;
					const contributions = params[0].value;
					const value = params[1].value;
					const returns = value - contributions;

					const formatPLN = (val: number) =>
						val.toLocaleString('pl-PL', { minimumFractionDigits: 0, maximumFractionDigits: 0 });

					return `${date}<br/>
						<span style="color:#5e81ac">●</span> Wartość portfela: <b>${formatPLN(value)} PLN</b><br/>
						<span style="color:#88c0d0">■</span> Wpłaty: ${formatPLN(contributions)} PLN<br/>
						<span style="color:#a3be8c">■</span> Zyski: ${formatPLN(returns)} PLN`;
				}
			},
			legend: {
				data: ['Wpłaty', 'Wartość portfela'],
				bottom: 10,
				textStyle: { color: '#2e3440', fontSize: 14 }
			},
			grid: {
				left: 80,
				right: 40,
				bottom: 80,
				top: 80,
				containLabel: false
			},
			xAxis: {
				type: 'category',
				data: dates,
				axisLabel: {
					color: '#2e3440',
					fontSize: 12,
					rotate: 45
				},
				axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
				boundaryGap: false
			},
			yAxis: {
				type: 'value',
				name: 'Wartość (PLN)',
				nameLocation: 'middle',
				nameGap: 60,
				nameTextStyle: { color: '#2e3440', fontSize: 14, fontWeight: 'bold' },
				axisLabel: {
					color: '#2e3440',
					fontSize: 12,
					formatter: function (value: number) {
						return (value / 1000).toFixed(0) + 'k';
					}
				},
				axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
				splitLine: { lineStyle: { color: '#e5e9f0', type: 'dashed' } }
			},
			series: [
				{
					name: 'Wpłaty',
					type: 'line',
					data: contributions,
					smooth: true,
					lineStyle: { width: 0 },
					showSymbol: false,
					areaStyle: {
						color: '#88c0d0',
						opacity: 0.8
					},
					emphasis: {
						focus: 'series'
					}
				},
				{
					name: 'Wartość portfela',
					type: 'line',
					data: values,
					smooth: true,
					lineStyle: { width: 3, color: '#5e81ac' },
					showSymbol: false,
					emphasis: {
						focus: 'series'
					}
				}
			]
		});

		// Helper function to create wrapper chart config
		const createWrapperChart = (chartElement: HTMLDivElement, title: string, data: any[]) => {
			const chartInstance = echarts.init(chartElement);
			const dates = data.map((item: any) => item.date);
			const contributions = data.map((item: any) => item.contributions);
			const values = data.map((item: any) => item.value);

			chartInstance.setOption({
				backgroundColor: 'transparent',
				title: {
					text: title,
					left: 'center',
					top: 10,
					textStyle: { color: '#2e3440', fontSize: 16, fontWeight: 'bold' }
				},
				tooltip: {
					trigger: 'axis',
					backgroundColor: 'rgba(255, 255, 255, 0.95)',
					borderColor: '#d8dee9',
					textStyle: { color: '#2e3440' },
					formatter: function (params: any) {
						const date = params[0].axisValue;
						const contributions = params[0].value;
						const value = params[1].value;
						const returns = value - contributions;

						const formatPLN = (val: number) =>
							val.toLocaleString('pl-PL', { minimumFractionDigits: 0, maximumFractionDigits: 0 });

						return `${date}<br/>
							<span style="color:#5e81ac">●</span> Wartość: <b>${formatPLN(value)} PLN</b><br/>
							<span style="color:#88c0d0">■</span> Wpłaty: ${formatPLN(contributions)} PLN<br/>
							<span style="color:#a3be8c">■</span> Zyski: ${formatPLN(returns)} PLN`;
					}
				},
				legend: {
					data: ['Wpłaty', 'Wartość portfela'],
					bottom: 10,
					textStyle: { color: '#2e3440', fontSize: 14 }
				},
				grid: {
					left: 80,
					right: 40,
					bottom: 80,
					top: 80,
					containLabel: false
				},
				xAxis: {
					type: 'category',
					data: dates,
					axisLabel: {
						color: '#2e3440',
						fontSize: 11,
						rotate: 45
					},
					axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
					boundaryGap: false
				},
				yAxis: {
					type: 'value',
					name: 'Wartość (PLN)',
					nameLocation: 'middle',
					nameGap: 60,
					nameTextStyle: { color: '#2e3440', fontSize: 14, fontWeight: 'bold' },
					axisLabel: {
						color: '#2e3440',
						fontSize: 11,
						formatter: function (value: number) {
							return (value / 1000).toFixed(0) + 'k';
						}
					},
					axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
					splitLine: { lineStyle: { color: '#e5e9f0', type: 'dashed' } }
				},
				series: [
					{
						name: 'Wpłaty',
						type: 'line',
						data: contributions,
						smooth: true,
						lineStyle: { width: 0 },
						showSymbol: false,
						areaStyle: {
							color: '#88c0d0',
							opacity: 0.8
						},
						emphasis: {
							focus: 'series'
						}
					},
					{
						name: 'Wartość portfela',
						type: 'line',
						data: values,
						smooth: true,
						lineStyle: { width: 3, color: '#5e81ac' },
						showSymbol: false,
						emphasis: {
							focus: 'series'
						}
					}
				]
			});

			return chartInstance;
		};

		// IKE chart
		const ikeChartInstance = createWrapperChart(ikeChart, 'IKE w czasie', wrapperTimeSeries.ike);

		// IKZE chart
		const ikzeChartInstance = createWrapperChart(
			ikzeChart,
			'IKZE w czasie',
			wrapperTimeSeries.ikze
		);

		// PPK chart
		const ppkChartInstance = createWrapperChart(ppkChart, 'PPK w czasie', wrapperTimeSeries.ppk);

		// Stock chart
		const stockChartInstance = createWrapperChart(
			stockChart,
			'Akcje w czasie',
			categoryTimeSeries.stock
		);

		// Bond chart
		const bondChartInstance = createWrapperChart(
			bondChart,
			'Obligacje w czasie',
			categoryTimeSeries.bond
		);

		// Yearly ROI chart
		function computeYearlyROI(
			series: Array<{
				date: string;
				value?: number;
				total_value?: number;
				contributions?: number;
				cumulative_contributions?: number;
			}>
		): Map<number, number> {
			type Pt = { date: string; value: number; contribs: number };
			const byYear = new Map<number, { first: Pt; last: Pt }>();
			for (const point of series) {
				const year = new Date(point.date).getFullYear();
				const val = point.value ?? point.total_value ?? 0;
				const contrib = point.contributions ?? point.cumulative_contributions ?? 0;
				const pt: Pt = { date: point.date, value: val, contribs: contrib };
				const existing = byYear.get(year);
				if (!existing) {
					byYear.set(year, { first: pt, last: pt });
				} else {
					if (point.date < existing.first.date) existing.first = pt;
					if (point.date > existing.last.date) existing.last = pt;
				}
			}
			const years = [...byYear.keys()].sort((a, b) => a - b);
			const result = new Map<number, number>();
			for (let i = 0; i < years.length; i++) {
				const { first, last: end } = byYear.get(years[i])!;
				let start: Pt;
				if (i === 0) {
					// Skip first year if it starts with value=0 (placeholder entry)
					if (first.value === 0) continue;
					start = first;
				} else {
					start = byYear.get(years[i - 1])!.last;
				}
				const yearContribs = end.contribs - start.contribs;
				const denom = start.value + yearContribs / 2;
				const roi = denom > 0 ? ((end.value - start.value - yearContribs) / denom) * 100 : 0;
				result.set(years[i], parseFloat(roi.toFixed(2)));
			}
			return result;
		}

		const stockRoi = computeYearlyROI(categoryTimeSeries.stock);
		const bondRoi = computeYearlyROI(categoryTimeSeries.bond);
		const ppkRoi = computeYearlyROI(wrapperTimeSeries.ppk);
		const allYears = [...new Set([...stockRoi.keys(), ...bondRoi.keys(), ...ppkRoi.keys()])].sort(
			(a, b) => a - b
		);

		const yearlyRoiChartInstance = echarts.init(yearlyRoiChart);
		yearlyRoiChartInstance.setOption({
			backgroundColor: 'transparent',
			title: {
				text: 'Roczny ROI: Akcje, Obligacje, PPK',
				left: 'center',
				top: 10,
				textStyle: { color: '#2e3440', fontSize: 16, fontWeight: 'bold' }
			},
			tooltip: {
				trigger: 'axis',
				axisPointer: { type: 'shadow' },
				backgroundColor: 'rgba(255, 255, 255, 0.95)',
				borderColor: '#d8dee9',
				textStyle: { color: '#2e3440' },
				formatter: (params: Array<{ seriesName: string; value: number }>) =>
					params
						.map((p) => `${p.seriesName}: ${p.value != null ? p.value + '%' : 'brak danych'}`)
						.join('<br/>')
			},
			legend: {
				top: 40,
				textStyle: { color: '#2e3440' }
			},
			grid: { top: 80, left: 60, right: 30, bottom: 60 },
			xAxis: {
				type: 'category',
				data: allYears.map(String),
				axisLabel: { color: '#4c566a' }
			},
			yAxis: {
				type: 'value',
				axisLabel: { formatter: '{value}%', color: '#4c566a' },
				splitLine: { lineStyle: { color: '#e5e9f0' } }
			},
			series: [
				{
					name: 'Akcje',
					type: 'bar',
					barWidth: '25%',
					data: allYears.map((y) => stockRoi.get(y) ?? null),
					itemStyle: { color: '#a3be8c' },
					label: {
						show: true,
						position: (params: { value: number }) => (params.value >= 0 ? 'top' : 'bottom'),
						formatter: '{c}%',
						color: '#2e3440',
						fontSize: 11
					},
					markLine: {
						silent: true,
						symbol: 'none',
						data: [{ yAxis: 0 }],
						lineStyle: { color: '#4c566a', type: 'solid', width: 1 }
					}
				},
				{
					name: 'Obligacje',
					type: 'bar',
					barWidth: '25%',
					data: allYears.map((y) => bondRoi.get(y) ?? null),
					itemStyle: { color: '#d08770' },
					label: {
						show: true,
						position: (params: { value: number }) => (params.value >= 0 ? 'top' : 'bottom'),
						formatter: '{c}%',
						color: '#2e3440',
						fontSize: 11
					}
				},
				{
					name: 'PPK',
					type: 'bar',
					barWidth: '25%',
					data: allYears.map((y) => ppkRoi.get(y) ?? null),
					itemStyle: { color: '#88c0d0' },
					label: {
						show: true,
						position: (params: { value: number }) => (params.value >= 0 ? 'top' : 'bottom'),
						formatter: '{c}%',
						color: '#2e3440',
						fontSize: 11
					}
				}
			]
		});

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
						<span class="action-label"> 📈 KUP </span>
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

			<p class="rebalancing-note">
				💡 Całkowita wartość portfela inwestycyjnego: {allocationAnalysis.total_investment_value.toLocaleString(
					'pl-PL',
					{ minimumFractionDigits: 0, maximumFractionDigits: 0 }
				)} PLN
			</p>
		</div>
	{:else}
		<div class="no-rebalancing">
			✅ Portfel jest zgodny z docelową alokacją (różnice mniejsze niż 1%)
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

	.rebalancing-item.sell {
		background: rgba(191, 97, 106, 0.1);
		border-color: var(--red-6);
	}

	.action-label {
		font-weight: var(--font-weight-7);
		min-width: 120px;
	}

	.rebalancing-item.buy .action-label {
		color: var(--green-6);
	}

	.rebalancing-item.sell .action-label {
		color: var(--red-6);
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
		.metrics-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
