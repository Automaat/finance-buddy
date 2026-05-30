<script lang="ts">
	import MetricCard from '$lib/components/MetricCard.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { ownerName } from '$lib/types/owners';
	import { PiggyBank, Wallet } from 'lucide-svelte';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}
	let { data }: Props = $props();

	// Household-wide rollup across every owner's PPK summary. ROI is recomputed
	// from the summed totals rather than averaged, so it stays contribution-
	// weighted.
	const totalContributed = $derived(data.stats.reduce((s, p) => s + p.total_contributed, 0));
	const totalValue = $derived(data.stats.reduce((s, p) => s + p.total_value, 0));
	const totalReturns = $derived(data.stats.reduce((s, p) => s + p.returns, 0));
	const roiPct = $derived(totalContributed > 0 ? (totalReturns / totalContributed) * 100 : 0);
	const activeAccounts = $derived(data.accounts.filter((a) => a.is_active).length);
</script>

<svelte:head>
	<title>PPK | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-1 mb-6">
	<h1 class="h2">PPK</h1>
	<p class="text-surface-700-300 text-sm">
		Pracownicze Plany Kapitałowe — wpłaty pracownika, pracodawcy i dopłaty państwa.
	</p>
</div>

{#if data.stats.length === 0}
	<div class="card preset-filled-surface-100-900 p-4">
		<div class="text-center py-12 text-surface-700-300">
			<p>Brak danych PPK</p>
			<p class="text-sm mt-1">Dodaj konto z opakowaniem PPK i wpłaty, aby zobaczyć podsumowanie.</p>
		</div>
	</div>
{:else}
	<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
		<div class="card preset-filled-surface-100-900 p-4 space-y-1">
			<header class="text-sm text-surface-700-300">Wpłacono</header>
			<div class="text-2xl font-bold">{formatPLN(totalContributed)}</div>
		</div>
		<div class="card preset-filled-surface-100-900 p-4 space-y-1">
			<header class="text-sm text-surface-700-300">Wartość bieżąca</header>
			<div class="text-2xl font-bold text-primary-600-400">{formatPLN(totalValue)}</div>
		</div>
		<div class="card preset-filled-surface-100-900 p-4 space-y-1">
			<header class="text-sm text-surface-700-300">Zysk</header>
			<div class="text-2xl font-bold {totalReturns >= 0 ? 'text-success-500' : 'text-error-500'}">
				{totalReturns >= 0 ? '+' : ''}{formatPLN(totalReturns)}
			</div>
			<div class="text-xs text-surface-600-400">
				{totalReturns >= 0 ? '+' : ''}{roiPct.toFixed(2)}%
			</div>
		</div>
		<div class="card preset-filled-surface-100-900 p-4 space-y-1">
			<header class="text-sm text-surface-700-300">Liczba kont</header>
			<div class="text-2xl font-bold">{activeAccounts}</div>
		</div>
	</div>

	<div class="space-y-6">
		{#each data.stats as stat (stat.owner_user_id)}
			<div class="card preset-filled-surface-100-900 p-4 space-y-4">
				<header>
					<h3 class="h3 flex items-center gap-2">
						<PiggyBank size={20} />
						{ownerName(data.owners, stat.owner_user_id)}
					</h3>
				</header>
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					<MetricCard
						label="Wartość całkowita"
						value={stat.total_value}
						decimals={0}
						suffix=" PLN"
						color="green"
					/>
					<MetricCard
						label="Wpłaty pracownika"
						value={stat.employee_contributed}
						decimals={0}
						suffix=" PLN"
						color="blue"
					/>
					<MetricCard
						label="Wpłaty pracodawcy"
						value={stat.employer_contributed}
						decimals={0}
						suffix=" PLN"
						color="blue"
					/>
					<MetricCard
						label="Dopłaty państwa"
						value={stat.government_contributed}
						decimals={0}
						suffix=" PLN"
						color="blue"
					/>
					<MetricCard
						label="Łącznie wpłacone"
						value={stat.total_contributed}
						decimals={0}
						suffix=" PLN"
						color="blue"
					/>
					<MetricCard
						label="Zyski z inwestycji"
						value={stat.returns}
						decimals={0}
						suffix=" PLN"
						color={stat.returns >= 0 ? 'green' : 'red'}
					/>
					<MetricCard
						label="ROI"
						value={stat.roi_percentage}
						decimals={2}
						suffix="%"
						color={stat.roi_percentage >= 0 ? 'green' : 'red'}
					/>
				</div>
			</div>
		{/each}

		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header>
				<h3 class="h3 flex items-center gap-2"><Wallet size={20} /> Konta PPK</h3>
			</header>

			{#if data.accounts.length === 0}
				<div class="text-center py-12 text-surface-700-300">
					<p>Brak kont PPK</p>
				</div>
			{:else}
				<div class="table-wrap">
					<table class="table table-hover">
						<thead>
							<tr>
								<th>Konto</th>
								<th>Właściciel</th>
								<th>Status</th>
								<th class="text-right">Wartość bieżąca</th>
							</tr>
						</thead>
						<tbody>
							{#each data.accounts as account (account.id)}
								<tr>
									<td class="font-medium">{account.name}</td>
									<td>{ownerName(data.owners, account.owner_user_id)}</td>
									<td class="text-xs text-surface-700-300">
										{account.is_active ? 'Aktywne' : 'Nieaktywne'}
									</td>
									<td class="text-right font-semibold text-primary-600-400">
										{formatPLN(account.current_value)}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
	</div>
{/if}
