<script lang="ts">
	import { formatPLN, formatDate } from '$lib/utils/format';
	import { daysLabel } from '$lib/utils/yearEnd';
	import { CalendarClock, Coins, ArrowDownToLine } from 'lucide-svelte';
	import type {
		MaturityLadderEvent,
		NextMaturityWarning,
		LadderEventKind
	} from '../../routes/investments/bonds/+page';

	interface Props {
		events: MaturityLadderEvent[];
		nextMaturity: NextMaturityWarning | null;
		taxRatePct: number;
	}

	let { events, nextMaturity, taxRatePct }: Props = $props();

	const monthFormatter = new Intl.DateTimeFormat('pl-PL', {
		year: 'numeric',
		month: 'long'
	});

	function formatMonth(iso: string): string {
		const [y, m] = iso.split('-').map(Number);
		return monthFormatter.format(new Date(y, m - 1, 1));
	}

	function kindLabel(kind: LadderEventKind): string {
		return kind === 'redemption' ? 'Wykup' : 'Kupon';
	}

	type Grouped = {
		monthISO: string;
		monthLabel: string;
		totalNet: number;
		events: MaturityLadderEvent[];
	};

	const groupedByMonth = $derived.by<Grouped[]>(() => {
		const map = new Map<string, Grouped>();
		for (const ev of events) {
			const existing = map.get(ev.month);
			if (existing) {
				existing.events.push(ev);
				existing.totalNet += ev.net_cashflow;
			} else {
				map.set(ev.month, {
					monthISO: ev.month,
					monthLabel: formatMonth(ev.month),
					totalNet: ev.net_cashflow,
					events: [ev]
				});
			}
		}
		return [...map.values()].sort((a, b) => a.monthISO.localeCompare(b.monthISO));
	});
</script>

<div class="card preset-filled-surface-100-900 p-4 space-y-4">
	<header class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
		<h3 class="h3 flex items-center gap-2">
			<CalendarClock size={20} /> Kalendarz wykupu
		</h3>
		<p class="text-xs text-surface-700-300">
			Wartości netto po podatku Belki ({taxRatePct.toFixed(0)}%)
		</p>
	</header>

	{#if nextMaturity}
		{@const tier =
			nextMaturity.days_until <= 30 ? 'urgent' : nextMaturity.days_until <= 90 ? 'warn' : 'info'}
		<div
			class="card p-3 text-sm flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 {tier ===
			'urgent'
				? 'preset-filled-error-500'
				: tier === 'warn'
					? 'preset-filled-warning-500'
					: 'preset-tonal-primary'}"
		>
			<div class="flex items-center gap-2">
				<ArrowDownToLine size={16} />
				<span>
					Najbliższy wykup: <strong>{nextMaturity.type}</strong> · {nextMaturity.count}
					{nextMaturity.count === 1 ? 'obligacja' : 'obligacji'}
					· {formatDate(nextMaturity.date)} ({nextMaturity.days_until}
					{daysLabel(nextMaturity.days_until)})
				</span>
			</div>
			<span class="font-semibold">{formatPLN(nextMaturity.net_cashflow)} netto</span>
		</div>
	{/if}

	{#if groupedByMonth.length === 0}
		<p class="text-sm text-surface-700-300 text-center py-6">
			Brak nadchodzących przepływów. Dodaj obligacje, aby zobaczyć kalendarz.
		</p>
	{:else}
		<div class="space-y-3">
			{#each groupedByMonth as group (group.monthISO)}
				<section class="space-y-2">
					<header class="flex items-center justify-between gap-2 flex-wrap">
						<h4 class="font-semibold capitalize">{group.monthLabel}</h4>
						<span class="text-sm text-surface-700-300">
							Łącznie netto: <strong class="text-primary-600-400"
								>{formatPLN(group.totalNet)}</strong
							>
						</span>
					</header>
					<div class="table-wrap">
						<table class="table table-hover text-sm">
							<thead>
								<tr>
									<th>Typ</th>
									<th>Zdarzenie</th>
									<th class="text-right">Liczba</th>
									<th class="text-right">Kapitał</th>
									<th class="text-right">Odsetki brutto</th>
									<th class="text-right">Podatek</th>
									<th class="text-right">Netto</th>
								</tr>
							</thead>
							<tbody>
								{#each group.events as ev (ev.type + ev.kind)}
									<tr>
										<td class="font-medium">{ev.type}</td>
										<td>
											<span class="badge preset-tonal-surface inline-flex items-center gap-1">
												{#if ev.kind === 'coupon'}<Coins size={12} />{:else}<ArrowDownToLine
														size={12}
													/>{/if}
												{kindLabel(ev.kind)}
											</span>
										</td>
										<td class="text-right">{ev.count}</td>
										<td class="text-right">
											{ev.principal > 0 ? formatPLN(ev.principal) : '—'}
										</td>
										<td class="text-right">
											{ev.interest_gross > 0 ? formatPLN(ev.interest_gross) : '—'}
										</td>
										<td class="text-right text-surface-700-300">
											{ev.tax > 0 ? formatPLN(ev.tax) : '—'}
										</td>
										<td class="text-right font-semibold text-primary-600-400">
											{formatPLN(ev.net_cashflow)}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</section>
			{/each}
		</div>
	{/if}
</div>
