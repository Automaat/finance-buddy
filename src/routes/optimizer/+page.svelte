<script lang="ts">
	import { untrack } from 'svelte';
	import { rankOptions, type OptionInputs, type OptionKey } from '$lib/utils/allocationOptimizer';
	import { formatPLN } from '$lib/utils/format';
	import { Compass, TrendingUp } from 'lucide-svelte';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}
	let { data }: Props = $props();

	let amount = $state(1000);
	let marginalPitRate = $state(0.32);
	// IKE/IKZE remaining limits come from /api/retirement/stats — summed across
	// owners for the current year. Read-only here; managed on the retirement page.
	const ikzeRemainingPLN = $derived(data?.ikzeRemainingPLN ?? 0);
	const ikeRemainingPLN = $derived(data?.ikeRemainingPLN ?? 0);
	let mortgageAPRPct = $state(7);
	let mortgageRemainingPLN = $state(0);
	let bondsYieldPct = $state(6.75);
	let brokerageReturnPct = $state(untrack(() => data?.defaults?.annualReturnPct ?? 7));
	let liquidityNeedScore = $state(1);

	const FALLBACK_DRIFT: Record<OptionKey, number> = {
		ikze: 0,
		ike: 0,
		mortgage: 0,
		bonds: 0,
		brokerage: 0
	};

	const inputs = $derived<OptionInputs>({
		amountPLN: amount,
		marginalPitRate,
		ikzeRemainingPLN,
		ikeRemainingPLN,
		mortgageAPRPct,
		mortgageRemainingPLN,
		bondsYieldPct,
		brokerageReturnPct,
		allocationDrift: data?.allocationDrift ?? FALLBACK_DRIFT,
		liquidityNeedScore
	});

	const ranked = $derived(rankOptions(inputs));

	function impactPLN(scorePP: number): number {
		return (amount * scorePP) / 100;
	}
</script>

<svelte:head>
	<title>Optymalizator | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-6">
	<header class="space-y-1">
		<h1 class="h2 flex items-center gap-2">
			<Compass size={24} class="text-primary-500" />
			Optymalizator następnych 1000 PLN
		</h1>
		<p class="text-surface-700-300 text-sm">
			Porównaj IKZE, IKE, nadpłatę hipoteki, obligacje i maklerskie — ranking według oczekiwanej
			korzyści w pkt. proc.
		</p>
	</header>

	<section class="card preset-filled-surface-100-900 p-5 space-y-4">
		<h2 class="h4">Parametry</h2>
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			<label class="space-y-1">
				<span class="text-xs font-semibold">Kwota (PLN)</span>
				<input type="number" min="0" class="input w-full" bind:value={amount} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Krańcowa stawka PIT (np. 0.32)</span>
				<input
					type="number"
					min="0"
					max="1"
					step="0.01"
					class="input w-full"
					bind:value={marginalPitRate}
				/>
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">APR kredytu hipotecznego (%)</span>
				<input type="number" min="0" step="0.1" class="input w-full" bind:value={mortgageAPRPct} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Pozostały kapitał hipoteki (PLN)</span>
				<input type="number" min="0" class="input w-full" bind:value={mortgageRemainingPLN} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Oczekiwany zysk obligacji (% brutto)</span>
				<input type="number" min="0" step="0.1" class="input w-full" bind:value={bondsYieldPct} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Oczekiwany zwrot maklerskiego (% brutto)</span>
				<input
					type="number"
					min="0"
					step="0.1"
					class="input w-full"
					bind:value={brokerageReturnPct}
				/>
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Potrzeba płynności (0-5)</span>
				<input
					type="number"
					min="0"
					max="5"
					step="1"
					class="input w-full"
					bind:value={liquidityNeedScore}
				/>
			</label>
		</div>
	</section>

	<section class="card preset-filled-surface-100-900 p-5 space-y-3">
		<h2 class="h4 flex items-center gap-2">
			<TrendingUp size={20} class="text-primary-500" />
			Ranking
		</h2>
		<ol class="space-y-3">
			{#each ranked as r, idx (r.option)}
				<li class="card preset-tonal-surface p-4 space-y-2 {r.available ? '' : 'opacity-60'}">
					<header class="flex flex-wrap items-baseline justify-between gap-2">
						<div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
							<span class="text-2xl font-bold">{idx + 1}.</span>
							<span class="text-lg font-semibold">{r.name}</span>
							{#if !r.available}
								<span class="badge preset-tonal-warning">{r.availabilityReason}</span>
							{/if}
						</div>
						<div class="text-right">
							<div class="text-xs text-surface-600-400">Wynik</div>
							<div class="text-xl font-bold">{r.total.toFixed(1)} pp</div>
							<div class="text-xs text-surface-600-400">
								≈ {formatPLN(impactPLN(r.total))}/rok przy {formatPLN(amount)}
							</div>
						</div>
					</header>
					<dl class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs">
						{#each r.factors as f (f.label)}
							<div>
								<dt class="text-surface-600-400">{f.label}</dt>
								<dd class="font-semibold {f.pp >= 0 ? '' : 'text-error-600-400'}">
									{f.pp >= 0 ? '+' : ''}{f.pp.toFixed(1)} pp
								</dd>
							</div>
						{/each}
					</dl>
				</li>
			{/each}
		</ol>
		<p class="text-xs text-surface-600-400">
			Wynik wyrażony w pkt. proc. odnosi się do kwoty wpłaty: 5 pp ≈ 5% nadwyżki w pierwszym roku.
			Tę kalkulację należy traktować jako szybką orientację — Belka i ulga IKZE są przybliżeniami.
			Złoty standard to indywidualny rachunek z doradcą.
		</p>
	</section>
</div>
