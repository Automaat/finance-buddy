<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import Modal from '$lib/components/Modal.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import DashboardCharts from '$lib/components/DashboardCharts.svelte';
	import AllocationDriftWidget from '$lib/components/AllocationDriftWidget.svelte';
	import DateRangePicker from '$lib/components/DateRangePicker.svelte';
	import DeltaBadge from '$lib/components/DeltaBadge.svelte';
	import { formatPLN } from '$lib/utils/format';
	import {
		Wallet,
		TrendingUp,
		TrendingDown,
		Settings,
		CheckCircle2,
		AlertTriangle,
		PiggyBank,
		Coins,
		Flame
	} from 'lucide-svelte';
	import { resolveApiUrl } from '$lib/api';
	import { invalidateAll } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import {
		countdownTier,
		daysLabel,
		daysUntilYearEnd,
		type CountdownTier
	} from '$lib/utils/yearEnd';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	let owners: OwnerOption[] = $state([]);

	$effect(() => {
		let cancelled = false;
		Promise.resolve(data.owners).then((o) => {
			if (!cancelled) owners = (o ?? []) as OwnerOption[];
		});
		return () => {
			cancelled = true;
		};
	});

	let showLimitsModal = $state(false);
	let limitsYear = $state(untrack(() => data.currentYear));
	let limits: Record<string, number> = $state({});

	type RetirementStat = {
		account_wrapper: string;
		owner_user_id: number | null;
		total_contributed: number;
		limit_amount: number;
		remaining: number;
		percentage_used: number;
	};

	function openLimitsModal(retirementStats: RetirementStat[]) {
		limits = {};
		for (const owner of owners) {
			for (const wrapper of ['IKE', 'IKZE']) {
				limits[`${wrapper}_${owner.id}`] = 0;
			}
		}
		for (const stat of retirementStats) {
			const key = `${stat.account_wrapper}_${stat.owner_user_id}`;
			if (key in limits) {
				limits[key] = stat.limit_amount || 0;
			}
		}
		showLimitsModal = true;
	}

	async function saveLimits() {
		const apiUrl = resolveApiUrl();
		try {
			const requests = Object.entries(limits).map(([key, amount]) => {
				const sep = key.indexOf('_');
				const wrapper = key.slice(0, sep);
				const ownerUserId = Number(key.slice(sep + 1));
				return fetch(`${apiUrl}/api/retirement/limits/${limitsYear}/${wrapper}/${ownerUserId}`, {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						year: limitsYear,
						account_wrapper: wrapper,
						owner_user_id: ownerUserId,
						limit_amount: amount,
						notes: ''
					})
				});
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
			toast.error('Nie udało się zapisać limitów. Spróbuj ponownie później.');
		}
	}

	// Polish plural form of "rok" (year). 1 → rok; 2–4 (except 12–14) → lata;
	// everything else → lat. Inline because it's the only spot that needs it.
	function plYears(n: number): string {
		if (n === 1) return 'rok';
		const last = n % 10;
		const lastTwo = n % 100;
		if (last >= 2 && last <= 4 && (lastTwo < 12 || lastTwo > 14)) return 'lata';
		return 'lat';
	}

	// Date-driven UI. SSR seeds with a server-side `now`; onMount resets to the
	// client clock to avoid timezone drift, and a daily tick keeps the counter
	// honest across midnight in long-lived sessions.
	let now = $state(new Date());
	const daysLeft = $derived(daysUntilYearEnd(now));

	onMount(() => {
		now = new Date();
		const id = setInterval(
			() => {
				now = new Date();
			},
			60 * 60 * 1000
		);
		return () => clearInterval(id);
	});

	const tierBar: Record<CountdownTier, string> = {
		maxed: 'bg-success-500',
		safe: 'bg-success-500',
		warn: 'bg-warning-500',
		urgent: 'bg-error-500'
	};

	const tierText: Record<CountdownTier, string> = {
		maxed: 'text-success-600-400',
		safe: 'text-success-600-400',
		warn: 'text-warning-600-400',
		urgent: 'text-error-600-400'
	};
</script>

