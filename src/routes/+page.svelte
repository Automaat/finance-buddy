<script lang="ts">
	import { untrack } from 'svelte';
	import Modal from '$lib/components/Modal.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import DashboardCharts from '$lib/components/DashboardCharts.svelte';
	import DeltaBadge from '$lib/components/DeltaBadge.svelte';
	import { formatPLN } from '$lib/utils/format';
	import {
		Wallet,
		TrendingUp,
		TrendingDown,
		Settings,
		CheckCircle2,
		AlertTriangle,
		PiggyBank
	} from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Persona } from '$lib/types/personas';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	let personas: Persona[] = $state([]);

	$effect(() => {
		let cancelled = false;
		Promise.resolve(data.personas).then((p) => {
			if (!cancelled) personas = (p ?? []) as Persona[];
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
		owner: string;
		total_contributed: number;
		limit_amount: number;
		remaining: number;
		percentage_used: number;
	};

	function openLimitsModal(retirementStats: RetirementStat[]) {
		limits = {};
		for (const persona of personas) {
			for (const wrapper of ['IKE', 'IKZE']) {
				limits[`${wrapper}_${persona.name}`] = 0;
			}
		}
		for (const stat of retirementStats) {
			const key = `${stat.account_wrapper}_${stat.owner}`;
			if (key in limits) {
				limits[key] = stat.limit_amount || 0;
			}
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
			toast.error('Nie udało się zapisać limitów. Spróbuj ponownie później.');
		}
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
				<div class="text-3xl font-bold text-success-600-400">{formatPLN(dashboard.total_assets)}</div>
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
				<div class="text-3xl font-bold text-error-600-400">{formatPLN(dashboard.total_liabilities)}</div>
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

		{#if dashboard.retirementStats && dashboard.retirementStats.length > 0}
			<div class="card preset-filled-surface-100-900 p-4 space-y-4">
				<header class="flex items-center justify-between gap-2">
					<h3 class="h3 flex items-center gap-2">
						<PiggyBank size={20} /> Limity Emerytalne {data.currentYear}
					</h3>
					<button
						type="button"
						class="btn-icon btn-icon-sm"
						aria-label="Konfiguruj limity"
						onclick={() => openLimitsModal(dashboard.retirementStats)}
					>
						<Settings size={18} />
					</button>
				</header>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					{#each dashboard.retirementStats as stat}
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
								<div
									class="card preset-filled-success-500 p-2 text-xs text-center flex items-center justify-center gap-1"
								>
									<CheckCircle2 size={14} /> Limit osiągnięty
								</div>
							{:else if stat.percentage_used >= 50}
								<div
									class="card preset-filled-warning-500 p-2 text-xs text-center flex items-center justify-center gap-1"
								>
									<AlertTriangle size={14} /> Zbliżasz się do limitu
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<DashboardCharts
			netWorthHistory={dashboard.net_worth_history}
			allocation={dashboard.allocation}
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
