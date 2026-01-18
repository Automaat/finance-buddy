<script lang="ts">
	import Card from '$lib/components/Card.svelte';
	import CardHeader from '$lib/components/CardHeader.svelte';
	import CardTitle from '$lib/components/CardTitle.svelte';
	import CardContent from '$lib/components/CardContent.svelte';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import { formatPLN } from '$lib/utils/format';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';

	// Default values (Polish retirement)
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

	// Form state
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

	let error = '';
	let saving = false;

	// Reactive validation and calculations
	$: marketSum = allocationStocks + allocationBonds + allocationGold + allocationCommodities;
	$: isValidAllocation = marketSum === 100;

	// Calculate current age and years until retirement
	$: currentAge = birthDate
		? Math.max(
				0,
				Math.floor(
					(new Date().getTime() - new Date(birthDate).getTime()) / (365.25 * 24 * 60 * 60 * 1000)
				)
			)
		: 0;
	$: yearsUntilRetirement = Math.max(0, retirementAge - currentAge);

	// Calculate required capital using 4% safe withdrawal rate (25x annual expenses)
	$: requiredCapital = retirementMonthlySalary * 12 * 25;

	// Calculate monthly savings needed (simple linear calculation without investment returns)
	// Subtract already saved money in retirement accounts from required capital
	$: remainingCapital = Math.max(0, requiredCapital - (data.retirementAccountValue ?? 0));
	$: monthlySavingsNeeded =
		yearsUntilRetirement > 0 ? remainingCapital / (yearsUntilRetirement * 12) : 0;

	async function saveConfig() {
		if (!isValidAllocation) {
			return;
		}

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
					allocation_commodities: allocationCommodities
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

<div class="container">
	<h1>⚙️ Konfiguracja</h1>

	{#if data.isFirstTime}
		<div class="info-banner">
			ℹ️ Konfiguracja nie istnieje. Poniżej znajdują się wartości domyślne.
		</div>
	{/if}

	<Card>
		<CardHeader>
			<CardTitle>🏖️ Emerytura</CardTitle>
		</CardHeader>
		<CardContent>
			<div class="form-group">
				<label for="birth-date">Data urodzenia</label>
				<input id="birth-date" type="date" bind:value={birthDate} class="input" />
				{#if currentAge > 0}
					<div class="field-hint">Obecny wiek: {currentAge} lat</div>
				{/if}
			</div>
			<div class="form-group">
				<label for="retirement-age">Wiek emerytalny</label>
				<input
					id="retirement-age"
					type="number"
					min="18"
					max="100"
					bind:value={retirementAge}
					class="input"
				/>
				{#if yearsUntilRetirement > 0}
					<div class="field-hint">Za {yearsUntilRetirement} lat</div>
				{/if}
			</div>
			<div class="form-group">
				<label for="retirement-salary">Oczekiwany miesięczny dochód emerytalny (PLN)</label>
				<input
					id="retirement-salary"
					type="number"
					min="0"
					step="100"
					bind:value={retirementMonthlySalary}
					class="input"
				/>
			</div>
			<div class="calculated-info">
				<div class="info-label">Potrzebny kapitał (reguła 4%):</div>
				<div class="info-value">{formatPLN(requiredCapital)}</div>
			</div>
			{#if data.retirementAccountValue && data.retirementAccountValue > 0}
				<div class="calculated-info">
					<div class="info-label">Wartość kont emerytalnych:</div>
					<div class="info-value">{formatPLN(data.retirementAccountValue)}</div>
				</div>
				<div class="calculated-info">
					<div class="info-label">Pozostało do zgromadzenia:</div>
					<div class="info-value">{formatPLN(remainingCapital)}</div>
				</div>
			{/if}
			{#if yearsUntilRetirement > 0}
				<div class="calculated-info">
					<div class="info-label">Miesięczne oszczędności potrzebne:</div>
					<div class="info-value">{formatPLN(monthlySavingsNeeded)}</div>
				</div>
			{:else if currentAge > 0}
				<div class="calculated-info">
					<div class="info-label">Już w wieku emerytalnym</div>
				</div>
			{/if}
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>📊 Docelowa alokacja inwestycyjna</CardTitle>
		</CardHeader>
		<CardContent>
			<div class="allocation-section">
				<h3 class="section-title">🏡 Nieruchomości</h3>
				<div class="form-group">
					<label for="allocation-real-estate">Nieruchomości (%)</label>
					<input
						id="allocation-real-estate"
						type="number"
						min="0"
						max="100"
						bind:value={allocationRealEstate}
						class="input"
					/>
				</div>
			</div>

			<div class="allocation-section">
				<h3 class="section-title">📈 Część rynkowa</h3>
				<div class="form-group">
					<label for="allocation-stocks">Akcje (%)</label>
					<input
						id="allocation-stocks"
						type="number"
						min="0"
						max="100"
						bind:value={allocationStocks}
						class="input"
					/>
				</div>
				<div class="form-group">
					<label for="allocation-bonds">Obligacje (%)</label>
					<input
						id="allocation-bonds"
						type="number"
						min="0"
						max="100"
						bind:value={allocationBonds}
						class="input"
					/>
				</div>
				<div class="form-group">
					<label for="allocation-gold">Złoto (%)</label>
					<input
						id="allocation-gold"
						type="number"
						min="0"
						max="100"
						bind:value={allocationGold}
						class="input"
					/>
				</div>
				<div class="form-group">
					<label for="allocation-commodities">Surowce (%)</label>
					<input
						id="allocation-commodities"
						type="number"
						min="0"
						max="100"
						bind:value={allocationCommodities}
						class="input"
					/>
				</div>
				<div
					class="allocation-sum"
					class:valid={isValidAllocation}
					class:invalid={!isValidAllocation}
				>
					Suma części rynkowej: {marketSum}%
					{#if isValidAllocation}
						<span class="check">✓</span>
					{:else}
						<span class="warning">⚠️</span>
					{/if}
				</div>
			</div>
		</CardContent>
	</Card>

	{#if error}
		<div class="error-message">{error}</div>
	{/if}

	<button class="save-button" on:click={saveConfig} disabled={!isValidAllocation || saving}>
		{saving ? 'Zapisywanie...' : 'Zapisz konfigurację'}
	</button>
</div>

<style>
	.container {
		max-width: 800px;
		margin: 0 auto;
		padding: var(--size-4);
	}

	h1 {
		margin-bottom: var(--size-4);
		font-size: var(--font-size-5);
		color: var(--color-text);
	}

	.info-banner {
		background: rgba(136, 192, 208, 0.1);
		border: 1px solid var(--nord8);
		border-radius: var(--radius-2);
		padding: var(--size-3);
		margin-bottom: var(--size-4);
		color: var(--nord10);
	}

	.form-group {
		margin-bottom: var(--size-3);
	}

	label {
		display: block;
		margin-bottom: var(--size-2);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
	}

	.input {
		width: 100%;
		padding: var(--size-2) var(--size-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
		background: var(--color-bg);
		color: var(--color-text);
		font-family: inherit;
		transition: all 0.2s;
	}

	.input:focus {
		outline: none;
		border-color: var(--color-primary);
		box-shadow: 0 0 0 2px rgba(94, 129, 172, 0.2);
	}

	.allocation-sum {
		margin-top: var(--size-4);
		padding: var(--size-3);
		border-radius: var(--radius-2);
		font-weight: var(--font-weight-6);
		display: flex;
		align-items: center;
		gap: var(--size-2);
	}

	.allocation-sum.valid {
		background: rgba(163, 190, 140, 0.15);
		color: var(--color-success);
		border: 1px solid var(--color-success);
	}

	.allocation-sum.invalid {
		background: rgba(191, 97, 106, 0.15);
		color: var(--color-error);
		border: 1px solid var(--color-error);
	}

	.check {
		color: var(--color-success);
	}

	.warning {
		color: var(--color-error);
	}

	.error-message {
		background: rgba(191, 97, 106, 0.15);
		border: 1px solid var(--color-error);
		border-radius: var(--radius-2);
		padding: var(--size-3);
		margin-top: var(--size-4);
		color: var(--color-error);
	}

	.save-button {
		margin-top: var(--size-4);
		padding: var(--size-3) var(--size-5);
		background: var(--color-primary);
		color: var(--nord6);
		border: none;
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
		font-weight: var(--font-weight-6);
		cursor: pointer;
		transition: all 0.2s;
	}

	.save-button:hover:not(:disabled) {
		background: var(--nord9);
	}

	.save-button:disabled {
		background: var(--nord4);
		cursor: not-allowed;
		opacity: 0.6;
	}

	.calculated-info {
		margin-top: var(--size-4);
		padding: var(--size-3);
		background: rgba(143, 188, 187, 0.1);
		border: 1px solid var(--nord7);
		border-radius: var(--radius-2);
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.info-label {
		color: var(--color-text);
		font-weight: var(--font-weight-5);
		font-size: var(--font-size-2);
	}

	.info-value {
		color: var(--nord10);
		font-weight: var(--font-weight-7);
		font-size: var(--font-size-4);
	}

	.field-hint {
		margin-top: var(--size-1);
		font-size: var(--font-size-1);
		color: var(--color-text-secondary);
		font-style: italic;
	}

	.allocation-section {
		margin-bottom: var(--size-5);
		padding-bottom: var(--size-4);
		border-bottom: 1px solid var(--color-border);
	}

	.allocation-section:last-of-type {
		border-bottom: none;
	}

	.section-title {
		font-size: var(--font-size-3);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		margin-bottom: var(--size-3);
	}
</style>
