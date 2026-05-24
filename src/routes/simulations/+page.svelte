<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { resolveApiUrl } from '$lib/api';
	import * as echarts from 'echarts';
	import type { PageData } from './$types';
	import {
		buildRetirementByWrapperOption,
		buildRetirementProjectionOption,
		getTotalBalanceAtAge,
		type AccountSimulation
	} from '$lib/utils/charts/simulations';
	import { createChart, type ChartHandle } from '$lib/utils/charts/lifecycle';
	import { ownerName, type OwnerOption } from '$lib/types/owners';

	interface SimulationSummary {
		total_final_balance: number;
		total_contributions: number;
		total_returns: number;
		total_tax_savings: number;
		total_subsidies?: number;
		estimated_monthly_income: number;
		estimated_monthly_income_today: number;
		years_until_retirement: number;
	}

	interface SimulationResponse {
		simulations: AccountSimulation[];
		summary: SimulationSummary;
	}

	interface IkeIkzeConfig {
		enabled: boolean;
		wrapper: string;
		ownerUserId: number;
		balance: number;
		autoFill: boolean;
		monthly: number;
		taxRate: number;
	}

	interface PpkConfig {
		enabled: boolean;
		ownerUserId: number;
		balance: number;
		salary: number;
		employeeRate: number;
		employerRate: number;
		belowThreshold: boolean;
		includeSubsidies: boolean;
	}

	interface BrokerageConfig {
		enabled: boolean;
		ownerUserId: number;
		balance: number;
		monthly: number;
	}

	const SALARY_THRESHOLD_2026 = 5767;

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const owners = $derived((data.owners || []) as OwnerOption[]);

	let currentAge = $state(untrack(() => data.current_age));
	let retirementAge = $state(untrack(() => data.retirement_age));

	// Dynamic IKE/IKZE accounts - one per owner per wrapper
	let ikeIkzeAccounts: IkeIkzeConfig[] = $state([]);
	let ppkAccounts: PpkConfig[] = $state([]);
	let brokerageAccounts: BrokerageConfig[] = $state([]);

	function initAccounts() {
		ikeIkzeAccounts = [];
		ppkAccounts = [];
		brokerageAccounts = [];

		for (const owner of owners) {
			const key = String(owner.id);

			for (const wrapper of ['IKE', 'IKZE']) {
				const balanceKey = `${wrapper.toLowerCase()}_${key}`;
				ikeIkzeAccounts.push({
					enabled: true,
					wrapper,
					ownerUserId: owner.id,
					balance: data.balances?.[balanceKey] ?? 0,
					autoFill: false,
					monthly: 0,
					taxRate: wrapper === 'IKZE' ? 17.0 : 0
				});
			}

			ppkAccounts.push({
				enabled: false,
				ownerUserId: owner.id,
				balance: data.ppk_balances?.[key] ?? 0,
				salary: data.monthly_salaries?.[key] ?? 10000,
				employeeRate: data.ppk_rates?.[key]?.employee ?? 2.0,
				employerRate: data.ppk_rates?.[key]?.employer ?? 1.5,
				belowThreshold: false,
				includeSubsidies: true
			});

			brokerageAccounts.push({
				enabled: false,
				ownerUserId: owner.id,
				balance: 0,
				monthly: 0
			});
		}
	}

	$effect(() => {
		if (owners.length > 0 && ikeIkzeAccounts.length === 0) {
			initAccounts();
		}
	});

	// Assumptions
	let annualReturnRate = $state(7.0);
	let limitGrowthRate = $state(5.0);
	let expectedSalaryGrowth = $state(3.0);
	let inflationRate = $state(3.0);

	// Results
	let results: SimulationResponse | null = $state(null);
	let loading = $state(false);
	let error = $state('');

	// --- Saved scenarios (issue #547) ---
	interface RetirementScenarioInputs {
		current_age: number;
		retirement_age: number;
		ike_ikze_accounts: IkeIkzeConfig[];
		ppk_accounts: PpkConfig[];
		brokerage_accounts: BrokerageConfig[];
		annual_return_rate: number;
		limit_growth_rate: number;
		expected_salary_growth: number;
		inflation_rate: number;
	}

	interface SavedScenario {
		id: number;
		name: string;
		kind: string;
		inputs_json: RetirementScenarioInputs;
		created_at: string;
		updated_at: string;
	}

	let scenarios: SavedScenario[] = $state([]);
	let scenarioName = $state('');
	let scenarioBusy = $state(false);
	let scenarioError = $state('');

	function snapshotInputs(): RetirementScenarioInputs {
		return {
			current_age: currentAge,
			retirement_age: retirementAge,
			ike_ikze_accounts: ikeIkzeAccounts.map((a) => ({ ...a })),
			ppk_accounts: ppkAccounts.map((a) => ({ ...a })),
			brokerage_accounts: brokerageAccounts.map((a) => ({ ...a })),
			annual_return_rate: annualReturnRate,
			limit_growth_rate: limitGrowthRate,
			expected_salary_growth: expectedSalaryGrowth,
			inflation_rate: inflationRate
		};
	}

	function applyInputs(inputs: RetirementScenarioInputs) {
		currentAge = inputs.current_age;
		retirementAge = inputs.retirement_age;
		ikeIkzeAccounts = (inputs.ike_ikze_accounts ?? []).map((a) => ({ ...a }));
		ppkAccounts = (inputs.ppk_accounts ?? []).map((a) => ({ ...a }));
		brokerageAccounts = (inputs.brokerage_accounts ?? []).map((a) => ({ ...a }));
		annualReturnRate = inputs.annual_return_rate;
		limitGrowthRate = inputs.limit_growth_rate;
		expectedSalaryGrowth = inputs.expected_salary_growth;
		inflationRate = inputs.inflation_rate;
	}

	async function loadScenarios() {
		try {
			const r = await fetch(`${resolveApiUrl()}/api/scenarios?kind=retirement`);
			if (!r.ok) throw new Error(`HTTP ${r.status}`);
			const body = await r.json();
			scenarios = (body.scenarios ?? []) as SavedScenario[];
		} catch (err) {
			console.error('Failed to load scenarios:', err);
			scenarioError = err instanceof Error ? err.message : 'Nie udało się pobrać scenariuszy';
		}
	}

	async function saveCurrentScenario() {
		const trimmed = scenarioName.trim();
		if (!trimmed) {
			scenarioError = 'Wprowadź nazwę scenariusza';
			return;
		}
		scenarioBusy = true;
		scenarioError = '';
		try {
			const r = await fetch(`${resolveApiUrl()}/api/scenarios`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name: trimmed,
					kind: 'retirement',
					inputs_json: snapshotInputs()
				})
			});
			if (!r.ok) {
				const body = await r.json().catch(() => ({}));
				throw new Error(body.detail?.[0]?.msg ?? `HTTP ${r.status}`);
			}
			scenarioName = '';
			await loadScenarios();
		} catch (err) {
			scenarioError = err instanceof Error ? err.message : String(err);
		} finally {
			scenarioBusy = false;
		}
	}

	async function cloneScenario(id: number) {
		scenarioBusy = true;
		scenarioError = '';
		try {
			const r = await fetch(`${resolveApiUrl()}/api/scenarios/${id}/clone`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({})
			});
			if (!r.ok) throw new Error(`HTTP ${r.status}`);
			await loadScenarios();
		} catch (err) {
			scenarioError = err instanceof Error ? err.message : String(err);
		} finally {
			scenarioBusy = false;
		}
	}

	function loadScenarioIntoForm(s: SavedScenario) {
		applyInputs(s.inputs_json);
		scenarioError = '';
	}

	async function deleteScenario(id: number) {
		if (!confirm('Usunąć ten scenariusz?')) return;
		scenarioBusy = true;
		scenarioError = '';
		try {
			const r = await fetch(`${resolveApiUrl()}/api/scenarios/${id}`, { method: 'DELETE' });
			if (!r.ok && r.status !== 204) throw new Error(`HTTP ${r.status}`);
			await loadScenarios();
		} catch (err) {
			scenarioError = err instanceof Error ? err.message : String(err);
		} finally {
			scenarioBusy = false;
		}
	}

	onMount(() => {
		loadScenarios();
	});

	// Charts
	const MILESTONE_AGES = [60, 65, 70];
	let chartContainer: HTMLDivElement | undefined = $state();
	let wrapperChartContainer: HTMLDivElement | undefined = $state();
	let chart: echarts.ECharts | null = null;
	let chartHandle: ChartHandle | null = null;
	let wrapperChart: echarts.ECharts | null = null;
	let wrapperChartHandle: ChartHandle | null = null;

	const milestoneBalances = $derived.by(() => {
		const r = results;
		if (!r) return [];
		return MILESTONE_AGES.map((age) => ({
			age,
			balance: getTotalBalanceAtAge(r.simulations, age)
		}));
	});

	async function runSimulation() {
		loading = true;
		error = '';
		results = null;

		try {
			// Validate PPK inputs
			for (const ppk of ppkAccounts) {
				if (!ppk.enabled) continue;
				const label = ownerName(owners, ppk.ownerUserId);
				if (ppk.salary > SALARY_THRESHOLD_2026 && ppk.belowThreshold) {
					error = `PPK ${label}: Wynagrodzenie przekracza próg (${SALARY_THRESHOLD_2026} PLN)`;
					loading = false;
					return;
				}
				if (ppk.employeeRate < 0.5 || ppk.employeeRate > 4.0) {
					error = `PPK ${label}: Składka pracownika musi być w zakresie 0.5-4%`;
					loading = false;
					return;
				}
				if (ppk.employerRate < 1.5 || ppk.employerRate > 4.0) {
					error = `PPK ${label}: Składka pracodawcy musi być w zakresie 1.5-4%`;
					loading = false;
					return;
				}
			}

			const apiUrl = resolveApiUrl();

			const requestBody = {
				current_age: currentAge,
				retirement_age: retirementAge,
				ike_ikze_accounts: ikeIkzeAccounts.map((a) => ({
					enabled: a.enabled,
					wrapper: a.wrapper,
					owner_user_id: a.ownerUserId,
					balance: a.balance,
					auto_fill_limit: a.autoFill,
					monthly_contribution: a.monthly,
					tax_rate: a.taxRate
				})),
				ppk_accounts: ppkAccounts
					.filter((p) => p.enabled)
					.map((p) => ({
						owner_user_id: p.ownerUserId,
						enabled: true,
						starting_balance: p.balance,
						monthly_gross_salary: p.salary,
						employee_rate: p.employeeRate,
						employer_rate: p.employerRate,
						salary_below_threshold: p.belowThreshold,
						include_welcome_bonus: p.includeSubsidies,
						include_annual_subsidy: p.includeSubsidies
					})),
				brokerage_accounts: brokerageAccounts.map((b) => ({
					enabled: b.enabled,
					owner_user_id: b.ownerUserId,
					balance: b.balance,
					monthly_contribution: b.monthly
				})),
				annual_return_rate: annualReturnRate,
				limit_growth_rate: limitGrowthRate,
				expected_salary_growth: expectedSalaryGrowth,
				inflation_rate: inflationRate
			};

			const response = await fetch(`${apiUrl}/api/simulations/retirement`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(requestBody)
			});

			if (!response.ok) {
				throw new Error(`Simulation failed: ${response.statusText}`);
			}

			const responseData = await response.json();
			results = responseData;
		} catch (err) {
			console.error('Simulation failed:', err);
			if (err instanceof Error) {
				error = err.message;
			}
		} finally {
			loading = false;
		}
	}

	// Main per-account chart. Driven by $effect so it renders on the tick
	// after `results = ...` flushes the DOM (the {#if results} container
	// isn't bound until then).
	$effect(() => {
		if (!chartContainer) {
			chartHandle?.dispose();
			chartHandle = null;
			chart = null;
			return;
		}
		if (!results) return;

		if (!chartHandle) {
			chartHandle = createChart(chartContainer);
			chart = chartHandle.chart;
		}

		if (results.simulations.length === 0) {
			chart?.clear();
			return;
		}

		chart?.setOption(buildRetirementProjectionOption(results.simulations));
	});

	// Render the wrapper-aggregate chart reactively: it only mounts after the
	// milestone cards reveal, so we can't draw it synchronously when results
	// arrive — wait for the container to bind. When the {#if} unmounts the
	// container (e.g. user re-runs with a sub-60 retirement age), dispose the
	// handle so a later remount creates a fresh chart instead of writing to
	// a detached canvas.
	$effect(() => {
		if (!wrapperChartContainer) {
			wrapperChartHandle?.dispose();
			wrapperChartHandle = null;
			wrapperChart = null;
			return;
		}
		if (!results) return;

		if (!wrapperChartHandle) {
			wrapperChartHandle = createChart(wrapperChartContainer);
			wrapperChart = wrapperChartHandle.chart;
		}

		if (results.simulations.length === 0) {
			wrapperChart?.clear();
			return;
		}

		wrapperChart?.setOption(buildRetirementByWrapperOption(results.simulations, MILESTONE_AGES));
	});

	onMount(() => {
		return () => {
			chartHandle?.dispose();
			chartHandle = null;
			chart = null;
			wrapperChartHandle?.dispose();
			wrapperChartHandle = null;
			wrapperChart = null;
		};
	});

	function formatCurrency(value: number): string {
		return value.toLocaleString('pl-PL', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}
</script>

<div class="space-y-4">
	<h1 class="h1">Symulacje Emerytalne</h1>

	<div class="card preset-filled-surface-100-900 p-4 space-y-3">
		<h2 class="h3">Zapisane scenariusze</h2>
		<div class="flex flex-wrap items-end gap-2">
			<label class="label flex-1 min-w-[200px]">
				<span class="text-sm font-semibold">Nazwa</span>
				<input
					type="text"
					maxlength="200"
					placeholder="np. Plan A — 7% zwrotu"
					bind:value={scenarioName}
					disabled={scenarioBusy}
					class="input"
				/>
			</label>
			<button
				type="button"
				class="btn preset-filled-primary-500"
				onclick={saveCurrentScenario}
				disabled={scenarioBusy || scenarioName.trim() === ''}
			>
				Zapisz bieżący
			</button>
		</div>
		{#if scenarioError}
			<div class="text-sm text-error-600-400">{scenarioError}</div>
		{/if}
		{#if scenarios.length === 0}
			<p class="text-sm text-surface-700-300 italic">
				Brak zapisanych scenariuszy. Skonfiguruj symulację powyżej i kliknij „Zapisz bieżący".
			</p>
		{:else}
			<ul class="divide-y divide-surface-200-800">
				{#each scenarios as s (s.id)}
					<li class="py-2 flex flex-wrap items-center gap-2">
						<span class="flex-1 min-w-[180px] text-sm font-semibold">{s.name}</span>
						<span class="text-xs text-surface-700-300">
							{new Date(s.updated_at).toLocaleString('pl-PL')}
						</span>
						<button
							type="button"
							class="btn btn-sm preset-tonal-primary"
							onclick={() => loadScenarioIntoForm(s)}
							disabled={scenarioBusy}
						>
							Wczytaj
						</button>
						<button
							type="button"
							class="btn btn-sm preset-tonal-surface"
							onclick={() => cloneScenario(s.id)}
							disabled={scenarioBusy}
						>
							Duplikuj
						</button>
						<button
							type="button"
							class="btn btn-sm preset-tonal-error"
							onclick={() => deleteScenario(s.id)}
							disabled={scenarioBusy}
						>
							Usuń
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
		<div class="card preset-filled-surface-100-900 p-5 space-y-4">
			<h2 class="h3">Parametry symulacji</h2>

			<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
				<label class="label">
					<span class="text-sm font-semibold">Obecny wiek</span>
					<input type="number" bind:value={currentAge} min="18" max="100" class="input" />
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Wiek emerytalny</span>
					<input type="number" bind:value={retirementAge} min="18" max="100" class="input" />
				</label>
			</div>

			<h3 class="h4">Konta do symulacji</h3>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
				{#each ikeIkzeAccounts as account, i}
					<div class="card preset-tonal-surface p-4 flex flex-col gap-2">
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="checkbox" bind:checked={ikeIkzeAccounts[i].enabled} class="checkbox" />
							<span class="text-sm font-semibold"
								>{account.wrapper} ({ownerName(owners, account.ownerUserId)})</span
							>
						</label>
						{#if account.enabled}
							<label class="label">
								<span class="text-sm font-semibold">Saldo obecne (PLN)</span>
								<input
									type="number"
									bind:value={ikeIkzeAccounts[i].balance}
									min="0"
									step="100"
									class="input"
								/>
							</label>
							<label class="flex items-center gap-2 cursor-pointer">
								<input
									type="checkbox"
									bind:checked={ikeIkzeAccounts[i].autoFill}
									class="checkbox"
								/>
								<span class="text-sm">Auto-wypełnienie limitu</span>
							</label>
							{#if !account.autoFill}
								<label class="label">
									<span class="text-sm font-semibold">Wpłata miesięczna (PLN)</span>
									<input
										type="number"
										bind:value={ikeIkzeAccounts[i].monthly}
										min="0"
										step="100"
										class="input"
									/>
								</label>
							{/if}
							{#if account.wrapper === 'IKZE'}
								<label class="label">
									<span class="text-sm font-semibold">Stawka podatkowa (%)</span>
									<input
										type="number"
										bind:value={ikeIkzeAccounts[i].taxRate}
										min="0"
										max="50"
										step="1"
										class="input"
									/>
								</label>
							{/if}
						{/if}
					</div>
				{/each}

				{#each ppkAccounts as ppk, i}
					<div class="card preset-tonal-surface p-4 flex flex-col gap-2">
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="checkbox" bind:checked={ppkAccounts[i].enabled} class="checkbox" />
							<span class="text-sm font-semibold">PPK ({ownerName(owners, ppk.ownerUserId)})</span>
						</label>
						{#if ppk.enabled}
							<label class="label">
								<span class="text-sm font-semibold">Obecna wartość (PLN)</span>
								<input
									type="number"
									bind:value={ppkAccounts[i].balance}
									min="0"
									step="1000"
									class="input"
								/>
							</label>
							<label class="label">
								<span class="text-sm font-semibold">Miesięczne wynagrodzenie brutto (PLN)</span>
								<input
									type="number"
									bind:value={ppkAccounts[i].salary}
									min="1000"
									step="500"
									class="input"
								/>
							</label>
							<div class="space-y-2">
								<label class="label">
									<span class="text-sm font-semibold">Składka pracownika (%)</span>
									<input
										type="number"
										bind:value={ppkAccounts[i].employeeRate}
										min="0.5"
										max="4"
										step="0.5"
										class="input"
									/>
									<span class="text-xs text-surface-600-400">Zakres: 0.5-4% (podstawa: 2%)</span>
								</label>
								<label class="label">
									<span class="text-sm font-semibold">Składka pracodawcy (%)</span>
									<input
										type="number"
										bind:value={ppkAccounts[i].employerRate}
										min="1.5"
										max="4"
										step="0.5"
										class="input"
									/>
									<span class="text-xs text-surface-600-400">Zakres: 1.5-4% (podstawa: 1.5%)</span>
								</label>
							</div>
							<label class="flex items-start gap-2 cursor-pointer">
								<input
									type="checkbox"
									bind:checked={ppkAccounts[i].belowThreshold}
									class="checkbox mt-0.5"
								/>
								<span class="text-sm"
									>Wynagrodzenie poniżej progu ({SALARY_THRESHOLD_2026} PLN)
									<span class="block text-xs text-surface-600-400"
										>Dotyczy dopłaty rocznej 240 PLN</span
									>
								</span>
							</label>
							<label class="flex items-center gap-2 cursor-pointer">
								<input
									type="checkbox"
									bind:checked={ppkAccounts[i].includeSubsidies}
									class="checkbox"
								/>
								<span class="text-sm">Uwzględnij dopłaty państwa (250 PLN + 240 PLN/rok)</span>
							</label>
							<p class="text-xs text-surface-600-400">
								Szacowana miesięczna składka:
								{formatCurrency((ppk.salary * (ppk.employeeRate + ppk.employerRate)) / 100)}
								PLN (pracownik: {formatCurrency((ppk.salary * ppk.employeeRate) / 100)} PLN, pracodawca:
								{formatCurrency((ppk.salary * ppk.employerRate) / 100)} PLN)
							</p>
						{/if}
					</div>
				{/each}

				{#each brokerageAccounts as brokerage, i}
					<div class="card preset-tonal-surface p-4 flex flex-col gap-2">
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="checkbox" bind:checked={brokerageAccounts[i].enabled} class="checkbox" />
							<span class="text-sm font-semibold"
								>Rachunek maklerski ({ownerName(owners, brokerage.ownerUserId)})</span
							>
						</label>
						{#if brokerage.enabled}
							<label class="label">
								<span class="text-sm font-semibold">Obecna wartość (PLN)</span>
								<input
									type="number"
									bind:value={brokerageAccounts[i].balance}
									min="0"
									step="1000"
									class="input"
								/>
							</label>
							<label class="label">
								<span class="text-sm font-semibold">Wpłata miesięczna (PLN)</span>
								<input
									type="number"
									bind:value={brokerageAccounts[i].monthly}
									min="0"
									step="100"
									class="input"
								/>
							</label>
							<p class="text-xs text-surface-600-400">
								Rachunki maklerskie są opodatkowane 19% podatkiem Belki od zysków kapitałowych
							</p>
						{/if}
					</div>
				{/each}
			</div>

			<h3 class="h4">Założenia</h3>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
				<label class="label">
					<span class="text-sm font-semibold">Roczna stopa zwrotu (%)</span>
					<input
						type="number"
						bind:value={annualReturnRate}
						min="-50"
						max="50"
						step="0.1"
						class="input"
					/>
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Wzrost limitów wpłat (%)</span>
					<input
						type="number"
						bind:value={limitGrowthRate}
						min="0"
						max="20"
						step="0.1"
						class="input"
					/>
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Przewidywany wzrost wynagrodzeń (%)</span>
					<input
						type="number"
						bind:value={expectedSalaryGrowth}
						min="0"
						max="10"
						step="0.5"
						class="input"
					/>
					<span class="text-xs text-surface-600-400">Roczny wzrost płacy brutto (typowo 3-5%)</span>
				</label>
				<label class="label">
					<span class="text-sm font-semibold">Inflacja (%)</span>
					<input
						type="number"
						bind:value={inflationRate}
						min="0"
						max="20"
						step="0.1"
						class="input"
					/>
					<span class="text-xs text-surface-600-400"
						>Roczna inflacja do przeliczenia dochodu na dzisiejsze pieniądze</span
					>
				</label>
			</div>

			<button
				class="btn preset-filled-primary-500 w-full"
				onclick={runSimulation}
				disabled={loading}
			>
				{loading ? 'Obliczanie...' : 'Uruchom symulację'}
			</button>

			{#if error}
				<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
			{/if}
		</div>

		{#if results}
			<div class="card preset-filled-surface-100-900 p-5 space-y-4">
				<h2 class="h3">Wyniki symulacji</h2>

				<div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Końcowy kapitał</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.total_final_balance)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Miesięczny dochód (4% rule)</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.estimated_monthly_income)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">
							Miesięczny dochód (w dzisiejszych pieniądzach)
						</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.estimated_monthly_income_today)} PLN
						</div>
						<div class="text-xs text-surface-600-400 mt-1">przy 3% inflacji rocznie</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Suma wpłat</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.total_contributions)} PLN
						</div>
					</div>
					<div class="card preset-tonal-surface p-4">
						<div class="text-xs text-surface-600-400 mb-2">Zyski z inwestycji</div>
						<div class="text-xl font-bold">
							{formatCurrency(results.summary.total_returns)} PLN
						</div>
					</div>
					{#if results.summary.total_tax_savings > 0}
						<div class="card preset-tonal-surface p-4">
							<div class="text-xs text-surface-600-400 mb-2">Oszczędności podatkowe (IKZE)</div>
							<div class="text-xl font-bold">
								{formatCurrency(results.summary.total_tax_savings)} PLN
							</div>
						</div>
					{/if}
					{#if results.summary.total_subsidies && results.summary.total_subsidies > 0}
						<div class="card preset-tonal-surface p-4">
							<div class="text-xs text-surface-600-400 mb-2">Dopłaty państwa (PPK)</div>
							<div class="text-xl font-bold">
								{formatCurrency(results.summary.total_subsidies)} PLN
							</div>
						</div>
					{/if}
				</div>

				<div bind:this={chartContainer} class="w-full h-[280px] sm:h-[400px]"></div>

				{#if milestoneBalances.some((m) => m.balance !== null)}
					<h3 class="h4">Saldo IKE + IKZE + PPK w wieku emerytalnym</h3>
					<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
						{#each milestoneBalances as milestone (milestone.age)}
							<div class="card preset-tonal-surface p-4">
								<div class="text-xs text-surface-600-400 mb-2">Wiek {milestone.age}</div>
								<div class="text-xl font-bold">
									{#if milestone.balance === null}
										—
									{:else}
										{formatCurrency(milestone.balance)} PLN
									{/if}
								</div>
							</div>
						{/each}
					</div>
					<div bind:this={wrapperChartContainer} class="w-full h-[280px] sm:h-[400px]"></div>
				{/if}

				<h3 class="h4">Szczegóły projekcji</h3>
				{#each results.simulations as simulation}
					<details class="card preset-tonal-surface p-3 mb-3">
						<summary class="cursor-pointer text-sm font-semibold">
							<strong>{simulation.account_name}</strong> - Końcowa wartość: {formatCurrency(
								simulation.final_balance
							)} PLN
						</summary>
						<div class="table-wrap mt-3">
							<table class="table table-hover text-xs">
								<thead>
									<tr>
										<th>Rok</th>
										<th class="text-right">Wiek</th>
										<th class="text-right">Roczna wpłata</th>
										{#if !simulation.account_name.startsWith('PPK')}
											<th class="text-right">Wykorzystanie limitu</th>
										{/if}
										<th class="text-right">Saldo</th>
										<th class="text-right">Suma wpłat</th>
										<th class="text-right">Zyski</th>
										{#if simulation.account_name.includes('IKZE')}
											<th class="text-right">Ulga podatkowa</th>
										{/if}
										{#if simulation.account_name.startsWith('PPK')}
											<th class="text-right">Dopłaty państwa</th>
											<th class="text-right">Roczne wynagrodzenie</th>
											<th class="text-right">Stopa zwrotu</th>
										{/if}
									</tr>
								</thead>
								<tbody>
									{#each simulation.yearly_projections as projection}
										<tr>
											<td>{projection.year}</td>
											<td class="text-right">{projection.age}</td>
											<td class="text-right">{formatCurrency(projection.annual_contribution)}</td>
											{#if !simulation.account_name.startsWith('PPK')}
												<td class="text-right">{projection.limit_utilized_pct.toFixed(1)}%</td>
											{/if}
											<td class="text-right">{formatCurrency(projection.balance_end_of_year)}</td>
											<td class="text-right"
												>{formatCurrency(projection.cumulative_contributions)}</td
											>
											<td class="text-right">{formatCurrency(projection.cumulative_returns)}</td>
											{#if simulation.account_name.includes('IKZE')}
												<td class="text-right">{formatCurrency(projection.tax_savings)}</td>
											{/if}
											{#if simulation.account_name.startsWith('PPK')}
												<td class="text-right"
													>{formatCurrency(projection.government_subsidies || 0)}</td
												>
												<td class="text-right">
													{formatCurrency((projection.monthly_salary || 0) * 12)}
												</td>
												<td class="text-right">{(projection.return_rate || 0).toFixed(1)}%</td>
											{/if}
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</details>
				{/each}
			</div>
		{/if}
	</div>
</div>
