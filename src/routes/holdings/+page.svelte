<script lang="ts">
	import { untrack } from 'svelte';
	import { invalidateAll } from '$app/navigation';
	import { resolveApiUrl } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import { Plus, Trash2, BarChart } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const ASSET_TYPES = [
		{ value: 'stock', label: 'Akcja' },
		{ value: 'etf', label: 'ETF' },
		{ value: 'bond', label: 'Obligacja' },
		{ value: 'fund', label: 'Fundusz' }
	];

	let securityModalOpen = $state(false);
	let lotModalOpen = $state(false);
	let quoteModalOpen = $state(false);
	let saving = $state(false);

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

	function openSecurityModal() {
		securityForm = { symbol: '', isin: '', name: '', asset_type: 'stock', currency: 'PLN' };
		securityModalOpen = true;
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
		lotModalOpen = true;
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
		quoteModalOpen = true;
	}

	async function saveSecurity() {
		saving = true;
		try {
			const apiUrl = resolveApiUrl();
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
			securityModalOpen = false;
			toast.success('Dodano papier wartościowy');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		} finally {
			saving = false;
		}
	}

	async function saveLot() {
		saving = true;
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/holdings/lots`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(lotForm)
			});
			if (!res.ok) {
				const d = await res.json().catch(() => ({ detail: res.statusText }));
				throw new Error(d.detail ?? res.statusText);
			}
			lotModalOpen = false;
			toast.success('Dodano transakcję');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		} finally {
			saving = false;
		}
	}

	async function saveQuote() {
		saving = true;
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/holdings/securities/${quoteForm.security_id}/quotes`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ date: quoteForm.date, price: quoteForm.price })
			});
			if (!res.ok) {
				const d = await res.json().catch(() => ({ detail: res.statusText }));
				throw new Error(d.detail ?? res.statusText);
			}
			quoteModalOpen = false;
			toast.success('Zapisano notowanie');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		} finally {
			saving = false;
		}
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
</script>

<svelte:head>
	<title>Holdings | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<h1 class="h1">Holdings</h1>
			<p class="text-sm text-surface-700-300">
				Pozycje skonsolidowane po tickerze: ilość, średnia cena, zysk zrealizowany i niezrealizowany
				na podstawie ręcznych notowań.
			</p>
		</div>
		<div class="flex gap-2">
			<button type="button" class="btn preset-tonal-surface" onclick={openSecurityModal}>
				<Plus size={16} /><span>Nowy papier</span>
			</button>
			<button type="button" class="btn preset-tonal-surface" onclick={openQuoteModal}>
				<BarChart size={16} /><span>Notowanie</span>
			</button>
			<button type="button" class="btn preset-filled-primary-500" onclick={openLotModal}>
				<Plus size={16} /><span>Nowa transakcja</span>
			</button>
		</div>
	</div>

	{#if data.holdings.length === 0}
		<div class="card preset-tonal-surface p-6 text-center text-sm text-surface-700-300">
			Brak otwartych pozycji. Dodaj papier wartościowy i pierwszą transakcję.
		</div>
	{:else}
		<div class="table-wrap">
			<table class="table table-hover text-sm">
				<thead>
					<tr>
						<th>Symbol</th>
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
						<tr>
							<td>
								<div class="font-semibold">{h.security.symbol}</div>
								{#if h.security.isin}
									<div class="text-xs text-surface-600-400">{h.security.isin}</div>
								{/if}
							</td>
							<td>{h.security.name}</td>
							<td class="text-right">{fmtQty(h.quantity)}</td>
							<td class="text-right">{fmt(h.average_cost)} {h.security.currency}</td>
							<td class="text-right">{fmt(h.cost_basis)} {h.security.currency}</td>
							<td class="text-right">
								{#if h.latest_quote}
									{fmt(h.latest_quote)}
									{h.security.currency}
									<div class="text-xs text-surface-600-400">{h.latest_quote_date}</div>
								{:else}
									<span class="text-surface-600-400">—</span>
								{/if}
							</td>
							<td class="text-right">{fmt(h.market_value)} {h.security.currency}</td>
							<td
								class="text-right {Number(h.unrealized_gain) >= 0
									? 'text-success-500'
									: 'text-error-500'}"
							>
								{fmt(h.unrealized_gain)}
							</td>
							<td
								class="text-right {Number(h.realized_gain) >= 0
									? 'text-success-500'
									: 'text-error-500'}"
							>
								{fmt(h.realized_gain)}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

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
</div>

<Modal
	open={securityModalOpen}
	title="Nowy papier wartościowy"
	onCancel={() => (securityModalOpen = false)}
	onConfirm={saveSecurity}
	confirmDisabled={saving}
	confirmText={saving ? 'Zapisywanie...' : 'Zapisz'}
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
	open={lotModalOpen}
	title="Nowa transakcja"
	onCancel={() => (lotModalOpen = false)}
	onConfirm={saveLot}
	confirmDisabled={saving}
	confirmText={saving ? 'Zapisywanie...' : 'Zapisz'}
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
	open={quoteModalOpen}
	title="Nowe notowanie"
	onCancel={() => (quoteModalOpen = false)}
	onConfirm={saveQuote}
	confirmDisabled={saving}
	confirmText={saving ? 'Zapisywanie...' : 'Zapisz'}
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
