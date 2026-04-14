<script lang="ts">
	import { formatPLN } from '$lib/utils/format';
	import {
		Settings,
		Umbrella,
		PieChart,
		Home,
		TrendingUp,
		Info,
		CheckCircle2,
		AlertTriangle,
		Gauge
	} from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';

	const defaults = {
		birth_date: '1990-01-01',
		retirement_age: 67,
		retirement_monthly_salary: 8000,
		allocation_real_estate: 20,
		allocation_stocks: 48,
		allocation_bonds: 24,
		allocation_gold: 5,
		allocation_commodities: 3
	};

	let birthDate = data.config?.birth_date ?? defaults.birth_date;
	let retirementAge = data.config?.retirement_age ?? defaults.retirement_age;
	let retirementMonthlySalary =
		data.config?.retirement_monthly_salary ?? defaults.retirement_monthly_salary;
	let allocationRealEstate = data.config?.allocation_real_estate ?? defaults.allocation_real_estate;
	let allocationStocks = data.config?.allocation_stocks ?? defaults.allocation_stocks;
	let allocationBonds = data.config?.allocation_bonds ?? defaults.allocation_bonds;
	let allocationGold = data.config?.allocation_gold ?? defaults.allocation_gold;
	let allocationCommodities =
		data.config?.allocation_commodities ?? defaults.allocation_commodities;
	let monthlyExpenses = data.config?.monthly_expenses ?? 0;
	let monthlyMortgagePayment = data.config?.monthly_mortgage_payment ?? 0;

	let error = '';
	let saving = false;

	$: marketSum = allocationStocks + allocationBonds + allocationGold + allocationCommodities;
	$: isValidAllocation = marketSum === 100;

	$: currentAge = birthDate
		? Math.max(
				0,
				Math.floor(
					(new Date().getTime() - new Date(birthDate).getTime()) / (365.25 * 24 * 60 * 60 * 1000)
				)
			)
		: 0;
	$: yearsUntilRetirement = Math.max(0, retirementAge - currentAge);

	$: requiredCapital = retirementMonthlySalary * 12 * 25;

	$: remainingCapital = Math.max(0, requiredCapital - (data.retirementAccountValue ?? 0));

	$: monthlySavingsNeeded = (() => {
		if (yearsUntilRetirement <= 0) return 0;

		const annualReturnRate = 0.07;
		const monthlyRate = annualReturnRate / 12;
		const months = yearsUntilRetirement * 12;

		const futureValueOfSavings =
			(data.retirementAccountValue ?? 0) * Math.pow(1 + annualReturnRate, yearsUntilRetirement);
		const adjustedTarget = Math.max(0, requiredCapital - futureValueOfSavings);

		if (adjustedTarget === 0) return 0;

		return (adjustedTarget * monthlyRate) / (Math.pow(1 + monthlyRate, months) - 1);
	})();

	async function saveConfig() {
		if (!isValidAllocation) return;

		error = '';
		saving = true;

		try {
			const response = await fetch(`${apiUrl}/api/config`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					birth_date: birthDate,
					retirement_age: retirementAge,
					retirement_monthly_salary: retirementMonthlySalary,
					allocation_real_estate: allocationRealEstate,
					allocation_stocks: allocationStocks,
					allocation_bonds: allocationBonds,
					allocation_gold: allocationGold,
					allocation_commodities: allocationCommodities,
					monthly_expenses: monthlyExpenses,
					monthly_mortgage_payment: monthlyMortgagePayment
				})
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.detail || 'Failed to save configuration');
			}

			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) {
				error = err.message;
			} else {
				error = String(err) || 'Unknown error';
			}
		} finally {
			saving = false;
		}
	}
</script>

