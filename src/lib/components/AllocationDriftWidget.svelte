<script lang="ts">
	import { formatPLN, formatSignedPLN } from '$lib/utils/format';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import { Scale, AlertTriangle, CheckCircle2, ArrowUp, ArrowDown } from 'lucide-svelte';

	interface DriftItem {
		category: string;
		owner_user_id: number | null;
		current_value: number;
		current_percentage: number;
		target_percentage: number;
		drift_pp: number;
		severity: 'ok' | 'warning' | 'missing_target';
		rebalance_amount: number;
	}

	interface DriftScope {
		owner_user_id: number | null;
		total_value: number;
		target_sum_pct: number;
		has_complete_target: boolean;
		items: DriftItem[];
	}

	interface Props {
		drift: { scopes: DriftScope[] };
		owners: OwnerOption[];
	}

	const CATEGORY_LABELS: Record<string, string> = {
		bank: 'Bank',
		saving_account: 'Konto oszczędn.',
		stock: 'Akcje',
		bond: 'Obligacje',
		gold: 'Złoto',
		real_estate: 'Nieruchomości',
		ppk: 'PPK',
		fund: 'Fundusze',
		etf: 'ETF',
		vehicle: 'Pojazdy'
	};

	let { drift, owners }: Props = $props();

	function scopeLabel(ownerId: number | null): string {
		if (ownerId === null) return 'Wspólne';
		return ownerName(owners, ownerId);
	}

	function categoryLabel(value: string): string {
		return CATEGORY_LABELS[value] ?? value;
	}

	function barWidth(pct: number): string {
		return `${Math.max(0, Math.min(100, pct))}%`;
	}
</script>

{#if drift.scopes.length > 0}
	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="flex items-center gap-2">
			<Scale size={20} />
			<h3 class="h3">Dryft alokacji</h3>
		</header>

		{#each drift.scopes as scope (scope.owner_user_id ?? 'household')}
			<section class="space-y-3">
				<header class="flex items-center justify-between flex-wrap gap-2">
					<h4 class="h4 font-semibold">{scopeLabel(scope.owner_user_id)}</h4>
					<span class="text-xs text-surface-700-300">
						Suma celów: {scope.target_sum_pct.toFixed(2)}% · Aktywa: {formatPLN(scope.total_value)}
					</span>
				</header>

				{#if !scope.has_complete_target}
					<div class="card preset-tonal-warning p-2 text-xs flex items-center gap-2">
						<AlertTriangle size={14} />
						Cele dla tego zakresu nie sumują się do 100%. Uzupełnij w
						<a href="/settings/allocation" class="underline">ustawieniach</a>.
					</div>
				{/if}

				<div class="space-y-2">
					{#each scope.items as item (item.category)}
						{@const overTarget = item.current_percentage > item.target_percentage}
						<div class="space-y-1">
							<div class="flex items-center justify-between gap-2 text-sm">
								<span class="font-semibold">{categoryLabel(item.category)}</span>
								<div class="flex items-center gap-2">
									{#if item.severity === 'warning'}
										<span class="badge preset-filled-warning-500 flex items-center gap-1">
											<AlertTriangle size={12} /> Dryft {item.drift_pp >= 0
												? '+'
												: ''}{item.drift_pp.toFixed(1)} pp
										</span>
									{:else if item.severity === 'missing_target'}
										<span class="badge preset-tonal-surface">Brak celu</span>
									{:else}
										<span class="badge preset-filled-success-500 flex items-center gap-1"
											><CheckCircle2 size={12} /> OK</span
										>
									{/if}
								</div>
							</div>

							<div
								class="h-3 rounded-full bg-surface-200-800 overflow-hidden relative"
								title="Aktualne: {item.current_percentage.toFixed(
									1
								)}% · Cel: {item.target_percentage.toFixed(1)}%"
							>
								<div
									class="absolute inset-y-0 left-0 bg-surface-300-700"
									style="width: {barWidth(item.target_percentage)}"
								></div>
								<div
									class="absolute inset-y-0 left-0 {item.severity === 'warning'
										? 'bg-warning-500'
										: item.severity === 'missing_target'
											? 'bg-error-400'
											: 'bg-primary-500'} opacity-80"
									style="width: {barWidth(item.current_percentage)}"
								></div>
							</div>

							<div class="flex items-center justify-between text-xs text-surface-700-300">
								<span>
									{item.current_percentage.toFixed(1)}% ({formatPLN(item.current_value)}) · cel {item.target_percentage.toFixed(
										1
									)}%
								</span>
								{#if Math.abs(item.rebalance_amount) >= 1}
									<span
										class="font-semibold inline-flex items-center gap-1 {overTarget
											? 'text-error-600-400'
											: 'text-success-600-400'}"
									>
										{#if overTarget}
											<ArrowDown size={12} /> SPRZEDAJ
										{:else}
											<ArrowUp size={12} /> DOKUP
										{/if}
										{formatSignedPLN(item.rebalance_amount)}
									</span>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/each}
	</div>
{/if}