<svelte:head>
	<title>Dashboard | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-8">
	<div class="space-y-1">
		<h1 class="h2">Dashboard</h1>
		<p class="text-surface-700-300 text-sm">Twoja sytuacja finansowa w jednym miejscu</p>
	</div>

	<DateRangePicker />

	{#await data.dashboardData}
		<div role="status" aria-live="polite" aria-label="Ładowanie dashboardu" class="space-y-8">
			<div class="grid gap-4 sm:gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
				{#each { length: 3 } as _, i (i)}
					<div class="card preset-filled-surface-100-900 p-4 space-y-3">
						<Skeleton height="1.25rem" width="60%" />
						<Skeleton height="2rem" width="80%" />
						<Skeleton height="0.875rem" width="70%" />
					</div>
				{/each}
			</div>

			<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
				<div class="card preset-filled-surface-100-900 p-4">
					<Skeleton height="400px" rounded="lg" />
				</div>
				<div class="card preset-filled-surface-100-900 p-4">
					<Skeleton height="400px" rounded="lg" />
				</div>
			</div>
		</div>
	{:then dashboard}
		<div class="grid gap-4 sm:gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
			<div class="card preset-filled-surface-100-900 p-4 space-y-2">
				<header>
					<h3 class="h4 flex items-center gap-2"><Wallet size={18} /> Wartość Netto</h3>
				</header>
				<div class="text-3xl font-bold">{formatPLN(dashboard.current_net_worth)}</div>
				<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1">
					<DeltaBadge
						label="MoM"
						absolute={dashboard.tile_deltas?.net_worth?.mom?.absolute ?? null}
						percentage={dashboard.tile_deltas?.net_worth?.mom?.percentage ?? null}
						formulaTitle="Δ MoM = bieżąca − sprzed ~1 miesiąca; % = Δ / |sprzed ~1 miesiąca|"
					/>
					<DeltaBadge
						label="YoY"
						absolute={dashboard.tile_deltas?.net_worth?.yoy?.absolute ?? null}
						percentage={dashboard.tile_deltas?.net_worth?.yoy?.percentage ?? null}
						formulaTitle="Δ YoY = bieżąca − sprzed ~12 miesięcy; % = Δ / |sprzed ~12 miesięcy|"
					/>
				</div>
			</div>

			<div class="card preset-filled-surface-100-900 p-4 space-y-2">
				<header>
					<h3 class="h4 flex items-center gap-2"><TrendingUp size={18} /> Aktywa</h3>
				</header>
				<div class="text-3xl font-bold text-success-600-400">
					{formatPLN(dashboard.total_assets)}
				</div>
				<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1">
					<DeltaBadge
						label="MoM"
						absolute={dashboard.tile_deltas?.assets?.mom?.absolute ?? null}
						percentage={dashboard.tile_deltas?.assets?.mom?.percentage ?? null}
						formulaTitle="Δ MoM = bieżąca − sprzed ~1 miesiąca; % = Δ / |sprzed ~1 miesiąca|"
					/>
					<DeltaBadge
						label="YoY"
						absolute={dashboard.tile_deltas?.assets?.yoy?.absolute ?? null}
						percentage={dashboard.tile_deltas?.assets?.yoy?.percentage ?? null}
						formulaTitle="Δ YoY = bieżąca − sprzed ~12 miesięcy; % = Δ / |sprzed ~12 miesięcy|"
					/>
				</div>
			</div>

			<div class="card preset-filled-surface-100-900 p-4 space-y-2">
				<header>
					<h3 class="h4 flex items-center gap-2"><TrendingDown size={18} /> Zobowiązania</h3>
				</header>
				<div class="text-3xl font-bold text-error-600-400">
					{formatPLN(dashboard.total_liabilities)}
				</div>
				<div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1">
					<DeltaBadge
						label="MoM"
						absolute={dashboard.tile_deltas?.liabilities?.mom?.absolute ?? null}
						percentage={dashboard.tile_deltas?.liabilities?.mom?.percentage ?? null}
						formulaTitle="Δ MoM = bieżąca − sprzed ~1 miesiąca; % = Δ / |sprzed ~1 miesiąca| — niższe zobowiązania (znak ujemny) = poprawa"
					/>
					<DeltaBadge
						label="YoY"
						absolute={dashboard.tile_deltas?.liabilities?.yoy?.absolute ?? null}
						percentage={dashboard.tile_deltas?.liabilities?.yoy?.percentage ?? null}
						formulaTitle="Δ YoY = bieżąca − sprzed ~12 miesięcy; % = Δ / |sprzed ~12 miesięcy| — niższe zobowiązania (znak ujemny) = poprawa"
					/>
				</div>
			</div>
		</div>

		{@const _fire = dashboard.metric_cards}
		{#if _fire?.fire_number != null || _fire?.runway_months != null || _fire?.lean_fire_number != null || _fire?.fat_fire_number != null}
			{@const fire = _fire!}
			{@const hasBase = fire.fire_number != null && fire.runway_months != null}
			{@const progress = fire.fi_progress}
			{@const annualExpensesPLN = formatPLN(fire.annual_expenses ?? 0)}
			{@const firePLN = formatPLN(fire.fire_number ?? 0)}
			{@const wrPct = ((fire.withdrawal_rate ?? 0.04) * 100).toFixed(1)}
			{@const runwayLabel = (fire.runway_months ?? 0).toFixed(1)}
			{@const coastNum = fire.coast_fire_number}
			{@const coastGap = fire.coast_fire_gap}
			{@const coastTargetAge = fire.coast_fire_target_age}
			{@const coastReturnPct = ((fire.expected_return_rate ?? 0.07) * 100).toFixed(1)}
			{@const baristaNum = fire.barista_fire_number}
			{@const baristaProgress = fire.barista_fi_progress}
			{@const baristaIncome = fire.barista_monthly_income}
			{@const baristaYears = fire.barista_years_to_fi}
			{@const leanFire = fire.lean_fire_number}
			{@const leanProgress = fire.lean_fi_progress}
			{@const fatFire = fire.fat_fire_number}
			{@const fatProgress = fire.fat_fi_progress}
			{@const fiYears = fire.fi_years_remaining}
			{@const fiDate = fire.fi_projected_date}
			{@const monthlySavings = fire.monthly_savings}
			{@const bridgeYears = fire.bridge_years}
			{@const bridgeNeeded = fire.bridge_capital_needed}
			{@const bridgeLiquid = fire.bridge_liquid_capital}
			{@const bridgeGap = fire.bridge_capital_gap}
			<div class="card preset-filled-surface-100-900 p-4 space-y-3">
				<header class="flex items-start justify-between gap-2 flex-wrap">
					<h3 class="h4 flex items-center gap-2"><Flame size={18} /> FIRE i runway</h3>
					{#if hasBase}
						<span
							class="text-xs text-surface-700-300"
							title="annual_expenses = miesięczne wydatki × 12&#10;FIRE = annual_expenses ÷ withdrawal_rate (np. ÷ 0.04 = ×25)&#10;FI = fire_net_worth ÷ FIRE × 100%&#10;fire_net_worth = wartość netto − aktywa oznaczone „poza FIRE”&#10;runway = aktywa płynne (bank + saving_account) ÷ miesięczne wydatki"
						>
							SWR {wrPct}% · ø {annualExpensesPLN}/rok
						</span>
					{/if}
				</header>
				{#if fire.fire_excluded_value != null && fire.fire_net_worth != null}
					<div class="text-xs text-surface-700-300">
						FIRE liczone z {formatPLN(fire.fire_net_worth)} (wartość netto − {formatPLN(
							fire.fire_excluded_value
						)} aktywów oznaczonych „poza FIRE”).
					</div>
				{/if}
				{#if hasBase}
					<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
						<div class="space-y-1">
							<div class="text-xs text-surface-700-300">FI progress</div>
							<div class="text-2xl font-bold">
								{progress != null ? `${progress.toFixed(1)}%` : '—'}
							</div>
							<div class="h-2 rounded-full bg-surface-200-800 overflow-hidden">
								<div
									class="h-full transition-all {(progress ?? 0) >= 100
										? 'bg-success-500'
										: (progress ?? 0) >= 50
											? 'bg-warning-500'
											: 'bg-primary-500'}"
									style="width: {Math.min(progress ?? 0, 100)}%"
								></div>
							</div>
							<div class="text-xs text-surface-700-300">cel: {firePLN}</div>
						</div>
						<div class="space-y-1">
							<div class="text-xs text-surface-700-300">Runway</div>
							<div class="text-2xl font-bold">{runwayLabel} mies.</div>
							<div class="text-xs text-surface-700-300">
								ile miesięcy wydatków pokrywają aktywa płynne
							</div>
						</div>
					</div>
				{/if}
				{#if coastNum != null && coastGap != null && coastTargetAge != null}
					{@const surplus = coastGap <= 0}
					<div class="pt-3 border-t border-surface-200-800 grid grid-cols-1 sm:grid-cols-2 gap-3">
						<div class="space-y-1">
							<div
								class="text-xs text-surface-700-300"
								title="coast_fire_number = FIRE ÷ (1 + expected_return)^(target_age − current_age) — kapitał potrzebny dziś, aby bez dalszych wpłat osiągnąć FIRE w docelowym wieku."
							>
								Coast FIRE (cel {coastTargetAge} r., {coastReturnPct}%/rok)
							</div>
							<div class="text-2xl font-bold">{formatPLN(coastNum)}</div>
							<div class="text-xs text-surface-700-300">kapitał dziś, by przestać inwestować</div>
						</div>
						<div class="space-y-1">
							<div class="text-xs text-surface-700-300">
								{surplus ? 'Nadwyżka' : 'Brakuje'}
							</div>
							<div
								class="text-2xl font-bold {surplus
									? 'text-success-600-400'
									: 'text-warning-600-400'}"
							>
								{formatPLN(Math.abs(coastGap))}
							</div>
							<div class="text-xs text-surface-700-300">
								{surplus ? 'już osiągnięto Coast FIRE' : 'do osiągnięcia Coast FIRE'}
							</div>
						</div>
					</div>
				{/if}
				{#if leanFire != null || fatFire != null}
					<div class="{hasBase ? 'pt-3 border-t border-surface-200-800' : ''} space-y-2">
						<div
							class="text-xs text-surface-700-300"
							title="Każde pasmo = roczne wydatki danego poziomu ÷ withdrawal_rate. Bazowe = Twoje zwykłe miesięczne wydatki."
						>
							Pasma FIRE
						</div>
						<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
							{#if leanFire != null}
								<div class="space-y-1">
									<div class="text-xs text-surface-700-300">Lean FIRE</div>
									<div class="text-xl font-bold">{formatPLN(leanFire)}</div>
									<div class="text-xs text-surface-700-300">
										{leanProgress != null ? `${leanProgress.toFixed(1)}% celu` : 'brak danych'}
									</div>
								</div>
							{/if}
							{#if hasBase}
								<div class="space-y-1">
									<div class="text-xs text-surface-700-300">Base FIRE</div>
									<div class="text-xl font-bold">{firePLN}</div>
									<div class="text-xs text-surface-700-300">
										{progress != null ? `${progress.toFixed(1)}% celu` : 'brak danych'}
									</div>
								</div>
							{/if}
							{#if fatFire != null}
								<div class="space-y-1">
									<div class="text-xs text-surface-700-300">Fat FIRE</div>
									<div class="text-xl font-bold">{formatPLN(fatFire)}</div>
									<div class="text-xs text-surface-700-300">
										{fatProgress != null ? `${fatProgress.toFixed(1)}% celu` : 'brak danych'}
									</div>
								</div>
							{/if}
						</div>
					</div>
				{/if}
				{#if fiYears != null && fiDate != null && monthlySavings != null}
					{@const [year, month] = fiDate.split('-')}
					<div class="pt-3 border-t border-surface-200-800 grid grid-cols-1 sm:grid-cols-2 gap-3">
						<div class="space-y-1">
							<div
								class="text-xs text-surface-700-300"
								title="t = ln((FIRE×r + S) ÷ (NW×r + S)) ÷ ln(1+r)&#10;NW = wartość netto · S = roczne oszczędności · r = expected_return_rate · FIRE = roczne wydatki ÷ withdrawal_rate"
							>
								Prognozowana data FI
							</div>
							<div class="text-2xl font-bold">{month}/{year}</div>
							<div class="text-xs text-surface-700-300">
								~{fiYears.toFixed(1)} lat · oszczędności {formatPLN(monthlySavings)}/mies.
							</div>
						</div>
						<div class="space-y-1">
							<div class="text-xs text-surface-700-300">Lata do FI</div>
							<div class="text-2xl font-bold">{fiYears.toFixed(1)}</div>
							<div class="text-xs text-surface-700-300">
								przy {coastReturnPct}%/rok zwrotu i bieżących wpłatach
							</div>
						</div>
					</div>
				{:else if monthlySavings == null && hasBase}
					<div class="pt-3 border-t border-surface-200-800 text-sm text-surface-700-300">
						<div class="font-semibold mb-1">Prognozowana data FI</div>
						<div class="italic">
							Wprowadź miesięczne oszczędności w
							<a href="/settings/config" class="underline">konfiguracji</a>, aby zobaczyć datę FI.
						</div>
					</div>
				{/if}
				{#if baristaNum != null && baristaIncome != null}
					<div class="pt-3 border-t border-surface-200-800 space-y-2">
						<div class="text-xs text-surface-700-300">
							Praca dorywcza: {formatPLN(baristaIncome)}/mies.
						</div>
						<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
							<div class="space-y-1">
								<div class="text-xs text-surface-700-300">FIRE — porównanie</div>
								<div class="text-sm">
									Klasyczne: <span class="font-bold">{firePLN}</span>
								</div>
								<div
									class="text-sm"
									title="barista_fire_number = max(0, annual_expenses − barista_annual_income) ÷ withdrawal_rate"
								>
									Barista: <span class="font-bold">{formatPLN(baristaNum)}</span>
								</div>
							</div>
							<div class="space-y-1">
								<div class="text-xs text-surface-700-300">Barista FI progress</div>
								<div class="text-2xl font-bold">
									{baristaProgress != null ? `${baristaProgress.toFixed(1)}%` : '—'}
								</div>
								{#if baristaYears != null}
									<div class="text-xs text-surface-700-300">
										FI za ~{baristaYears.toFixed(1)} lat (bez dopłat, {coastReturnPct}%/rok)
									</div>
								{:else if (baristaProgress ?? 0) >= 100}
									<div class="text-xs text-success-600-400">już osiągnięto Barista FIRE</div>
								{/if}
							</div>
						</div>
					</div>
				{/if}
				{#if bridgeYears != null && bridgeNeeded != null && bridgeLiquid != null && bridgeGap != null}
					{@const bridgeSurplus = bridgeGap < 0}
					{@const bridgeExact = bridgeGap === 0}
					{@const bridgeOK = bridgeGap <= 0}
					{@const bridgeYearsLabel = plYears(bridgeYears)}
					<div class="pt-3 border-t border-surface-200-800 space-y-2">
						<div
							class="text-xs text-surface-700-300"
							title="bridge_capital_needed = annual_expenses × (60 − current_age)&#10;bridge_liquid_capital = wartość netto − (IKE + IKZE + PPK)&#10;bridge_capital_gap = needed − liquid (dodatni = niedobór, ujemny = nadwyżka)"
						>
							Bridge do 60 ({bridgeYears}
							{bridgeYearsLabel})
						</div>
						<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
							<div class="space-y-1">
								<div class="text-xs text-surface-700-300">Potrzebne</div>
								<div class="text-xl font-bold">{formatPLN(bridgeNeeded)}</div>
								<div class="text-xs text-surface-700-300">wydatki × lata do 60</div>
							</div>
							<div class="space-y-1">
								<div class="text-xs text-surface-700-300">Płynne</div>
								<div class="text-xl font-bold">{formatPLN(bridgeLiquid)}</div>
								<div class="text-xs text-surface-700-300">netto bez IKE/IKZE/PPK</div>
							</div>
							<div class="space-y-1">
								<div class="text-xs text-surface-700-300">
									{#if bridgeSurplus}Nadwyżka{:else if bridgeExact}Pokryty{:else}Brakuje{/if}
								</div>
								<div
									class="text-xl font-bold {bridgeOK
										? 'text-success-600-400'
										: 'text-warning-600-400'}"
								>
									{formatPLN(Math.abs(bridgeGap))}
								</div>
								<div class="text-xs text-surface-700-300">
									{bridgeOK ? 'mostek pokryty' : 'do uzbierania w aktywach płynnych'}
								</div>
							</div>
						</div>
					</div>
				{/if}
			</div>
		{/if}

		{#if dashboard.treasuryBondsCount > 0}
			<a
				href="/bonds"
				class="card preset-filled-surface-100-900 p-4 space-y-2 block hover:preset-tonal-primary transition-colors"
			>
				<header>
					<h3 class="h4 flex items-center gap-2"><Coins size={18} /> Obligacje skarbowe</h3>
				</header>
				<div class="text-3xl font-bold text-primary-600-400">
					{formatPLN(dashboard.treasuryBondsValue)}
				</div>
				<p class="text-xs text-surface-700-300">
					{dashboard.treasuryBondsCount}
					{dashboard.treasuryBondsCount === 1 ? 'obligacja' : 'obligacji'} (auto-wycena wg CPI)
				</p>
				{#if dashboard.bondsNextMaturity && dashboard.bondsNextMaturity.days_until <= 90}
					{@const nm = dashboard.bondsNextMaturity}
					{@const urgent = nm.days_until <= 30}
					<div
						class="card p-2 text-xs flex items-center gap-1 {urgent
							? 'preset-filled-error-500'
							: 'preset-filled-warning-500'}"
					>
						<AlertTriangle size={14} />
						<span>
							Wykup {nm.type} za {nm.days_until}
							{daysLabel(nm.days_until)} ({formatPLN(nm.net_cashflow)} netto)
						</span>
					</div>
				{/if}
			</a>
		{/if}

		{#if dashboard.retirementStats && dashboard.retirementStats.length > 0}
			<div class="card preset-filled-surface-100-900 p-4 space-y-4">
				<header class="flex items-center justify-between gap-2 flex-wrap">
					<h3 class="h3 flex items-center gap-2">
						<PiggyBank size={20} /> Limity Emerytalne {data.currentYear}
					</h3>
					<div class="flex items-center gap-2">
						<span class="text-xs text-surface-700-300 whitespace-nowrap">
							{daysLeft}
							{daysLabel(daysLeft)} do 31 grudnia
						</span>
						<button
							type="button"
							class="btn-icon btn-icon-sm"
							aria-label="Konfiguruj limity"
							onclick={() => openLimitsModal(dashboard.retirementStats)}
						>
							<Settings size={18} />
						</button>
					</div>
				</header>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					{#each dashboard.retirementStats as stat}
						{@const tier = countdownTier(now, stat.percentage_used)}
						<div class="card preset-tonal-surface p-4 space-y-2">
							<div class="flex items-start justify-between gap-2">
								<h4 class="font-bold">
									{stat.account_wrapper} ({ownerName(owners, stat.owner_user_id)})
								</h4>
								<span class="text-sm font-semibold whitespace-nowrap">
									{formatPLN(stat.total_contributed)} / {formatPLN(stat.limit_amount)}
								</span>
							</div>

							<div class="h-2 rounded-full bg-surface-200-800 overflow-hidden">
								<div
									class="h-full transition-all {tierBar[tier]}"
									style="width: {Math.min(stat.percentage_used, 100)}%"
								></div>
							</div>

							<div class="flex items-center justify-between text-sm">
								<span class="text-surface-700-300">Pozostało: {formatPLN(stat.remaining)}</span>
								<span class="font-semibold {tierText[tier]}">
									{stat.percentage_used}%
								</span>
							</div>

							{#if tier === 'maxed'}
								<div
									class="card preset-filled-success-500 p-2 text-xs text-center flex items-center justify-center gap-1"
								>
									<CheckCircle2 size={14} /> Limit osiągnięty
								</div>
							{:else if tier === 'urgent'}
								<div
									class="card preset-filled-error-500 p-2 text-xs text-center flex items-center justify-center gap-1"
								>
									<AlertTriangle size={14} /> Końcówka roku — limit przepadnie 31.12
								</div>
							{:else if tier === 'warn'}
								<div
									class="card preset-filled-warning-500 p-2 text-xs text-center flex items-center justify-center gap-1"
								>
									<AlertTriangle size={14} /> Coraz mniej czasu na wykorzystanie limitu
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{/if}

		{#if dashboard.allocationDrift?.scopes?.length > 0}
			<AllocationDriftWidget drift={dashboard.allocationDrift} {owners} />
		{/if}

		<DashboardCharts
			netWorthHistory={dashboard.net_worth_history}
			allocation={dashboard.allocation}
			{owners}
		/>
	{:catch err}
		<div class="card preset-filled-error-500 p-4">
			<p class="font-semibold">Nie udało się załadować danych dashboardu.</p>
			<p class="text-sm">{err?.message ?? 'Spróbuj ponownie później.'}</p>
		</div>
	{/await}
</div>

<Modal
	open={showLimitsModal}
	title="Konfiguracja Limitów Emerytalnych"
	onCancel={() => (showLimitsModal = false)}
	onConfirm={saveLimits}
	confirmText="Zapisz"
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			saveLimits();
		}}
		class="space-y-4"
	>
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
				{#each owners as owner}
					<label class="label">
						<span class="font-semibold text-sm">{wrapper} {owner.name} (PLN)</span>
						<input
							type="number"
							class="input"
							bind:value={limits[`${wrapper}_${owner.id}`]}
							step="0.01"
							required
						/>
					</label>
				{/each}
			{/each}
		</div>
	</form>
</Modal>
