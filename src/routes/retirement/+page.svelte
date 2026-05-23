<script lang="ts">
	import { resolveApiUrl } from '$lib/api';
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import { buildMonteCarloFanOption, type MonteCarloResult } from '$lib/utils/charts/montecarlo';
	import { Info, Sparkles } from 'lucide-svelte';
	import * as echarts from 'echarts';

	let currentPortfolio = $state(100000);
	let annualContribution = $state(20000);
	let expectedReturn = $state(6);
	let volatility = $state(15);
	let currentAge = $state(35);
	let retirementAge = $state(65);
	let lifeExpectancy = $state(90);
	let annualWithdrawal = $state(40000);

	let loading = $state(false);
	let error = $state('');
	let result: MonteCarloResult | null = $state(null);

	let chartContainer: HTMLDivElement | undefined = $state();
	let chartHandle: ChartHandle | null = null;
	let chart: echarts.ECharts | null = null;

	let showHelp = $state(false);

	async function runMonteCarlo() {
		loading = true;
		error = '';
		result = null;

		try {
			if (lifeExpectancy <= currentAge) {
				error = 'Oczekiwana długość życia musi być większa niż obecny wiek';
				loading = false;
				return;
			}
			if (retirementAge < currentAge || retirementAge > lifeExpectancy) {
				error = 'Wiek emerytalny musi mieścić się między obecnym wiekiem a długością życia';
				loading = false;
				return;
			}
			if (volatility < 0) {
				error = 'Zmienność nie może być ujemna';
				loading = false;
				return;
			}

			const apiUrl = resolveApiUrl();
			const response = await fetch(`${apiUrl}/api/simulations/monte-carlo`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					current_portfolio: currentPortfolio,
					annual_contribution: annualContribution,
					expected_return: expectedReturn,
					volatility,
					current_age: currentAge,
					retirement_age: retirementAge,
					life_expectancy: lifeExpectancy,
					annual_withdrawal: annualWithdrawal
				})
			});
			if (!response.ok) throw new Error(`Symulacja nieudana: ${response.statusText}`);
			result = await response.json();
		} catch (err) {
			if (err instanceof Error) error = err.message;
		} finally {
			loading = false;
		}
	}

	// Render the fan chart whenever results land and the container is bound.
	$effect(() => {
		if (!chartContainer) {
			chartHandle?.dispose();
			chartHandle = null;
			chart = null;
			return;
		}
		if (!result) return;

		if (!chartHandle) {
			chartHandle = createChart(chartContainer);
			chart = chartHandle.chart;
		}
		chart?.setOption(buildMonteCarloFanOption(result));
	});

	const successPercent = $derived.by(() => {
		const r = result;
		return r ? Math.round(r.success_rate * 100) : null;
	});
	const successClass = $derived.by(() => {
		if (successPercent === null) return '';
		if (successPercent >= 90) return 'text-success-600-400';
		if (successPercent >= 70) return 'text-warning-600-400';
		return 'text-error-600-400';
	});

	function formatPLN(value: number): string {
		return value.toLocaleString('pl-PL', { maximumFractionDigits: 0 });
	}
</script>

<svelte:head>
	<title>Monte Carlo emerytura | Finansowa Forteca</title>
</svelte:head>

<div class="flex flex-col gap-6">
	<header class="space-y-1">
		<h1 class="h2 flex items-center gap-2">
			<Sparkles size={24} class="text-primary-500" />
			Symulacja Monte Carlo
		</h1>
		<p class="text-surface-700-300 text-sm">
			Tysiąc losowych ścieżek emerytalnych zamiast jednego deterministycznego planu.
		</p>
	</header>

	<section class="card preset-filled-surface-100-900 p-5 space-y-4">
		<h2 class="h4">Parametry</h2>
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
			<label class="space-y-1">
				<span class="text-xs font-semibold">Obecny portfel (PLN)</span>
				<input type="number" min="0" class="input w-full" bind:value={currentPortfolio} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Roczna wpłata (PLN)</span>
				<input type="number" min="0" class="input w-full" bind:value={annualContribution} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Oczekiwana stopa zwrotu (%)</span>
				<input type="number" step="0.1" class="input w-full" bind:value={expectedReturn} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Zmienność / odchylenie std. (%)</span>
				<input type="number" min="0" step="0.1" class="input w-full" bind:value={volatility} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Obecny wiek</span>
				<input type="number" min="0" max="120" class="input w-full" bind:value={currentAge} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Wiek emerytalny</span>
				<input type="number" min="0" max="120" class="input w-full" bind:value={retirementAge} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Oczekiwana długość życia</span>
				<input type="number" min="0" max="120" class="input w-full" bind:value={lifeExpectancy} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Roczna wypłata na emeryturze (PLN)</span>
				<input type="number" min="0" class="input w-full" bind:value={annualWithdrawal} />
			</label>
		</div>

		<button
			type="button"
			class="btn preset-filled-primary-500 gap-2 w-full sm:w-auto"
			disabled={loading}
			onclick={runMonteCarlo}
		>
			{loading ? 'Liczę 1000 ścieżek…' : 'Uruchom symulację'}
		</button>

		{#if error}
			<div class="card preset-tonal-error p-3 text-sm">{error}</div>
		{/if}
	</section>

	{#if result}
		<section class="card preset-filled-surface-100-900 p-5 space-y-4">
			<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
				<div>
					<div class="text-xs text-surface-600-400">Wskaźnik powodzenia</div>
					<div class="text-4xl font-bold {successClass}">{successPercent}%</div>
					<div class="text-sm text-surface-700-300">
						{result.paths} ścieżek; oszczędności starczają na całe życie w
						{successPercent}% scenariuszy.
					</div>
				</div>
				<button
					type="button"
					class="btn-icon btn-icon-sm"
					aria-label="Co to znaczy?"
					onclick={() => (showHelp = !showHelp)}
				>
					<Info size={20} />
				</button>
			</div>
			{#if showHelp}
				<div class="card preset-tonal-surface p-3 text-sm space-y-2">
					<p>
						<strong>Wskaźnik powodzenia</strong> to odsetek symulowanych ścieżek, w których oszczędności
						są dodatnie po osiągnięciu oczekiwanego wieku zgonu. Każda ścieżka losuje roczną stopę zwrotu
						z rozkładu normalnego N(stopa, zmienność).
					</p>
					<p>
						<strong>Pasmo P5–P95</strong> obejmuje 90% scenariuszy: pesymistyczny dolny brzeg (P5), mediana
						(P50) i optymistyczny górny (P95). Im węższe, tym mniejsze ryzyko sekwencji zwrotów.
					</p>
				</div>
			{/if}

			<div bind:this={chartContainer} class="w-full h-[320px] sm:h-[420px]"></div>

			{#if result.bands.length > 0}
				{@const last = result.bands[result.bands.length - 1]}
				<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400">Pesymistyczne (P5) w wieku {last.age}</div>
						<div class="text-xl font-bold">{formatPLN(last.p5)} PLN</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400">Mediana (P50) w wieku {last.age}</div>
						<div class="text-xl font-bold">{formatPLN(last.p50)} PLN</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400">Optymistyczne (P95) w wieku {last.age}</div>
						<div class="text-xl font-bold">{formatPLN(last.p95)} PLN</div>
					</div>
				</div>
			{/if}
		</section>
	{/if}
</div>
