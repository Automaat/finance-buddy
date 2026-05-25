<script lang="ts">
	import { resolveApiUrl } from '$lib/api';
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import { buildMonteCarloFanOption, type MonteCarloResult } from '$lib/utils/charts/montecarlo';
	import { Info, Sparkles } from 'lucide-svelte';
	import * as echarts from 'echarts';
	import IKZEPITTracker from '$lib/components/IKZEPITTracker.svelte';
	import IKZEOptimizer from '$lib/components/IKZEOptimizer.svelte';
	import FireGapChart from '$lib/components/FireGapChart.svelte';

	let currentPortfolio = $state(100000);
	let annualContribution = $state(20000);
	let expectedReturn = $state(6);
	let volatility = $state(15);
	let currentAge = $state(35);
	let retirementAge = $state(65);
	let lifeExpectancy = $state(90);
	let annualWithdrawal = $state(40000);

	let useAllocation = $state(false);
	let allocStocks = $state(60);
	let allocBonds = $state(30);
	let allocCash = $state(10);
	const allocSum = $derived(allocStocks + allocBonds + allocCash);

	let inflationMean = $state(0);
	let inflationVolatility = $state(0);
	let showReal = $state(true);

	let useAccountMix = $state(false);
	let mixTaxable = $state(50);
	let mixIke = $state(20);
	let mixIkze = $state(20);
	let mixZus = $state(10);
	let mixTaxableGain = $state(60);
	const mixSum = $derived(mixTaxable + mixIke + mixIkze + mixZus);
	let showNet = $state(false);

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
			if (useAllocation && Math.abs(allocSum - 100) > 0.01) {
				error = `Alokacja musi sumować się do 100% (obecnie ${allocSum.toFixed(1)}%)`;
				loading = false;
				return;
			}
			if (inflationVolatility < 0) {
				error = 'Zmienność inflacji nie może być ujemna';
				loading = false;
				return;
			}
			if (useAccountMix && (mixTaxable < 0 || mixIke < 0 || mixIkze < 0 || mixZus < 0)) {
				error = 'Udziały kont muszą być nieujemne';
				loading = false;
				return;
			}
			if (useAccountMix && Math.abs(mixSum - 100) > 0.01) {
				error = `Konta muszą sumować się do 100% (obecnie ${mixSum.toFixed(1)}%)`;
				loading = false;
				return;
			}
			if (useAccountMix && (mixTaxableGain < 0 || mixTaxableGain > 100)) {
				error = 'Udział zysku w koncie maklerskim musi mieścić się w 0–100%';
				loading = false;
				return;
			}

			const apiUrl = resolveApiUrl();
			const body: Record<string, unknown> = {
				current_portfolio: currentPortfolio,
				annual_contribution: annualContribution,
				expected_return: expectedReturn,
				volatility,
				current_age: currentAge,
				retirement_age: retirementAge,
				life_expectancy: lifeExpectancy,
				annual_withdrawal: annualWithdrawal,
				inflation_mean: inflationMean,
				inflation_volatility: inflationVolatility
			};
			if (useAllocation) {
				body.allocation = {
					stocks_pct: allocStocks,
					bonds_pct: allocBonds,
					cash_pct: allocCash
				};
			}
			if (useAccountMix) {
				body.account_mix = {
					taxable_pct: mixTaxable,
					ike_pct: mixIke,
					ikze_pct: mixIkze,
					zus_pct: mixZus,
					taxable_gain_pct: mixTaxableGain
				};
			}
			const response = await fetch(`${apiUrl}/api/simulations/monte-carlo`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});
			if (!response.ok) throw new Error(`Symulacja nieudana: ${response.statusText}`);
			result = await response.json();
		} catch (err) {
			if (err instanceof Error) error = err.message;
		} finally {
			loading = false;
		}
	}

	// Render the fan chart whenever results land and the container is bound,
	// or the real/nominal toggle flips.
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
		const net = showNet && !!result.tax;
		chart?.setOption(buildMonteCarloFanOption(result, { real: showReal, net }), true);
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

	<IKZEPITTracker />

	<IKZEOptimizer />

	<FireGapChart
		bind:currentAge
		bind:retirementAge
		bind:lifeExpectancy
		bind:currentPortfolioPLN={currentPortfolio}
		bind:annualContributionPLN={annualContribution}
		bind:expectedReturnPct={expectedReturn}
	/>

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
				<input
					type="number"
					step="0.1"
					class="input w-full"
					bind:value={expectedReturn}
					disabled={useAllocation}
				/>
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Zmienność / odchylenie std. (%)</span>
				<input
					type="number"
					min="0"
					step="0.1"
					class="input w-full"
					bind:value={volatility}
					disabled={useAllocation}
				/>
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Obecny wiek</span>
				<input type="number" min="18" max="120" class="input w-full" bind:value={currentAge} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Wiek emerytalny</span>
				<input type="number" min="18" max="120" class="input w-full" bind:value={retirementAge} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Oczekiwana długość życia</span>
				<input type="number" min="18" max="120" class="input w-full" bind:value={lifeExpectancy} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Roczna wypłata na emeryturze (PLN)</span>
				<input type="number" min="0" class="input w-full" bind:value={annualWithdrawal} />
			</label>
		</div>

		<div class="space-y-3 pt-2 border-t border-surface-300-700">
			<div class="text-sm font-semibold">Inflacja</div>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<label class="space-y-1">
					<span class="text-xs font-semibold">Średnia roczna inflacja (%)</span>
					<input type="number" step="0.1" class="input w-full" bind:value={inflationMean} />
				</label>
				<label class="space-y-1">
					<span class="text-xs font-semibold">Zmienność inflacji / odch. std. (%)</span>
					<input
						type="number"
						min="0"
						step="0.1"
						class="input w-full"
						bind:value={inflationVolatility}
					/>
				</label>
			</div>
			<p class="text-xs text-surface-600-400">
				Roczna wypłata jest interpretowana jako kwota w dzisiejszych PLN i rośnie razem z inflacją.
			</p>
		</div>

		<div class="space-y-3 pt-2 border-t border-surface-300-700">
			<label class="flex items-center gap-2 text-sm font-semibold">
				<input type="checkbox" class="checkbox" bind:checked={useAccountMix} />
				Uwzględnij podatek (Belka / IKZE 10%) wg miksu kont
			</label>
			{#if useAccountMix}
				<div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
					<label class="space-y-1">
						<span class="text-xs font-semibold">Konto maklerskie (%)</span>
						<input
							type="number"
							min="0"
							max="100"
							step="1"
							class="input w-full"
							bind:value={mixTaxable}
						/>
					</label>
					<label class="space-y-1">
						<span class="text-xs font-semibold">IKE (%)</span>
						<input
							type="number"
							min="0"
							max="100"
							step="1"
							class="input w-full"
							bind:value={mixIke}
						/>
					</label>
					<label class="space-y-1">
						<span class="text-xs font-semibold">IKZE (%)</span>
						<input
							type="number"
							min="0"
							max="100"
							step="1"
							class="input w-full"
							bind:value={mixIkze}
						/>
					</label>
					<label class="space-y-1">
						<span class="text-xs font-semibold">ZUS (%)</span>
						<input
							type="number"
							min="0"
							max="100"
							step="1"
							class="input w-full"
							bind:value={mixZus}
						/>
					</label>
				</div>
				<div
					class="text-xs {Math.abs(mixSum - 100) < 0.01
						? 'text-success-600-400'
						: 'text-error-600-400'}"
				>
					Suma kont: {mixSum.toFixed(1)}% (musi być 100%)
				</div>
				<label class="space-y-1 block">
					<span class="text-xs font-semibold">
						Udział zysku w koncie maklerskim (%) — pozostała część to wkład własny
					</span>
					<input
						type="number"
						min="0"
						max="100"
						step="1"
						class="input w-full sm:max-w-xs"
						bind:value={mixTaxableGain}
					/>
				</label>
				<p class="text-xs text-surface-600-400">
					Roczna wypłata to kwota po podatku. Portfel jest pomniejszany o brutto, aby pokryć daniny.
					IKE jest wolne od podatku po 60. r.ż., IKZE płaci 10% po 65.
				</p>
			{/if}
		</div>

		<div class="space-y-3 pt-2 border-t border-surface-300-700">
			<label class="flex items-center gap-2 text-sm font-semibold">
				<input type="checkbox" class="checkbox" bind:checked={useAllocation} />
				Wyprowadź stopę zwrotu i zmienność z alokacji portfela
			</label>
			{#if useAllocation}
				<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
					<label class="space-y-1">
						<span class="text-xs font-semibold">Akcje (%)</span>
						<input
							type="number"
							min="0"
							max="100"
							step="1"
							class="input w-full"
							bind:value={allocStocks}
						/>
					</label>
					<label class="space-y-1">
						<span class="text-xs font-semibold">Obligacje (%)</span>
						<input
							type="number"
							min="0"
							max="100"
							step="1"
							class="input w-full"
							bind:value={allocBonds}
						/>
					</label>
					<label class="space-y-1">
						<span class="text-xs font-semibold">Gotówka (%)</span>
						<input
							type="number"
							min="0"
							max="100"
							step="1"
							class="input w-full"
							bind:value={allocCash}
						/>
					</label>
				</div>
				<div
					class="text-xs {Math.abs(allocSum - 100) < 0.01
						? 'text-success-600-400'
						: 'text-error-600-400'}"
				>
					Suma: {allocSum.toFixed(1)}% (musi być 100%)
				</div>
			{/if}
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

			<div class="card preset-tonal-surface p-3 text-sm space-y-1">
				<div class="font-semibold text-xs uppercase text-surface-600-400">Założenia symulacji</div>
				<div>
					Stopa zwrotu: <strong>{result.assumptions.expected_return.toFixed(2)}%</strong>,
					zmienność: <strong>{result.assumptions.volatility.toFixed(2)}%</strong>
					<span class="text-xs text-surface-600-400">
						({result.assumptions.source === 'allocation'
							? 'wyprowadzone z alokacji'
							: 'wpisane ręcznie'})
					</span>
				</div>
				<div class="text-xs text-surface-700-300">
					Inflacja: średnia <strong>{result.assumptions.inflation_mean.toFixed(2)}%</strong>,
					zmienność <strong>{result.assumptions.inflation_volatility.toFixed(2)}%</strong>
				</div>
				{#if result.assumptions.allocation}
					<div class="text-xs text-surface-700-300">
						Alokacja: {result.assumptions.allocation.stocks_pct}% akcje /
						{result.assumptions.allocation.bonds_pct}% obligacje /
						{result.assumptions.allocation.cash_pct}% gotówka
					</div>
				{/if}
			</div>

			<div class="flex flex-wrap gap-4">
				<label class="flex items-center gap-2 text-sm">
					<input type="checkbox" class="checkbox" bind:checked={showReal} />
					Pokaż wartości realne (w dzisiejszych PLN)
				</label>
				{#if result.tax}
					<label class="flex items-center gap-2 text-sm">
						<input type="checkbox" class="checkbox" bind:checked={showNet} />
						Pokaż portfel netto (po likwidacji + Belka/IKZE)
					</label>
				{/if}
			</div>

			{#if result.tax}
				<div class="card preset-tonal-surface p-3 text-sm space-y-1">
					<div class="font-semibold text-xs uppercase text-surface-600-400">
						Podatek (efektywny wymiar życiowy)
					</div>
					<div>
						Stopa efektywna:
						<strong>{(result.tax.effective_lifetime_rate * 100).toFixed(2)}%</strong>
						(wzięte z portfela: {formatPLN(result.tax.gross_withdrawals_total)} PLN, podatek: {formatPLN(
							result.tax.tax_total
						)} PLN)
					</div>
				</div>
			{/if}

			<div bind:this={chartContainer} class="w-full h-[320px] sm:h-[420px]"></div>

			{#if result.bands.length > 0}
				{@const last = result.bands[result.bands.length - 1]}
				{@const netActive = showNet && !!result.tax}
				{@const lastP5 = netActive
					? showReal
						? last.p5_net_real
						: last.p5_net
					: showReal
						? last.p5_real
						: last.p5}
				{@const lastP50 = netActive
					? showReal
						? last.p50_net_real
						: last.p50_net
					: showReal
						? last.p50_real
						: last.p50}
				{@const lastP95 = netActive
					? showReal
						? last.p95_net_real
						: last.p95_net
					: showReal
						? last.p95_real
						: last.p95}
				{@const spendingBand = result.bands.find((b) => b.spending > 0)}
				{@const valueLabel = `${showReal ? 'realne' : 'nominalne'}${netActive ? ', netto' : ''}`}
				<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400">
							Pesymistyczne (P5) w wieku {last.age} ({valueLabel})
						</div>
						<div class="text-xl font-bold">{formatPLN(lastP5)} PLN</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400">
							Mediana (P50) w wieku {last.age} ({valueLabel})
						</div>
						<div class="text-xl font-bold">{formatPLN(lastP50)} PLN</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400">
							Optymistyczne (P95) w wieku {last.age} ({valueLabel})
						</div>
						<div class="text-xl font-bold">{formatPLN(lastP95)} PLN</div>
					</div>
				</div>
				{#if spendingBand}
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400">
							Średnia wypłata na początku emerytury (wiek {spendingBand.age}, {valueLabel})
						</div>
						<div class="text-xl font-bold">
							{formatPLN(showReal ? spendingBand.spending_real : spendingBand.spending)} PLN
						</div>
					</div>
				{/if}
			{/if}
		</section>
	{/if}
</div>