<div class="max-w-3xl mx-auto space-y-4">
	<h1 class="h2 flex items-center gap-2"><Settings size={24} /> Konfiguracja</h1>

	{#if data.isFirstTime}
		<div class="card preset-tonal-primary p-3 text-sm flex items-center gap-2">
			<Info size={16} />
			Konfiguracja nie istnieje. Poniżej znajdują się wartości domyślne.
		</div>
	{/if}

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Umbrella size={20} /> Emerytura</h3>
		</header>

		<label class="label">
			<span class="font-semibold text-sm">Data urodzenia</span>
			<input id="birth-date" type="date" bind:value={birthDate} class="input" />
			{#if currentAge > 0}
				<span class="text-xs italic text-surface-700-300">Obecny wiek: {currentAge} lat</span>
			{/if}
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Wiek emerytalny</span>
			<input
				id="retirement-age"
				type="number"
				min="18"
				max="100"
				bind:value={retirementAge}
				class="input"
			/>
			{#if yearsUntilRetirement > 0}
				<span class="text-xs italic text-surface-700-300">Za {yearsUntilRetirement} lat</span>
			{/if}
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Oczekiwany miesięczny dochód emerytalny (PLN)</span>
			<input
				id="retirement-salary"
				type="number"
				min="0"
				step="100"
				bind:value={retirementMonthlySalary}
				class="input"
			/>
		</label>

		<div class="card preset-tonal-primary p-3 flex justify-between items-center flex-wrap gap-2">
			<div class="text-sm">Potrzebny kapitał (reguła 4%):</div>
			<div class="text-lg font-bold">{formatPLN(requiredCapital)}</div>
		</div>
		{#if data.retirementAccountValue && data.retirementAccountValue > 0}
			<div class="card preset-tonal-primary p-3 flex justify-between items-center flex-wrap gap-2">
				<div class="text-sm">Wartość kont emerytalnych:</div>
				<div class="text-lg font-bold">{formatPLN(data.retirementAccountValue)}</div>
			</div>
			<div class="card preset-tonal-primary p-3 flex justify-between items-center flex-wrap gap-2">
				<div class="text-sm">Pozostało do zgromadzenia:</div>
				<div class="text-lg font-bold">{formatPLN(remainingCapital)}</div>
			</div>
		{/if}
		{#if yearsUntilRetirement > 0}
			<div class="card preset-tonal-primary p-3 flex justify-between items-center flex-wrap gap-2">
				<div class="text-sm">Miesięczne oszczędności potrzebne (przy 7% rocznie):</div>
				<div class="text-lg font-bold">{formatPLN(monthlySavingsNeeded)}</div>
			</div>
		{:else if currentAge > 0}
			<div class="card preset-tonal-primary p-3 text-sm">Już w wieku emerytalnym</div>
		{/if}
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2">
				<PieChart size={20} /> Docelowa alokacja inwestycyjna
			</h3>
		</header>

		<section class="space-y-3 pb-4 border-b border-surface-200-800">
			<h4 class="h5 flex items-center gap-2"><Home size={18} /> Nieruchomości</h4>
			<label class="label">
				<span class="font-semibold text-sm">Nieruchomości (%)</span>
				<input
					id="allocation-real-estate"
					type="number"
					min="0"
					max="100"
					bind:value={allocationRealEstate}
					class="input"
				/>
			</label>
		</section>

		<section class="space-y-3">
			<h4 class="h5 flex items-center gap-2"><TrendingUp size={18} /> Część rynkowa</h4>
			<label class="label">
				<span class="font-semibold text-sm">Akcje (%)</span>
				<input
					id="allocation-stocks"
					type="number"
					min="0"
					max="100"
					bind:value={allocationStocks}
					class="input"
				/>
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Obligacje (%)</span>
				<input
					id="allocation-bonds"
					type="number"
					min="0"
					max="100"
					bind:value={allocationBonds}
					class="input"
				/>
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Złoto (%)</span>
				<input
					id="allocation-gold"
					type="number"
					min="0"
					max="100"
					bind:value={allocationGold}
					class="input"
				/>
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Surowce (%)</span>
				<input
					id="allocation-commodities"
					type="number"
					min="0"
					max="100"
					bind:value={allocationCommodities}
					class="input"
				/>
			</label>

			<div
				class="card p-3 font-semibold flex items-center gap-2 {isValidAllocation
					? 'preset-filled-success-500'
					: 'preset-filled-error-500'}"
			>
				Suma części rynkowej: {marketSum}%
				{#if isValidAllocation}
					<CheckCircle2 size={16} />
				{:else}
					<AlertTriangle size={16} />
				{/if}
			</div>
		</section>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Gauge size={20} /> Metryki</h3>
		</header>

		<label class="label">
			<span class="font-semibold text-sm">Miesięczne wydatki (PLN)</span>
			<input
				id="monthly-expenses"
				type="number"
				min="0"
				step="100"
				bind:value={monthlyExpenses}
				class="input"
			/>
			<span class="text-xs italic text-surface-700-300"
				>Używane do obliczenia miesięcy funduszu awaryjnego</span
			>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Miesięczna rata hipoteki (PLN)</span>
			<input
				id="monthly-mortgage"
				type="number"
				min="0"
				step="100"
				bind:value={monthlyMortgagePayment}
				class="input"
			/>
			<span class="text-xs italic text-surface-700-300"
				>Używane do obliczenia czasu spłaty hipoteki</span
			>
		</label>
	</div>

	{#if error}
		<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
	{/if}

	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto"
		on:click={saveConfig}
		disabled={!isValidAllocation || saving}
	>
		{saving ? 'Zapisywanie...' : 'Zapisz konfigurację'}
	</button>
</div>
