<script lang="ts">
	import { rankOptions, type OptionInputs, type OptionKey } from '$lib/utils/allocationOptimizer';
	import { formatPLN } from '$lib/utils/format';
	import { Compass, TrendingUp } from 'lucide-svelte';

	let amount = $state(1000);
	let marginalPitRate = $state(0.32);
	let ikzeRemainingPLN = $state(5000);
	let ikeRemainingPLN = $state(15000);
	let ppkEmployerMatchRate = $state(0.015);
	let ppkMatched = $state(true);
	let mortgageAPRPct = $state(7);
	let mortgageRemainingPLN = $state(0);
	let bondsYieldPct = $state(6.75);
	let brokerageReturnPct = $state(7);
	let liquidityNeedScore = $state(1);

	type DriftEntry = { key: OptionKey; label: string; pp: number };
	let drift = $state<DriftEntry[]>([
		{ key: 'ikze', label: 'IKZE', pp: 0 },
		{ key: 'ike', label: 'IKE', pp: 0 },
		{ key: 'ppk', label: 'PPK', pp: 0 },
		{ key: 'mortgage', label: 'Nadpłata hipoteki', pp: 0 },
		{ key: 'bonds', label: 'Obligacje', pp: 0 },
		{ key: 'brokerage', label: 'Maklerskie', pp: 0 }
	]);

	const inputs = $derived<OptionInputs>({
		marginalPitRate,
		ikzeRemainingPLN,
		ikeRemainingPLN,
		ppkEmployerMatchRate,
		ppkMatched,
		mortgageAPRPct,
		mortgageRemainingPLN,
		bondsYieldPct,
		brokerageReturnPct,
		allocationDrift: Object.fromEntries(drift.map((d) => [d.key, d.pp])) as Record<
			OptionKey,
			number
		>,
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
			Porównaj IKZE, IKE, PPK, nadpłatę hipoteki, obligacje i maklerskie — ranking według
			oczekiwanej korzyści w pkt. proc.
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
				<span class="text-xs font-semibold">Pozostały limit IKZE (PLN)</span>
				<input type="number" min="0" class="input w-full" bind:value={ikzeRemainingPLN} />
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Pozostały limit IKE (PLN)</span>
				<input type="number" min="0" class="input w-full" bind:value={ikeRemainingPLN} />
			</label>
			<label class="space-y-1 flex flex-col">
				<span class="text-xs font-semibold">PPK aktywne</span>
				<select bind:value={ppkMatched} class="input">
					<option value={true}>Tak</option>
					<option value={false}>Nie</option>
				</select>
			</label>
			<label class="space-y-1">
				<span class="text-xs font-semibold">Stawka pracodawcy PPK (np. 0.015)</span>
				<input
					type="number"
					min="0"
					max="1"
					step="0.001"
					class="input w-full"
					bind:value={ppkEmployerMatchRate}
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

		<h3 class="h5">Dryft alokacji (pp, „+” = pod celem)</h3>
		<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
			{#each drift as d (d.key)}
				<label class="space-y-1">
					<span class="text-xs font-semibold">{d.label}</span>
					<input type="number" step="0.5" class="input w-full" bind:value={d.pp} />
				</label>
			{/each}
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
						<div class="flex items-baseline gap-3">
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
			Tę kalkulację należy traktować jako szybką orientację — Belka, ulga IKZE i dopłata PPK są
			przybliżeniami. Złoty standard to indywidualny rachunek z doradcą.
		</p>
	</section>
</div>
