<script lang="ts">
	import { untrack } from 'svelte';
	import { invalidateAll } from '$app/navigation';
	import { resolveApiUrl } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import PIT38Report from '$lib/components/PIT38Report.svelte';
	import { Plus, Trash2, BarChart, RefreshCw, Coins } from 'lucide-svelte';
	import { CrudForm } from '$lib/stores/crudForm.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const securityLabel = (id: number): string => {
		const s = data.securities.find((sec) => sec.id === id);
		return s ? s.symbol : `#${id}`;
	};
	const accountLabel = (id: number): string =>
		data.accounts.find((a) => a.id === id)?.name ?? `#${id}`;

	const ASSET_TYPES = [
		{ value: 'stock', label: 'Akcja' },
		{ value: 'etf', label: 'ETF' },
		{ value: 'bond', label: 'Obligacja' },
		{ value: 'fund', label: 'Fundusz' }
	];

	const securityCrud = new CrudForm();
	const lotCrud = new CrudForm();
	const quoteCrud = new CrudForm();
	const dividendCrud = new CrudForm();
	let refreshing = $state(false);

	let dividendForm = $state(
		untrack(() => ({
			account_id: data.accounts[0]?.id ?? 0,
			security_id: data.securities[0]?.id ?? 0,
			pay_date: new Date().toISOString().slice(0, 10),
			gross_amount: '0',
			withholding_tax: '0'
		}))
	);

	let securityForm = $state({
		symbol: '',
		isin: '',
		name: '',
		asset_type: 'stock',
		currency: 'PLN'
	});

	let lotForm = $state(
		untrack(() => ({
			account_id: data.accounts[0]?.id ?? 0,
			security_id: data.securities[0]?.id ?? 0,
			side: 'buy',
			quantity: '0',
			price: '0',
			fee: '0',
			date: new Date().toISOString().slice(0, 10)
		}))
	);

	let quoteForm = $state(
		untrack(() => ({
			security_id: data.securities[0]?.id ?? 0,
			date: new Date().toISOString().slice(0, 10),
			price: '0'
		}))
	);

	// saveVia wraps a create through a CrudForm (saving/error/close bookkeeping)
	// and surfaces the outcome as a toast — holdings reports via toast rather
	// than inline error.
	async function saveVia(form: CrudForm, action: () => Promise<void>, successMsg: string) {
		const ok = await form.submit(action);
		if (ok) {
			toast.success(successMsg);
		} else {
			toast.error(form.error);
		}
	}

	function openSecurityModal() {
		securityForm = { symbol: '', isin: '', name: '', asset_type: 'stock', currency: 'PLN' };
		securityCrud.openCreate();
	}

	function openLotModal() {
		if (data.accounts.length === 0) {
			toast.error('Najpierw dodaj konto.');
			return;
		}
		if (data.securities.length === 0) {
			toast.error('Najpierw dodaj papier wartościowy.');
			return;
		}
		lotForm = {
			account_id: data.accounts[0].id,
			security_id: data.securities[0].id,
			side: 'buy',
			quantity: '0',
			price: '0',
			fee: '0',
			date: new Date().toISOString().slice(0, 10)
		};
		lotCrud.openCreate();
	}

	function openQuoteModal() {
		if (data.securities.length === 0) {
			toast.error('Najpierw dodaj papier wartościowy.');
			return;
		}
		quoteForm = {
			security_id: data.securities[0].id,
			date: new Date().toISOString().slice(0, 10),
			price: '0'
		};
		quoteCrud.openCreate();
	}

	function openDividendModal() {
		if (data.accounts.length === 0) {
			toast.error('Najpierw dodaj konto.');
			return;
		}
		if (data.securities.length === 0) {
			toast.error('Najpierw dodaj papier wartościowy.');
			return;
		}
		dividendForm = {
			account_id: data.accounts[0].id,
			security_id: data.securities[0].id,
			pay_date: new Date().toISOString().slice(0, 10),
			gross_amount: '0',
			withholding_tax: '0'
		};
		dividendCrud.openCreate();
	}

	async function saveDividend() {
		const apiUrl = resolveApiUrl();
		await saveVia(
			dividendCrud,
			async () => {
				const res = await fetch(`${apiUrl}/api/holdings/dividends`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(dividendForm)
				});
				if (!res.ok) {
					const d = await res.json().catch(() => ({ detail: res.statusText }));
					throw new Error(d.detail ?? res.statusText);
				}
				await invalidateAll();
			},
			'Dodano dywidendę'
		);
	}

	async function deleteDividend(id: number) {
		const ok = await confirm({
			title: 'Usunąć dywidendę?',
			message: 'Wpis dywidendy zostanie trwale usunięty.',
			danger: true,
			confirmText: 'Usuń'
		});
		if (!ok) return;
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/holdings/dividends/${id}`, { method: 'DELETE' });
			if (!res.ok) {
				const d = await res.json().catch(() => ({ detail: res.statusText }));
				throw new Error(d.detail ?? res.statusText);
			}
			toast.success('Usunięto');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		}
	}

	async function saveSecurity() {
		const apiUrl = resolveApiUrl();
		await saveVia(
			securityCrud,
			async () => {
				const res = await fetch(`${apiUrl}/api/holdings/securities`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						symbol: securityForm.symbol,
						isin: securityForm.isin || null,
						name: securityForm.name,
						asset_type: securityForm.asset_type,
						currency: securityForm.currency
					})
				});
				if (!res.ok) {
					const d = await res.json().catch(() => ({ detail: res.statusText }));
					throw new Error(d.detail ?? res.statusText);
				}
				await invalidateAll();
			},
			'Dodano papier wartościowy'
		);
	}

	async function saveLot() {
		const apiUrl = resolveApiUrl();
		await saveVia(
			lotCrud,
			async () => {
				const res = await fetch(`${apiUrl}/api/holdings/lots`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(lotForm)
				});
				if (!res.ok) {
					const d = await res.json().catch(() => ({ detail: res.statusText }));
					throw new Error(d.detail ?? res.statusText);
				}
				await invalidateAll();
			},
			'Dodano transakcję'
		);
	}

	async function saveQuote() {
		const apiUrl = resolveApiUrl();
		await saveVia(
			quoteCrud,
			async () => {
				const res = await fetch(
					`${apiUrl}/api/holdings/securities/${quoteForm.security_id}/quotes`,
					{
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ date: quoteForm.date, price: quoteForm.price })
					}
				);
				if (!res.ok) {
					const d = await res.json().catch(() => ({ detail: res.statusText }));
					throw new Error(d.detail ?? res.statusText);
				}
				await invalidateAll();
			},
			'Zapisano notowanie'
		);
	}

	async function deleteSecurity(s: { id: number; symbol: string }) {
		const ok = await confirm({
			title: 'Usunąć papier wartościowy?',
			message: `„${s.symbol}” zostanie trwale usunięty.`,
			danger: true,
			confirmText: 'Usuń'
		});
		if (!ok) return;
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/holdings/securities/${s.id}`, {
				method: 'DELETE'
			});
			if (!res.ok) {
				const d = await res.json().catch(() => ({ detail: res.statusText }));
				throw new Error(d.detail ?? res.statusText);
			}
			toast.success('Usunięto');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		}
	}

	function fmt(s: string): string {
		const n = Number(s);
		if (Number.isNaN(n)) return s;
		return n.toLocaleString('pl-PL', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	function fmtQty(s: string): string {
		const n = Number(s);
		if (Number.isNaN(n)) return s;
		return n.toLocaleString('pl-PL', { maximumFractionDigits: 6 });
	}

	// Aggregate paid + current + profit across positions whose PLN values
	// are known. Skipping a row without an FX rate (rather than summing its
	// native USD/EUR figure into a PLN total) keeps the tiles honest at
	// the cost of under-counting a partially-priced portfolio. The skipped
	// count is surfaced under the tiles so an under-count isn't silent.
	const pricedHoldings = $derived(
		data.holdings.filter((h) => h.cost_basis_pln !== null && h.market_value_pln !== null)
	);
	const skippedCount = $derived(data.holdings.length - pricedHoldings.length);
	const totalPaid = $derived(pricedHoldings.reduce((sum, h) => sum + Number(h.cost_basis_pln), 0));
	const totalValue = $derived(
		pricedHoldings.reduce((sum, h) => sum + Number(h.market_value_pln), 0)
	);
	const totalProfit = $derived(totalValue - totalPaid);
	const profitPct = $derived(totalPaid > 0 ? (totalProfit / totalPaid) * 100 : 0);
	const totalDividendsNet = $derived(
		data.dividends.reduce((sum, d) => sum + Number(d.net_amount), 0)
	);

	function fmtPLN(n: number): string {
		return (
			n.toLocaleString('pl-PL', { minimumFractionDigits: 0, maximumFractionDigits: 0 }) + ' zł'
		);
	}

	async function refreshQuotes() {
		refreshing = true;
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/holdings/refresh-quotes`, {
				method: 'POST'
			});
			if (!res.ok) {
				const d = await res.json().catch(() => ({ detail: res.statusText }));
				throw new Error(d.detail ?? res.statusText);
			}
			const body = (await res.json()) as {
				total: number;
				written: number;
				skipped_manual: number;
				failed: number;
			};
			toast.success(
				`Stooq: ${body.written}/${body.total} zaktualizowano` +
					(body.skipped_manual > 0 ? ` · ${body.skipped_manual} ręcznych pominięto` : '') +
					(body.failed > 0 ? ` · ${body.failed} błędów` : '')
			);
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		} finally {
			refreshing = false;
		}
	}
</script>

<svelte:head>
	<title>Inwestycje | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<h2 class="h3">Akcje / ETF</h2>
			<p class="text-sm text-surface-700-300">
				Pozycje skonsolidowane po tickerze: ilość, średnia cena, zysk zrealizowany i niezrealizowany
				na podstawie ręcznych notowań.
			</p>
		</div>
		<div class="flex flex-wrap gap-2">
			<button
				type="button"
				class="btn preset-tonal-surface"
				onclick={refreshQuotes}
				disabled={refreshing}
				title="Pobierz aktualne ceny z Stooq"
			>
				<RefreshCw size={16} class={refreshing ? 'animate-spin' : ''} />
				<span>{refreshing ? 'Aktualizuję…' : 'Aktualizuj ceny'}</span>
			</button>
			<button type="button" class="btn preset-tonal-surface" onclick={openSecurityModal}>
				<Plus size={16} /><span>Nowy papier</span>
			</button>
			<button type="button" class="btn preset-tonal-surface" onclick={openQuoteModal}>
				<BarChart size={16} /><span>Notowanie</span>
			</button>
			<button type="button" class="btn preset-tonal-surface" onclick={openDividendModal}>
				<Coins size={16} /><span>Dywidenda</span>
			</button>
			<button type="button" class="btn preset-filled-primary-500" onclick={openLotModal}>
				<Plus size={16} /><span>Nowa transakcja</span>
			</button>
		</div>
	</div>

	{#if data.holdings.length > 0}
		<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
			<div class="card preset-filled-surface-100-900 p-4 space-y-1">
				<header class="text-sm text-surface-700-300">Wpłacono</header>
				<div class="text-2xl font-bold">{fmtPLN(totalPaid)}</div>
			</div>
			<div class="card preset-filled-surface-100-900 p-4 space-y-1">
				<header class="text-sm text-surface-700-300">Wartość bieżąca</header>
				<div class="text-2xl font-bold text-primary-600-400">{fmtPLN(totalValue)}</div>
			</div>
			<div class="card preset-filled-surface-100-900 p-4 space-y-1">
				<header class="text-sm text-surface-700-300">Zysk</header>
				<div class="text-2xl font-bold {totalProfit >= 0 ? 'text-success-500' : 'text-error-500'}">
					{totalProfit >= 0 ? '+' : ''}{fmtPLN(totalProfit)}
				</div>
				<div class="text-xs text-surface-600-400">
					{totalProfit >= 0 ? '+' : ''}{profitPct.toFixed(2)}%
				</div>
			</div>
			<div class="card preset-filled-surface-100-900 p-4 space-y-1">
				<header class="text-sm text-surface-700-300">Pozycji</header>
				<div class="text-2xl font-bold">{data.holdings.length}</div>
				<div class="text-xs text-surface-600-400">
					{data.holdings.map((h) => h.security.symbol).join(', ')}
				</div>
			</div>
		</div>
		{#if skippedCount > 0}
			<p class="text-xs text-warning-500">
				Sumy pomijają {skippedCount}
				{skippedCount === 1 ? 'pozycję' : 'pozycje'} bez kursu PLN.
			</p>
		{/if}
	{/if}

	{#if data.holdings.length === 0}
		<div class="card preset-tonal-surface p-6 text-center text-sm text-surface-700-300">
			Brak otwartych pozycji. Dodaj papier wartościowy i pierwszą transakcję.
		</div>
	{:else}
		<div class="table-wrap">
			<table class="table table-hover text-sm">
				<thead>
					<tr>
						<th>Symbol / Konto</th>
						<th>Nazwa</th>
						<th class="text-right">Ilość</th>
						<th class="text-right">Średnia cena</th>
						<th class="text-right">Koszt nabycia</th>
						<th class="text-right">Aktualna cena</th>
						<th class="text-right">Wartość rynkowa</th>
						<th class="text-right">Zysk niezreal.</th>
						<th class="text-right">Zysk zreal.</th>
					</tr>
				</thead>
				<tbody>
					{#each data.holdings as h (h.security.id)}
						<tr class="bg-surface-100-900/40 font-semibold">
							<td>
								<div>{h.security.symbol}</div>
								{#if h.security.isin}
									<div class="text-xs font-normal text-surface-600-400">{h.security.isin}</div>
								{/if}
							</td>
							<td>{h.security.name}</td>
							<td class="text-right">{fmtQty(h.quantity)}</td>
							<td class="text-right">
								<div>{fmt(h.average_cost)} {h.security.currency}</div>
								{#if h.average_cost_pln && h.security.currency !== 'PLN'}
									<div class="text-xs font-normal text-surface-600-400">
										{fmt(h.average_cost_pln)} PLN
									</div>
								{/if}
							</td>
							<td class="text-right">
								<div>{fmt(h.cost_basis)} {h.security.currency}</div>
								{#if h.cost_basis_pln && h.security.currency !== 'PLN'}
									<div class="text-xs font-normal text-surface-600-400">
										{fmt(h.cost_basis_pln)} PLN
									</div>
								{/if}
							</td>
							<td class="text-right">
								{#if h.latest_quote}
									{fmt(h.latest_quote)}
									{h.security.currency}
									<div class="text-xs font-normal text-surface-600-400">
										{h.latest_quote_date}{#if h.latest_quote_rate_pln && h.security.currency !== 'PLN'}
											· {fmt(h.latest_quote_rate_pln)} PLN/{h.security.currency}{/if}
									</div>
								{:else}
									<span class="text-surface-600-400">—</span>
								{/if}
							</td>
							<td class="text-right">
								<div>{fmt(h.market_value)} {h.security.currency}</div>
								{#if h.market_value_pln && h.security.currency !== 'PLN'}
									<div class="text-xs font-normal text-surface-600-400">
										{fmt(h.market_value_pln)} PLN
									</div>
								{/if}
							</td>
							<td
								class="text-right {Number(h.unrealized_gain) >= 0
									? 'text-success-500'
									: 'text-error-500'}"
							>
								<div>{fmt(h.unrealized_gain)} {h.security.currency}</div>
								{#if h.unrealized_gain_pln && h.security.currency !== 'PLN'}
									<div
										class="text-xs font-normal {Number(h.unrealized_gain_pln) >= 0
											? 'text-success-500'
											: 'text-error-500'} opacity-80"
									>
										{fmt(h.unrealized_gain_pln)} PLN
									</div>
								{/if}
							</td>
							<td
								class="text-right {Number(h.realized_gain) >= 0
									? 'text-success-500'
									: 'text-error-500'}"
							>
								<div>{fmt(h.realized_gain)} {h.security.currency}</div>
								{#if h.realized_gain_pln && h.security.currency !== 'PLN'}
									<div
										class="text-xs font-normal {Number(h.realized_gain_pln) >= 0
											? 'text-success-500'
											: 'text-error-500'} opacity-80"
									>
										{fmt(h.realized_gain_pln)} PLN
									</div>
								{/if}
							</td>
						</tr>
						{#each h.accounts as a (a.account_id)}
							<tr class="text-surface-700-300">
								<td class="pl-6 text-xs">↳ {a.account_name}</td>
								<td></td>
								<td class="text-right">{fmtQty(a.quantity)}</td>
								<td class="text-right">
									<div>{fmt(a.average_cost)} {h.security.currency}</div>
									{#if a.average_cost_pln && h.security.currency !== 'PLN'}
										<div class="text-xs text-surface-600-400">{fmt(a.average_cost_pln)} PLN</div>
									{/if}
								</td>
								<td class="text-right">
									<div>{fmt(a.cost_basis)} {h.security.currency}</div>
									{#if a.cost_basis_pln && h.security.currency !== 'PLN'}
										<div class="text-xs text-surface-600-400">{fmt(a.cost_basis_pln)} PLN</div>
									{/if}
								</td>
								<td class="text-right text-surface-600-400">—</td>
								<td class="text-right">
									<div>{fmt(a.market_value)} {h.security.currency}</div>
									{#if a.market_value_pln && h.security.currency !== 'PLN'}
										<div class="text-xs text-surface-600-400">{fmt(a.market_value_pln)} PLN</div>
									{/if}
								</td>
								<td
									class="text-right {Number(a.unrealized_gain) >= 0
										? 'text-success-500'
										: 'text-error-500'}"
								>
									<div>{fmt(a.unrealized_gain)} {h.security.currency}</div>
									{#if a.unrealized_gain_pln && h.security.currency !== 'PLN'}
										<div
											class="text-xs {Number(a.unrealized_gain_pln) >= 0
												? 'text-success-500'
												: 'text-error-500'} opacity-80"
										>
											{fmt(a.unrealized_gain_pln)} PLN
										</div>
									{/if}
								</td>
								<td
									class="text-right {Number(a.realized_gain) >= 0
										? 'text-success-500'
										: 'text-error-500'}"
								>
									{fmt(a.realized_gain)}
									{h.security.currency}
								</td>
							</tr>
						{/each}
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<PIT38Report />

	<section class="card preset-filled-surface-100-900 p-4 space-y-3">
		<header class="flex items-center justify-between">
			<h2 class="h3">Papiery wartościowe</h2>
		</header>
		{#if data.securities.length === 0}
			<p class="text-sm text-surface-700-300">Brak papierów wartościowych.</p>
		{:else}
			<ul class="space-y-2">
				{#each data.securities as s (s.id)}
					<li class="flex items-center gap-3 px-3 py-2 rounded-container bg-surface-50-950 text-sm">
						<span class="font-semibold w-16">{s.symbol}</span>
						<span class="flex-1">{s.name}</span>
						<span class="text-xs text-surface-600-400">{s.asset_type} · {s.currency}</span>
						<button
							type="button"
							class="btn-icon btn-icon-sm"
							aria-label="Usuń {s.symbol}"
							onclick={() => deleteSecurity(s)}
						>
							<Trash2 size={14} />
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section class="card preset-filled-surface-100-900 p-4 space-y-3">
		<header class="flex flex-wrap items-center justify-between gap-2">
			<h2 class="h3 flex items-center gap-2"><Coins size={18} /> Dywidendy</h2>
			{#if data.dividends.length > 0}
				<span class="text-sm text-surface-700-300">
					Netto łącznie: <span class="font-semibold text-success-500"
						>{fmtPLN(totalDividendsNet)}</span
					>
				</span>
			{/if}
		</header>
		{#if data.dividends.length === 0}
			<p class="text-sm text-surface-700-300">
				Brak zarejestrowanych dywidend. Dodaj wpłatę dywidendy, aby uwzględnić ją w zwrotach (XIRR).
			</p>
		{:else}
			<div class="table-wrap">
				<table class="table table-hover text-sm">
					<thead>
						<tr>
							<th>Data wypłaty</th>
							<th>Papier</th>
							<th>Konto</th>
							<th class="text-right">Brutto</th>
							<th class="text-right">Podatek</th>
							<th class="text-right">Netto</th>
							<th></th>
						</tr>
					</thead>
					<tbody>
						{#each data.dividends as d (d.id)}
							<tr>
								<td>{d.pay_date}</td>
								<td class="font-semibold">{securityLabel(d.security_id)}</td>
								<td class="text-surface-700-300">{accountLabel(d.account_id)}</td>
								<td class="text-right">{fmt(d.gross_amount)} {d.currency}</td>
								<td class="text-right text-surface-600-400"
									>{fmt(d.withholding_tax)} {d.currency}</td
								>
								<td class="text-right font-semibold text-success-500"
									>{fmt(d.net_amount)} {d.currency}</td
								>
								<td class="text-right">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń dywidendę"
										onclick={() => deleteDividend(d.id)}
									>
										<Trash2 size={14} />
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</div>

<Modal
	open={securityCrud.open}
	title="Nowy papier wartościowy"
	onCancel={() => securityCrud.close()}
	onConfirm={saveSecurity}
	confirmDisabled={securityCrud.saving}
	confirmText={securityCrud.saving ? 'Zapisywanie...' : 'Zapisz'}
>
	<div class="flex flex-col gap-3">
		<label class="label">
			<span class="text-sm font-semibold">Ticker / Symbol</span>
			<input type="text" bind:value={securityForm.symbol} maxlength="32" class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">ISIN (opcjonalnie)</span>
			<input type="text" bind:value={securityForm.isin} maxlength="12" class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Nazwa</span>
			<input type="text" bind:value={securityForm.name} maxlength="200" class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Typ</span>
			<select bind:value={securityForm.asset_type} class="select">
				{#each ASSET_TYPES as t}
					<option value={t.value}>{t.label}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Waluta</span>
			<input type="text" bind:value={securityForm.currency} maxlength="3" class="input" />
		</label>
	</div>
</Modal>

<Modal
	open={lotCrud.open}
	title="Nowa transakcja"
	onCancel={() => lotCrud.close()}
	onConfirm={saveLot}
	confirmDisabled={lotCrud.saving}
	confirmText={lotCrud.saving ? 'Zapisywanie...' : 'Zapisz'}
>
	<div class="flex flex-col gap-3">
		<label class="label">
			<span class="text-sm font-semibold">Konto</span>
			<select bind:value={lotForm.account_id} class="select">
				{#each data.accounts as a}
					<option value={a.id}>{a.name}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Papier</span>
			<select bind:value={lotForm.security_id} class="select">
				{#each data.securities as s}
					<option value={s.id}>{s.symbol} — {s.name}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Strona</span>
			<select bind:value={lotForm.side} class="select">
				<option value="buy">Kupno</option>
				<option value="sell">Sprzedaż</option>
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Ilość</span>
			<input type="text" bind:value={lotForm.quantity} class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Cena za sztukę</span>
			<input type="text" bind:value={lotForm.price} class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Prowizja</span>
			<input type="text" bind:value={lotForm.fee} class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Data</span>
			<input type="date" bind:value={lotForm.date} class="input" />
		</label>
	</div>
</Modal>

<Modal
	open={quoteCrud.open}
	title="Nowe notowanie"
	onCancel={() => quoteCrud.close()}
	onConfirm={saveQuote}
	confirmDisabled={quoteCrud.saving}
	confirmText={quoteCrud.saving ? 'Zapisywanie...' : 'Zapisz'}
>
	<div class="flex flex-col gap-3">
		<label class="label">
			<span class="text-sm font-semibold">Papier</span>
			<select bind:value={quoteForm.security_id} class="select">
				{#each data.securities as s}
					<option value={s.id}>{s.symbol} — {s.name}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Data</span>
			<input type="date" bind:value={quoteForm.date} class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Cena</span>
			<input type="text" bind:value={quoteForm.price} class="input" />
		</label>
		<p class="text-xs text-surface-600-400">
			Notowania na (papier, datę) są unikalne — kolejne wpisy dla tej samej daty nadpiszą cenę.
		</p>
	</div>
</Modal>

<Modal
	open={dividendCrud.open}
	title="Nowa dywidenda"
	onCancel={() => dividendCrud.close()}
	onConfirm={saveDividend}
	confirmDisabled={dividendCrud.saving}
	confirmText={dividendCrud.saving ? 'Zapisywanie...' : 'Zapisz'}
>
	<div class="flex flex-col gap-3">
		<label class="label">
			<span class="text-sm font-semibold">Konto</span>
			<select bind:value={dividendForm.account_id} class="select">
				{#each data.accounts as a}
					<option value={a.id}>{a.name}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Papier</span>
			<select bind:value={dividendForm.security_id} class="select">
				{#each data.securities as s}
					<option value={s.id}>{s.symbol} — {s.name}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Data wypłaty</span>
			<input type="date" bind:value={dividendForm.pay_date} class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Kwota brutto</span>
			<input type="text" bind:value={dividendForm.gross_amount} class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Podatek u źródła</span>
			<input type="text" bind:value={dividendForm.withholding_tax} class="input" />
		</label>
		<p class="text-xs text-surface-600-400">
			Netto (brutto − podatek) liczy się jako dochód i podnosi zwrot money-weighted (XIRR).
		</p>
	</div>
</Modal>
