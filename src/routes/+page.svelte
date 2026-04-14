<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN, formatPercent, calculateChange } from '$lib/utils/format';
	import { isMobile, isTablet } from '$lib/utils/viewport';
	import { Wallet, TrendingUp, TrendingDown, Settings, CheckCircle2, AlertTriangle, PiggyBank } from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import type { Persona } from '$lib/types/personas';

	export let data;

	let chartContainer: HTMLDivElement;
	let pieChartContainer: HTMLDivElement;
	let lineChart: echarts.ECharts;
	let pieChart: echarts.ECharts;

	$: gridConfig = $isMobile
		? { left: '40px', right: '20px' }
		: $isTablet
			? { left: '60px', right: '30px' }
			: { left: '80px', right: '40px' };

	$: titleFontSize = $isMobile ? 14 : 16;

	$: personas = (data.personas || []) as Persona[];

	let showLimitsModal = false;
	let limitsYear = data.currentYear;
	let limits: Record<string, number> = {};

	function openLimitsModal() {
		limits = {};
		for (const persona of personas) {
			for (const wrapper of ['IKE', 'IKZE']) {
				limits[`${wrapper}_${persona.name}`] = 0;
			}
		}
		if (data.retirementStats && data.retirementStats.length > 0) {
			data.retirementStats.forEach((stat: any) => {
				const key = `${stat.account_wrapper}_${stat.owner}`;
				if (key in limits) {
					limits[key] = stat.limit_amount || 0;
				}
			});
		}
		showLimitsModal = true;
	}

	async function saveLimits() {
		const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
		try {
			const requests = Object.entries(limits).map(([key, amount]) => {
				const sep = key.indexOf('_');
				const wrapper = key.slice(0, sep);
				const owner = key.slice(sep + 1);
				return fetch(
					`${apiUrl}/api/retirement/limits/${limitsYear}/${wrapper}/${encodeURIComponent(owner)}`,
					{
						method: 'PUT',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({
							year: limitsYear,
							account_wrapper: wrapper,
							owner: owner,
							limit_amount: amount,
							notes: ''
						})
					}
				);
			});

			const responses = await Promise.all(requests);

			const failedResponses = responses.filter((r) => !r.ok);
			if (failedResponses.length > 0) {
				throw new Error(`${failedResponses.length} request(s) failed`);
			}

			showLimitsModal = false;
			await invalidateAll();
		} catch (err) {
			console.error('Failed to save limits:', err);
			alert('Nie udało się zapisać limitów. Spróbuj ponownie później.');
		}
	}

	onMount(() => {
		lineChart = echarts.init(chartContainer);

		const lineOption: EChartsOption = {
			title: {
				text: 'Wartość Netto w Czasie',
				left: 'center',
				textStyle: { fontSize: titleFontSize }
			},
			tooltip: {
				trigger: 'axis',
				formatter: (params: any) => {
					const date = new Date(params[0].value[0]).toLocaleDateString('pl-PL');
					const value = formatPLN(params[0].value[1]);
					return `${date}<br/>Wartość: ${value}`;
				}
			},
			xAxis: { type: 'time' },
			yAxis: {
				type: 'value',
				axisLabel: { formatter: (value: number) => formatPLN(value) }
			},
			series: [
				{
					data: data.net_worth_history.map((h: any) => [h.date, h.value]),
					type: 'line',
					smooth: true,
					areaStyle: {
						color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
							{ offset: 0, color: 'rgba(225, 29, 72, 0.5)' },
							{ offset: 1, color: 'rgba(225, 29, 72, 0.1)' }
						])
					},
					lineStyle: { color: '#e11d48', width: 2 }
				}
			],
			grid: gridConfig
		};

		lineChart.setOption(lineOption);

		pieChart = echarts.init(pieChartContainer);

		const pieOption: EChartsOption = {
			title: {
				text: 'Alokacja Aktywów',
				left: 'center',
				textStyle: { fontSize: titleFontSize }
			},
			tooltip: { trigger: 'item', formatter: '{b}: {c} PLN ({d}%)' },
			color: [
				'#e11d48',
				'#f43f5e',
				'#fb7185',
				'#fda4af',
				'#881337',
				'#9f1239',
				'#be123c',
				'#be185d'
			],
			series: [
				{
					type: 'pie',
					radius: ['40%', '70%'],
					data: data.allocation.map((a: any) => ({
						name: `${a.category}${a.owner ? ` (${a.owner})` : ''}`,
						value: a.value
					})),
					emphasis: {
						itemStyle: {
							shadowBlur: 10,
							shadowOffsetX: 0,
							shadowColor: 'rgba(0, 0, 0, 0.5)'
						}
					}
				}
			]
		};

		pieChart.setOption(pieOption);

		const handleResize = () => {
			lineChart.resize();
			pieChart.resize();
		};

		window.addEventListener('resize', handleResize);

		return () => {
			window.removeEventListener('resize', handleResize);
		};
	});

	const change = calculateChange(
		data.current_net_worth,
		data.current_net_worth - data.change_vs_last_month
	);

	$: if (lineChart && gridConfig) {
		lineChart.setOption({
			grid: gridConfig,
			title: { textStyle: { fontSize: titleFontSize } }
		});
	}

	$: if (pieChart && titleFontSize) {
		pieChart.setOption({
			title: { textStyle: { fontSize: titleFontSize } }
		});
	}
</script>

