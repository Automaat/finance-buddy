<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { resolveApiUrl } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { formatDate } from '$lib/utils/format';
	import Modal from '$lib/components/Modal.svelte';
	import { Plus, Play, SkipForward, Trash2, Pencil } from 'lucide-svelte';
	import type { PageData } from './$types';
	import type { RecurringRow, AccountOption } from './+page';

	let { data }: { data: PageData } = $props();

	const FREQUENCIES = [
		{ value: 'daily', label: 'Codziennie' },
		{ value: 'weekly', label: 'Co tydzień' },
		{ value: 'monthly', label: 'Co miesiąc' },
		{ value: 'quarterly', label: 'Co kwartał' },
		{ value: 'yearly', label: 'Co rok' }
	];

	const accountsById = $derived(
		new Map<number, AccountOption>(data.accounts.map((a) => [a.id, a]))
	);

	let modalOpen = $state(false);
	let editingId = $state<number | null>(null);
	let form = $state(emptyForm());
	let saving = $state(false);

	function emptyForm() {
		const today = new Date().toISOString().slice(0, 10);
		return {
			account_id: data.accounts[0]?.id ?? 0,
			amount: '0.00',
			category: '',
			description: '',
			frequency: 'monthly',
			day_of_month: 1,
			start_date: today,
			end_date: '',
			active: true
		};
	}

	function openCreate() {
		if (data.accounts.length === 0) {
			toast.error('Najpierw utwórz konto, do którego będą trafiać transakcje.');
			return;
		}
		editingId = null;
		form = emptyForm();
		modalOpen = true;
	}

	function openEdit(row: RecurringRow) {
		editingId = row.id;
		form = {
			account_id: row.account_id,
			amount: row.amount,
			category: row.category ?? '',
			description: row.description,
			frequency: row.frequency,
			day_of_month: row.day_of_month ?? 1,
			start_date: row.start_date,
			end_date: row.end_date ?? '',
			active: row.active
		};
		modalOpen = true;
	}

	async function save() {
		saving = true;
		try {
			const apiUrl = resolveApiUrl();
			const body = {
				account_id: form.account_id,
				amount: form.amount,
				category: form.category || null,
				description: form.description,
				frequency: form.frequency,
				day_of_month: form.day_of_month,
				start_date: form.start_date,
				end_date: form.end_date || null,
				active: form.active
			};
			const url =
				editingId === null ? `${apiUrl}/api/recurring` : `${apiUrl}/api/recurring/${editingId}`;
			const method = editingId === null ? 'POST' : 'PUT';
			const res = await fetch(url, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});
			if (!res.ok) {
				const detail = await res.json().catch(() => ({ detail: res.statusText }));
				throw new Error(detail.detail ?? res.statusText);
			}
			modalOpen = false;
			toast.success(editingId === null ? 'Utworzono' : 'Zapisano');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		} finally {
			saving = false;
		}
	}

	async function runNow(row: RecurringRow) {
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/recurring/${row.id}/run-now`, { method: 'POST' });
			if (!res.ok) throw new Error('Nie udało się utworzyć transakcji');
			const payload = (await res.json().catch(() => null)) as { already_minted?: boolean } | null;
			const label = row.description || `recurring #${row.id}`;
			if (payload?.already_minted) {
				toast.info(`„${label}" już utworzono dziś — pominięto.`);
			} else {
				toast.success(`Utworzono transakcję na dziś dla „${label}".`);
			}
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		}
	}

	async function skipNext(row: RecurringRow) {
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/recurring/${row.id}/skip`, { method: 'POST' });
			if (!res.ok) throw new Error('Nie udało się pominąć następnego wystąpienia');
			toast.success('Pominięto następne wystąpienie');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		}
	}

	async function unskip(row: RecurringRow, date: string) {
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(
				`${apiUrl}/api/recurring/${row.id}/unskip?date=${encodeURIComponent(date)}`,
				{ method: 'POST' }
			);
			if (!res.ok) throw new Error('Nie udało się przywrócić wystąpienia');
			toast.success('Przywrócono wystąpienie');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		}
	}

	async function remove(row: RecurringRow) {
		const label = row.description || `recurring #${row.id}`;
		const ok = await confirm({
			title: 'Usunąć transakcję cykliczną?',
			message: `„${label}" zostanie trwale usunięta.`,
			danger: true,
			confirmText: 'Usuń'
		});
		if (!ok) return;
		try {
			const apiUrl = resolveApiUrl();
			const res = await fetch(`${apiUrl}/api/recurring/${row.id}`, { method: 'DELETE' });
			if (!res.ok) throw new Error('Nie udało się usunąć');
			toast.success('Usunięto');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) toast.error(err.message);
		}
	}

	function frequencyLabel(freq: string): string {
		return FREQUENCIES.find((f) => f.value === freq)?.label ?? freq;
	}
</script>

