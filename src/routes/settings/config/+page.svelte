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
		Gauge,
		Flame
	} from 'lucide-svelte';
	import { resolveApiUrl } from '$lib/api';
	import { invalidateAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const apiUrl = resolveApiUrl();

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

	const config = untrack(() => data.config);
	let birthDate = $state(config?.birth_date ?? defaults.birth_date);
	let retirementAge = $state(config?.retirement_age ?? defaults.retirement_age);
	// Money fields arrive as JSON strings (backend-go serializes decimals as
	// strings); coerce to number for the numeric inputs + projections below.
	let retirementMonthlySalary = $state(
		Number(config?.retirement_monthly_salary ?? defaults.retirement_monthly_salary)
	);
	let allocationRealEstate = $state(
		config?.allocation_real_estate ?? defaults.allocation_real_estate
	);
	let allocationStocks = $state(config?.allocation_stocks ?? defaults.allocation_stocks);
	let allocationBonds = $state(config?.allocation_bonds ?? defaults.allocation_bonds);
	let allocationGold = $state(config?.allocation_gold ?? defaults.allocation_gold);
	let allocationCommodities = $state(
		config?.allocation_commodities ?? defaults.allocation_commodities
	);
	let monthlyExpenses = $state(Number(config?.monthly_expenses ?? 0));
	let monthlyMortgagePayment = $state(Number(config?.monthly_mortgage_payment ?? 0));
	let withdrawalRate = $state(Number(config?.withdrawal_rate ?? 0.04));
	// Coast FIRE: target age is optional — empty input means "no Coast FIRE
	// tile". The backend treats a null coast_fire_target_age the same way.
	let coastFireTargetAge = $state<number | null>(config?.coast_fire_target_age ?? null);
	let expectedReturnRate = $state(Number(config?.expected_return_rate ?? 0.07));
	// Barista FIRE: monthly part-time income, nullable. Backend hides the
	// tile when null. Money field arrives as a JSON string when present.
	let baristaMonthlyIncome = $state<number | null>(
		config?.barista_monthly_income != null ? Number(config.barista_monthly_income) : null
	);
	// FIRE bands: optional Lean / Fat monthly expenses. Base = monthly_expenses
	// (the existing field). Each band is independently nullable; null = band
	// tile hidden on the dashboard.
	let leanMonthlyExpenses = $state<number | null>(
		config?.lean_monthly_expenses != null ? Number(config.lean_monthly_expenses) : null
	);
	let fatMonthlyExpenses = $state<number | null>(
		config?.fat_monthly_expenses != null ? Number(config.fat_monthly_expenses) : null
	);

	let error = $state('');
	let saving = $state(false);

	const marketSum = $derived(
		allocationStocks + allocationBonds + allocationGold + allocationCommodities
	);
	const isValidAllocation = $derived(marketSum === 100);

	const currentAge = $derived(
		birthDate
			? Math.max(
					0,
					Math.floor(
						(new Date().getTime() - new Date(birthDate).getTime()) / (365.25 * 24 * 60 * 60 * 1000)
					)
				)
			: 0
	);
	const yearsUntilRetirement = $derived(Math.max(0, retirementAge - currentAge));

	const requiredCapital = $derived(retirementMonthlySalary * 12 * 25);

	const remainingCapital = $derived(
		Math.max(0, requiredCapital - (data.retirementAccountValue ?? 0))
	);

	const monthlySavingsNeeded = $derived.by(() => {
		if (yearsUntilRetirement <= 0) return 0;

		const annualReturnRate = 0.07;
		const monthlyRate = annualReturnRate / 12;
		const months = yearsUntilRetirement * 12;

		const futureValueOfSavings =
			(data.retirementAccountValue ?? 0) * Math.pow(1 + annualReturnRate, yearsUntilRetirement);
		const adjustedTarget = Math.max(0, requiredCapital - futureValueOfSavings);

		if (adjustedTarget === 0) return 0;

		return (adjustedTarget * monthlyRate) / (Math.pow(1 + monthlyRate, months) - 1);
	});

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
					monthly_mortgage_payment: monthlyMortgagePayment,
					withdrawal_rate: withdrawalRate,
					coast_fire_target_age: coastFireTargetAge,
					expected_return_rate: expectedReturnRate,
					barista_monthly_income: baristaMonthlyIncome,
					lean_monthly_expenses: leanMonthlyExpenses,
					fat_monthly_expenses: fatMonthlyExpenses
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

		<label class="label">
			<span class="font-semibold text-sm">Bezpieczna stopa wypłaty (FIRE)</span>
			<select id="withdrawal-rate" class="select" bind:value={withdrawalRate}>
				<option value={0.03}>3% — bardzo ostrożna (~33× wydatków)</option>
				<option value={0.035}>3.5% — ostrożna (~28.6× wydatków)</option>
				<option value={0.04}>4% — klasyczna reguła Trinity (×25)</option>
			</select>
			<span class="text-xs italic text-surface-700-300"
				>Wyznacza cel FIRE = roczne wydatki ÷ stopa wypłaty.</span
			>
		</label>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Flame size={20} /> Coast FIRE</h3>
		</header>

		<label class="label">
			<span class="font-semibold text-sm">Docelowy wiek Coast FIRE</span>
			<input
				id="coast-fire-target-age"
				type="number"
				min="18"
				max="100"
				step="1"
				placeholder="np. 65 (puste = wyłączone)"
				value={coastFireTargetAge ?? ''}
				oninput={(e) => {
					const v = (e.target as HTMLInputElement).value;
					// Backend stores target age as *int; force integer + drop NaN so
					// the JSON encoder never sends a float that fails to decode.
					if (v === '') {
						coastFireTargetAge = null;
						return;
					}
					const n = parseInt(v, 10);
					coastFireTargetAge = Number.isNaN(n) ? null : n;
				}}
				class="input"
			/>
			<span class="text-xs italic text-surface-700-300">
				Wiek, w którym chcesz osiągnąć FIRE bez dalszych wpłat. Puste = ukryj kartę Coast FIRE.
			</span>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Oczekiwana stopa zwrotu (rocznie)</span>
			<input
				id="expected-return-rate"
				type="number"
				min="0.01"
				max="0.20"
				step="0.005"
				bind:value={expectedReturnRate}
				class="input"
			/>
			<span class="text-xs italic text-surface-700-300">
				Realna stopa zwrotu używana do dyskontowania celu FIRE do dziś (np. 0.07 = 7%).
			</span>
		</label>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Flame size={20} /> Barista FIRE</h3>
		</header>

		<label class="label">
			<span class="font-semibold text-sm">Miesięczny dochód z pracy dorywczej (PLN)</span>
			<input
				id="barista-monthly-income"
				type="number"
				min="0"
				step="100"
				placeholder="np. 3000 (puste = wyłączone)"
				value={baristaMonthlyIncome ?? ''}
				oninput={(e) => {
					const v = (e.target as HTMLInputElement).value;
					if (v === '') {
						baristaMonthlyIncome = null;
						return;
					}
					const n = Number(v);
					baristaMonthlyIncome = Number.isNaN(n) ? null : n;
				}}
				class="input"
			/>
			<span class="text-xs italic text-surface-700-300">
				Dochód z pracy na pół etatu pomniejsza wymagany kapitał. Puste = ukryj kartę Barista FIRE.
			</span>
		</label>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Flame size={20} /> Pasma FIRE (Lean / Base / Fat)</h3>
		</header>

		<p class="text-xs italic text-surface-700-300">
			Bazowe pasmo używa „Miesięczne wydatki” powyżej. Lean i Fat to opcjonalne warianty
			oszczędnościowy/luksusowy — puste = ukryj kartę.
		</p>

		<label class="label">
			<span class="font-semibold text-sm">Lean FIRE — miesięczne wydatki (PLN)</span>
			<input
				id="lean-monthly-expenses"
				type="number"
				min="0"
				step="100"
				placeholder="np. 3000 (puste = wyłączone)"
				value={leanMonthlyExpenses ?? ''}
				oninput={(e) => {
					const v = (e.target as HTMLInputElement).value;
					if (v === '') {
						leanMonthlyExpenses = null;
						return;
					}
					const n = Number(v);
					leanMonthlyExpenses = Number.isNaN(n) ? null : n;
				}}
				class="input"
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Fat FIRE — miesięczne wydatki (PLN)</span>
			<input
				id="fat-monthly-expenses"
				type="number"
				min="0"
				step="100"
				placeholder="np. 12000 (puste = wyłączone)"
				value={fatMonthlyExpenses ?? ''}
				oninput={(e) => {
					const v = (e.target as HTMLInputElement).value;
					if (v === '') {
						fatMonthlyExpenses = null;
						return;
					}
					const n = Number(v);
					fatMonthlyExpenses = Number.isNaN(n) ? null : n;
				}}
				class="input"
			/>
		</label>
	</div>

	{#if error}
		<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
	{/if}

	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto"
		onclick={saveConfig}
		disabled={!isValidAllocation || saving}
	>
		{saving ? 'Zapisywanie...' : 'Zapisz konfigurację'}
	</button>
</div>
