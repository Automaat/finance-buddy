<script lang="ts">
	import { onMount } from 'svelte';
	import { resolveApiUrl } from '$lib/api';
	import { formatDate } from '$lib/utils/format';
	import { GitCompareArrows, ArrowLeft } from 'lucide-svelte';

	interface RetirementScenarioInputs {
		current_age: number;
		retirement_age: number;
		ike_ikze_accounts: unknown[];
		ppk_accounts: unknown[];
		brokerage_accounts: unknown[];
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

	interface SimulationSummary {
		total_final_balance: number;
		total_contributions: number;
		total_returns: number;
		total_tax_savings: number;
		estimated_monthly_income: number;
		estimated_monthly_income_today: number;
		years_until_retirement: number;
	}

	interface ComparisonRow {
		scenario: SavedScenario;
		summary: SimulationSummary;
		fiDate: Date;
		successRate: number; // 0..∞ ratio of final balance to required capital
	}

	const apiUrl = resolveApiUrl();

	let scenarios: SavedScenario[] = $state([]);
	let selected: Set<number> = $state(new Set());
	let loading = $state(false);
	let error = $state('');
	let results: ComparisonRow[] = $state([]);
	let requiredCapital = $state(0);

	const selectedCount = $derived(selected.size);
	const canCompare = $derived(selectedCount >= 2 && !loading);

	async function loadScenarios() {
		try {
			const r = await fetch(`${apiUrl}/api/scenarios?kind=retirement`);
			if (!r.ok) throw new Error(`HTTP ${r.status}`);
			const body = await r.json();
			scenarios = (body.scenarios ?? []) as SavedScenario[];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Nie udało się pobrać scenariuszy';
		}
	}

	async function loadRequiredCapital() {
		try {
			const r = await fetch(`${apiUrl}/api/config`);
			if (!r.ok) return;
			const cfg = await r.json();
			const monthly = Number(cfg.monthly_expenses ?? 0);
			const withdrawalRate = Number(cfg.withdrawal_rate ?? 0.04);
			requiredCapital = withdrawalRate > 0 ? (monthly * 12) / withdrawalRate : 0;
		} catch {
			requiredCapital = 0;
		}
	}

	function toggle(id: number) {
		// Reassign so Svelte 5 sees the Set change.
		const next = new Set(selected);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		selected = next;
	}

	function inputsToRequestBody(inputs: RetirementScenarioInputs): unknown {
		// The /simulations/retirement endpoint expects the same shape the main
		// simulations page POSTs. The saved scenario stores that shape verbatim
		// (camelCase converted to snake_case in saveCurrentScenario), so the
		// payload is largely a direct echo. ike_ikze_accounts arrays already
		// carry the form's per-row fields — re-key the ones that differ from
		// the API contract.
		const reMap = (rows: unknown[], fn: (a: Record<string, unknown>) => unknown) =>
			(rows ?? []).map((r) => fn(r as Record<string, unknown>));
		return {
			current_age: inputs.current_age,
			retirement_age: inputs.retirement_age,
			ike_ikze_accounts: reMap(inputs.ike_ikze_accounts, (a) => ({
				enabled: a.enabled,
				wrapper: a.wrapper,
				owner_user_id: a.ownerUserId ?? a.owner_user_id,
				balance: a.balance,
				auto_fill_limit: a.autoFill ?? a.auto_fill_limit,
				monthly_contribution: a.monthly ?? a.monthly_contribution,
				tax_rate: a.taxRate ?? a.tax_rate
			})),
			// Mirror the main simulations page: only enabled PPK rows are sent so
			// disabled-but-saved configs don't distort the comparison.
			ppk_accounts: reMap(
				(inputs.ppk_accounts ?? []).filter((a) => (a as Record<string, unknown>).enabled !== false),
				(a) => ({
					owner_user_id: a.ownerUserId ?? a.owner_user_id,
					enabled: true,
					starting_balance: a.balance ?? a.starting_balance,
					monthly_gross_salary: a.salary ?? a.monthly_gross_salary,
					employee_rate: a.employeeRate ?? a.employee_rate,
					employer_rate: a.employerRate ?? a.employer_rate,
					salary_below_threshold: a.belowThreshold ?? a.salary_below_threshold,
					include_welcome_bonus: a.includeSubsidies ?? a.include_welcome_bonus,
					include_annual_subsidy: a.includeSubsidies ?? a.include_annual_subsidy
				})
			),
			brokerage_accounts: reMap(inputs.brokerage_accounts, (a) => ({
				enabled: a.enabled,
				owner_user_id: a.ownerUserId ?? a.owner_user_id,
				balance: a.balance,
				monthly_contribution: a.monthly ?? a.monthly_contribution
			})),
			annual_return_rate: inputs.annual_return_rate,
			limit_growth_rate: inputs.limit_growth_rate,
			expected_salary_growth: inputs.expected_salary_growth,
			inflation_rate: inputs.inflation_rate
		};
	}

	async function compareSelected() {
		if (!canCompare) return;
		loading = true;
		error = '';
		results = [];
		try {
			const chosen = scenarios.filter((s) => selected.has(s.id));
			// allSettled so one bad scenario doesn't blank the whole comparison —
			// the failed names are surfaced via `error` while the OK rows still
			// render the table.
			const settled = await Promise.allSettled(
				chosen.map(async (s) => {
					const r = await fetch(`${apiUrl}/api/simulations/retirement`, {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify(inputsToRequestBody(s.inputs_json))
					});
					if (!r.ok) throw new Error(`HTTP ${r.status}`);
					const body = await r.json();
					return { scenario: s, summary: body.summary as SimulationSummary };
				})
			);
			const errs: string[] = [];
			const ok = settled.flatMap((res, i) => {
				if (res.status === 'fulfilled') return [res.value];
				const reason = res.reason instanceof Error ? res.reason.message : String(res.reason);
				errs.push(`${chosen[i].name}: ${reason}`);
				return [];
			});
			if (errs.length) error = errs.join('; ');
			const today = new Date();
			results = ok.map(({ scenario, summary }) => {
				const fiDate = new Date(
					today.getFullYear() + summary.years_until_retirement,
					today.getMonth(),
					1
				);
				const successRate = requiredCapital > 0 ? summary.total_final_balance / requiredCapital : 0;
				return { scenario, summary, fiDate, successRate };
			});
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	}

	function fmtPLN(v: number): string {
		return v.toLocaleString('pl-PL', { maximumFractionDigits: 0 }) + ' PLN';
	}

	function fmtPct(v: number): string {
		return (v * 100).toFixed(1) + '%';
	}

	// Backend returns naive UTC strings; new Date() would re-anchor those to
	// the browser's local zone. Append Z when missing so the rendered moment
	// matches the server clock. Guarded so a future migration to suffixed
	// strings doesn't double-up.
	function fmtUpdatedAt(s: string): string {
		const utc = s.endsWith('Z') ? s : s + 'Z';
		return formatDate(utc, { timeZone: 'UTC' });
	}

	// Per-row min/max for color-coding so differences are easy to scan.
	// "Higher is better" rows get green=max, orange=min; "lower is better"
	// rows (years_until_retirement, fiDate) invert.
	type Direction = 'higher' | 'lower';

	function rowExtreme(values: number[], direction: Direction): { min: number; max: number } {
		const min = Math.min(...values);
		const max = Math.max(...values);
		return direction === 'higher' ? { min, max } : { min: max, max: min };
	}

	function cellClass(value: number, extremes: { min: number; max: number }): string {
		if (results.length < 2) return '';
		if (value === extremes.max && extremes.max !== extremes.min)
			return 'text-success-600-400 font-semibold';
		if (value === extremes.min && extremes.max !== extremes.min)
			return 'text-warning-600-400 font-semibold';
		return '';
	}

	const yearsExtremes = $derived(
		results.length
			? rowExtreme(
					results.map((r) => r.summary.years_until_retirement),
					'lower'
				)
			: { min: 0, max: 0 }
	);
	const balanceExtremes = $derived(
		results.length
			? rowExtreme(
					results.map((r) => r.summary.total_final_balance),
					'higher'
				)
			: { min: 0, max: 0 }
	);
	const incomeExtremes = $derived(
		results.length
			? rowExtreme(
					results.map((r) => r.summary.estimated_monthly_income_today),
					'higher'
				)
			: { min: 0, max: 0 }
	);
	const successExtremes = $derived(
		results.length
			? rowExtreme(
					results.map((r) => r.successRate),
					'higher'
				)
			: { min: 0, max: 0 }
	);
	const contributionsExtremes = $derived(
		results.length
			? rowExtreme(
					results.map((r) => r.summary.total_contributions),
					'higher'
				)
			: { min: 0, max: 0 }
	);
	const returnsExtremes = $derived(
		results.length
			? rowExtreme(
					results.map((r) => r.summary.total_returns),
					'higher'
				)
			: { min: 0, max: 0 }
	);
	const nominalIncomeExtremes = $derived(
		results.length
			? rowExtreme(
					results.map((r) => r.summary.estimated_monthly_income),
					'higher'
				)
			: { min: 0, max: 0 }
	);

	onMount(() => {
		loadScenarios();
		loadRequiredCapital();
	});
</script>

<svelte:head>
	<title>Porównanie scenariuszy | Symulacje</title>
</svelte:head>

<div class="space-y-4">
	<div class="flex items-center gap-2">
		<a href="/simulations" class="btn btn-sm preset-tonal-surface">
			<ArrowLeft size={16} />
			Wróć
		</a>
		<h1 class="h1 flex items-center gap-2">
			<GitCompareArrows size={28} /> Porównanie scenariuszy
		</h1>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-3">
		<h2 class="h3">Wybierz scenariusze do porównania</h2>
		{#if scenarios.length === 0}
			<p class="text-sm text-surface-700-300 italic">
				Brak zapisanych scenariuszy. Wróć do
				<a class="underline" href="/simulations">Symulacji</a> i zapisz przynajmniej dwa.
			</p>
		{:else}
			<ul class="space-y-1">
				{#each scenarios as s (s.id)}
					<li class="flex items-center gap-2">
						<input
							id={`sc-${s.id}`}
							type="checkbox"
							class="checkbox"
							checked={selected.has(s.id)}
							onchange={() => toggle(s.id)}
						/>
						<label for={`sc-${s.id}`} class="text-sm cursor-pointer flex-1">
							{s.name}
						</label>
						<span class="text-xs text-surface-700-300">
							{fmtUpdatedAt(s.updated_at)}
						</span>
					</li>
				{/each}
			</ul>
			<div class="flex items-center gap-2 flex-wrap">
				<button
					type="button"
					class="btn preset-filled-primary-500"
					onclick={compareSelected}
					disabled={!canCompare}
				>
					{loading ? 'Liczenie...' : `Porównaj zaznaczone (${selectedCount})`}
				</button>
				{#if selectedCount === 1}
					<span class="text-xs text-surface-700-300">Wybierz co najmniej dwa scenariusze</span>
				{/if}
			</div>
		{/if}
		{#if error}
			<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
		{/if}
	</div>

	{#if results.length >= 2}
		<div class="card preset-filled-surface-100-900 p-4 space-y-3 overflow-x-auto">
			<h2 class="h3">Wyniki porównania</h2>
			<table class="table w-full text-sm">
				<thead>
					<tr>
						<th class="text-left">Metryka</th>
						{#each results as r (r.scenario.id)}
							<th class="text-right">{r.scenario.name}</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					<tr class="font-semibold preset-tonal-surface">
						<td colspan={results.length + 1}>Założenia</td>
					</tr>
					<tr>
						<td>Obecny wiek</td>
						{#each results as r (r.scenario.id)}
							<td class="text-right">{r.scenario.inputs_json.current_age ?? '—'}</td>
						{/each}
					</tr>
					<tr>
						<td>Wiek emerytalny</td>
						{#each results as r (r.scenario.id)}
							<td class="text-right">{r.scenario.inputs_json.retirement_age ?? '—'}</td>
						{/each}
					</tr>
					<tr>
						<td>Roczna stopa zwrotu</td>
						{#each results as r (r.scenario.id)}
							<td class="text-right">
								{(r.scenario.inputs_json.annual_return_rate ?? 0).toFixed(1)}%
							</td>
						{/each}
					</tr>
					<tr>
						<td>Inflacja</td>
						{#each results as r (r.scenario.id)}
							<td class="text-right">
								{(r.scenario.inputs_json.inflation_rate ?? 0).toFixed(1)}%
							</td>
						{/each}
					</tr>
					<tr class="font-semibold preset-tonal-surface">
						<td colspan={results.length + 1}>Wyniki</td>
					</tr>
					<tr>
						<td>Lata do emerytury</td>
						{#each results as r (r.scenario.id)}
							<td class="text-right {cellClass(r.summary.years_until_retirement, yearsExtremes)}">
								{r.summary.years_until_retirement}
							</td>
						{/each}
					</tr>
					<tr>
						<td>Data FI</td>
						{#each results as r (r.scenario.id)}
							<td class="text-right">{formatDate(r.fiDate)}</td>
						{/each}
					</tr>
					{#if requiredCapital > 0}
						<tr>
							<td title="annual_expenses ÷ withdrawal_rate (z konfiguracji)">Wymagany kapitał</td>
							{#each results as r (r.scenario.id)}
								<td class="text-right">{fmtPLN(requiredCapital)}</td>
							{/each}
						</tr>
					{/if}
					<tr>
						<td>Wartość końcowa portfela</td>
						{#each results as r (r.scenario.id)}
							<td class="text-right {cellClass(r.summary.total_final_balance, balanceExtremes)}">
								{fmtPLN(r.summary.total_final_balance)}
							</td>
						{/each}
					</tr>
					{#if requiredCapital > 0}
						<tr>
							<td title="Wartość końcowa portfela ÷ wymagany kapitał × 100%">
								Pokrycie celu (success)
							</td>
							{#each results as r (r.scenario.id)}
								<td class="text-right {cellClass(r.successRate, successExtremes)}">
									{fmtPct(r.successRate)}
								</td>
							{/each}
						</tr>
					{/if}
					<tr>
						<td>Wpłaty łącznie</td>
						{#each results as r (r.scenario.id)}
							<td
								class="text-right {cellClass(r.summary.total_contributions, contributionsExtremes)}"
							>
								{fmtPLN(r.summary.total_contributions)}
							</td>
						{/each}
					</tr>
					<tr>
						<td>Zwroty łącznie</td>
						{#each results as r (r.scenario.id)}
							<td class="text-right {cellClass(r.summary.total_returns, returnsExtremes)}">
								{fmtPLN(r.summary.total_returns)}
							</td>
						{/each}
					</tr>
					<tr>
						<td>Miesięczny dochód (nominalny)</td>
						{#each results as r (r.scenario.id)}
							<td
								class="text-right {cellClass(
									r.summary.estimated_monthly_income,
									nominalIncomeExtremes
								)}"
							>
								{fmtPLN(r.summary.estimated_monthly_income)}
							</td>
						{/each}
					</tr>
					<tr>
						<td>Miesięczny dochód (dzisiejszej wartości)</td>
						{#each results as r (r.scenario.id)}
							<td
								class="text-right {cellClass(
									r.summary.estimated_monthly_income_today,
									incomeExtremes
								)}"
							>
								{fmtPLN(r.summary.estimated_monthly_income_today)}
							</td>
						{/each}
					</tr>
				</tbody>
			</table>
			<p class="text-xs text-surface-700-300 italic">
				Zielony = najlepszy wynik w danej kategorii; pomarańczowy = najsłabszy. „Pokrycie celu"
				liczone względem FIRE number z konfiguracji.
			</p>
		</div>
	{/if}
</div>