<svelte:head>
	<title>Transakcje cykliczne | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="h1">Transakcje cykliczne</h1>
			<p class="text-sm text-surface-700-300">
				Szablony, z których planowo generowane są transakcje (wypłata, rata, subskrypcje).
			</p>
		</div>
		<button type="button" class="btn preset-filled-primary-500" onclick={openCreate}>
			<Plus size={16} />
			<span>Dodaj</span>
		</button>
	</div>

	{#if data.recurring.length === 0}
		<div class="card preset-tonal-surface p-6 text-center text-sm text-surface-700-300">
			Brak transakcji cyklicznych. Kliknij „Dodaj” aby zacząć.
		</div>
	{:else}
		<div class="table-wrap">
			<table class="table table-hover text-sm">
				<thead>
					<tr>
						<th>Opis</th>
						<th>Konto</th>
						<th class="text-right">Kwota</th>
						<th>Częstotliwość</th>
						<th>Następne</th>
						<th>Status</th>
						<th class="text-right">Akcje</th>
					</tr>
				</thead>
				<tbody>
					{#each data.recurring as row (row.id)}
						<tr>
							<td>
								<div>{row.description || '—'}</div>
								{#if row.category}
									<div class="text-xs text-surface-600-400">{row.category}</div>
								{/if}
							</td>
							<td>{accountsById.get(row.account_id)?.name ?? '—'}</td>
							<td class="text-right">{row.amount} PLN</td>
							<td>{frequencyLabel(row.frequency)}</td>
							<td>{row.next_occurrence ?? '—'}</td>
							<td>
								{#if row.active}
									<span class="badge preset-tonal-success">Aktywna</span>
								{:else}
									<span class="badge preset-tonal-surface">Wstrzymana</span>
								{/if}
							</td>
							<td class="text-right">
								<div class="flex justify-end gap-1">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Uruchom teraz"
										title="Uruchom teraz"
										onclick={() => runNow(row)}
									>
										<Play size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Pomiń następne"
										title="Pomiń następne"
										disabled={!row.next_occurrence}
										onclick={() => skipNext(row)}
									>
										<SkipForward size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										title="Edytuj"
										onclick={() => openEdit(row)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										title="Usuń"
										onclick={() => remove(row)}
									>
										<Trash2 size={16} />
									</button>
								</div>
							</td>
						</tr>
						{#if row.skipped_dates.length > 0}
							<tr>
								<td colspan="7" class="text-xs">
									<div class="flex flex-wrap items-center gap-2 px-2 py-1">
										<span class="text-surface-600-400">Pominięte:</span>
										{#each row.skipped_dates as d}
											<button
												type="button"
												class="badge preset-tonal-warning cursor-pointer"
												title="Przywróć {formatDate(d)}"
												onclick={() => unskip(row, d)}
											>
												{formatDate(d)} ×
											</button>
										{/each}
									</div>
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<Modal
	open={modalOpen}
	title={editingId === null ? 'Nowa transakcja cykliczna' : 'Edytuj transakcję cykliczną'}
	onCancel={() => (modalOpen = false)}
	onConfirm={save}
	confirmDisabled={saving}
	confirmText={saving ? 'Zapisywanie...' : 'Zapisz'}
>
	<div class="flex flex-col gap-3">
		<label class="label">
			<span class="text-sm font-semibold">Konto</span>
			<select bind:value={form.account_id} class="select">
				{#each data.accounts as acc}
					<option value={acc.id}>{acc.name}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Opis</span>
			<input type="text" bind:value={form.description} maxlength="200" class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Kategoria (opcjonalnie)</span>
			<input type="text" bind:value={form.category} maxlength="50" class="input" />
			<span class="text-xs text-surface-600-400"> np. wynagrodzenie, hipoteka, subskrypcja </span>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Kwota (PLN)</span>
			<input type="text" bind:value={form.amount} class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Częstotliwość</span>
			<select bind:value={form.frequency} class="select">
				{#each FREQUENCIES as freq}
					<option value={freq.value}>{freq.label}</option>
				{/each}
			</select>
		</label>
		{#if form.frequency === 'monthly' || form.frequency === 'quarterly' || form.frequency === 'yearly'}
			<label class="label">
				<span class="text-sm font-semibold">Dzień miesiąca</span>
				<input type="number" bind:value={form.day_of_month} min="1" max="31" class="input" />
				<span class="text-xs text-surface-600-400">
					Jeśli miesiąc krótszy, użyje ostatniego dnia.
				</span>
			</label>
		{/if}
		<label class="label">
			<span class="text-sm font-semibold">Data początkowa</span>
			<input type="date" bind:value={form.start_date} class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Data końcowa (opcjonalnie)</span>
			<input type="date" bind:value={form.end_date} class="input" />
		</label>
		<label class="flex items-center gap-2 cursor-pointer">
			<input type="checkbox" bind:checked={form.active} class="checkbox" />
			<span class="text-sm">Aktywna</span>
		</label>
	</div>
</Modal>