<svelte:head>
	<title>Dashboard | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-8">
	<div class="space-y-1">
		<h1 class="h2">Dashboard</h1>
		<p class="text-surface-700-300 text-sm">Twoja sytuacja finansowa w jednym miejscu</p>
	</div>

	<div class="grid gap-4 sm:gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
		<div class="card preset-filled-surface-100-900 p-4 space-y-2">
			<header>
				<h3 class="h4 flex items-center gap-2"><Wallet size={18} /> Wartość Netto</h3>
			</header>
			<div class="text-3xl font-bold">{formatPLN(data.current_net_worth)}</div>
			<p
				class="text-sm flex items-center gap-1 {data.change_vs_last_month >= 0
					? 'text-success-600-400'
					: 'text-error-600-400'}"
			>
				{#if data.change_vs_last_month >= 0}
					<TrendingUp size={14} />
				{:else}
					<TrendingDown size={14} />
				{/if}
				{formatPLN(Math.abs(data.change_vs_last_month))}
				({formatPercent(Math.abs(change.percent))})
				<span class="text-surface-700-300">vs poprzedni miesiąc</span>
			</p>
		</div>

		<div class="card preset-filled-surface-100-900 p-4 space-y-2">
			<header>
				<h3 class="h4 flex items-center gap-2"><TrendingUp size={18} /> Aktywa</h3>
			</header>
			<div class="text-3xl font-bold text-success-600-400">{formatPLN(data.total_assets)}</div>
			<p class="text-sm text-surface-700-300">Suma wszystkich aktywów</p>
		</div>

		<div class="card preset-filled-surface-100-900 p-4 space-y-2">
			<header>
				<h3 class="h4 flex items-center gap-2"><TrendingDown size={18} /> Zobowiązania</h3>
			</header>
			<div class="text-3xl font-bold text-error-600-400">{formatPLN(data.total_liabilities)}</div>
			<p class="text-sm text-surface-700-300">Suma wszystkich zobowiązań</p>
		</div>
	</div>

	{#if data.retirementStats && data.retirementStats.length > 0}
		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header class="flex items-center justify-between gap-2">
				<h3 class="h3 flex items-center gap-2"><PiggyBank size={20} /> Limity Emerytalne {data.currentYear}</h3>
				<button
					type="button"
					class="btn-icon btn-icon-sm"
					aria-label="Konfiguruj limity"
					on:click={openLimitsModal}
				>
					<Settings size={18} />
				</button>
			</header>

			<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
				{#each data.retirementStats as stat}
					<div class="card preset-tonal-surface p-4 space-y-2">
						<div class="flex items-start justify-between gap-2">
							<h4 class="font-bold">{stat.account_wrapper} ({stat.owner})</h4>
							<span class="text-sm font-semibold whitespace-nowrap">
								{formatPLN(stat.total_contributed)} / {formatPLN(stat.limit_amount)}
							</span>
						</div>

						<div class="h-2 rounded-full bg-surface-200-800 overflow-hidden">
							<div
								class="h-full transition-all {stat.percentage_used >= 100
									? 'bg-success-500'
									: stat.percentage_used >= 50
										? 'bg-warning-500'
										: 'bg-error-500'}"
								style="width: {Math.min(stat.percentage_used, 100)}%"
							></div>
						</div>

						<div class="flex items-center justify-between text-sm">
							<span class="text-surface-700-300">Pozostało: {formatPLN(stat.remaining)}</span>
							<span
								class="font-semibold {stat.percentage_used >= 100
									? 'text-success-600-400'
									: stat.percentage_used >= 50
										? 'text-warning-600-400'
										: 'text-error-600-400'}"
							>
								{stat.percentage_used}%
							</span>
						</div>

						{#if stat.percentage_used >= 100}
							<div class="card preset-filled-success-500 p-2 text-xs text-center flex items-center justify-center gap-1">
								<CheckCircle2 size={14} /> Limit osiągnięty
							</div>
						{:else if stat.percentage_used >= 50}
							<div class="card preset-filled-warning-500 p-2 text-xs text-center flex items-center justify-center gap-1">
								<AlertTriangle size={14} /> Zbliżasz się do limitu
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
		<div class="card preset-filled-surface-100-900 p-4">
			<div bind:this={chartContainer} class="w-full h-[400px]"></div>
		</div>

		<div class="card preset-filled-surface-100-900 p-4">
			<div bind:this={pieChartContainer} class="w-full h-[400px]"></div>
		</div>
	</div>
</div>

<Modal
	open={showLimitsModal}
	title="Konfiguracja Limitów Emerytalnych"
	onCancel={() => (showLimitsModal = false)}
	onConfirm={saveLimits}
	confirmText="Zapisz"
>
	<form on:submit|preventDefault={saveLimits} class="space-y-4">
		<label class="label">
			<span class="font-semibold text-sm">Rok</span>
			<select class="select" bind:value={limitsYear}>
				{#each [data.currentYear, data.currentYear + 1, data.currentYear + 2] as year}
					<option value={year}>{year}</option>
				{/each}
			</select>
		</label>

		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each ['IKE', 'IKZE'] as wrapper}
				{#each personas as persona}
					<label class="label">
						<span class="font-semibold text-sm">{wrapper} {persona.name} (PLN)</span>
						<input
							type="number"
							class="input"
							bind:value={limits[`${wrapper}_${persona.name}`]}
							step="0.01"
							required
						/>
					</label>
				{/each}
			{/each}
		</div>
	</form>
</Modal>
