<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import SortableTable, { type SortableColumn } from '$lib/components/SortableTable.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { Plus, Landmark, Search, BarChart3, Trash2 } from 'lucide-svelte';
	import { api } from '$lib/apiClient';
	import { goto, invalidateAll } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { CrudForm } from '$lib/stores/crudForm.svelte';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import { untrack } from 'svelte';
	import type { Transaction } from '$lib/types/transactions';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const owners = $derived(data.owners as OwnerOption[]);
	const defaultOwnerUserId = $derived(owners.length > 0 ? owners[0].id : null);
	const defaultOwnerName = $derived(owners.length > 0 ? owners[0].name : '');

	const transactionColumns = $derived<SortableColumn<Transaction>[]>([
		{ key: 'date', label: 'Data zakupu', sortable: true, accessor: (t) => new Date(t.date) },
		{ key: 'account', label: 'Konto', sortable: true, accessor: (t) => t.account_name },
		{
			key: 'owner',
			label: 'Właściciel',
			sortable: true,
			accessor: (t) => ownerName(owners, t.owner_user_id)
		},
		{ key: 'amount', label: 'Kwota', sortable: true, accessor: (t) => t.amount },
		{ key: 'actions', label: 'Akcje', align: 'right' }
	]);

	let filterAccountId = $state(untrack(() => data.filters.account_id || ''));
	let filterOwnerUserId = $state(untrack(() => data.filters.owner_user_id || ''));
	let filterDateFrom = $state(untrack(() => data.filters.date_from || ''));
	let filterDateTo = $state(untrack(() => data.filters.date_to || ''));

	const txForm = new CrudForm();
	let newTransactionData = $state({
		account_id: '',
		amount: 0,
		date: new Date().toISOString().split('T')[0],
		owner_user_id: untrack(() => defaultOwnerUserId)
	});

	const ppkForm = new CrudForm();
	let ppkGenerateData = $state({
		owner: untrack(() => defaultOwnerName),
		month: new Date().getMonth() + 1,
		year: new Date().getFullYear()
	});

	function applyFilters() {
		const params = new URLSearchParams();
		if (filterAccountId) params.set('account_id', filterAccountId);
		if (filterOwnerUserId) params.set('owner_user_id', filterOwnerUserId);
		if (filterDateFrom) params.set('date_from', filterDateFrom);
		if (filterDateTo) params.set('date_to', filterDateTo);

		goto(`/transactions?${params.toString()}`);
	}

	function clearFilters() {
		filterAccountId = '';
		filterOwnerUserId = '';
		filterDateFrom = '';
		filterDateTo = '';
		goto('/transactions');
	}

	async function deleteTransaction(accountId: number, transactionId: number) {
		const ok = await confirm({
			title: 'Usuń transakcję',
			message: 'Czy na pewno chcesz usunąć tę transakcję?',
			confirmText: 'Usuń',
			danger: true
		});
		if (!ok) return;

		try {
			await api.del(`/api/accounts/${accountId}/transactions/${transactionId}`);
			await invalidateAll();
		} catch (err) {
			console.error('Failed to delete transaction:', err);
			toast.error('Nie udało się usunąć transakcji');
		}
	}

	function openNewTransactionModal() {
		newTransactionData = {
			account_id: '',
			amount: 0,
			date: new Date().toISOString().split('T')[0],
			owner_user_id: defaultOwnerUserId
		};
		txForm.openCreate();
	}

	function closeNewTransactionModal() {
		txForm.close();
	}

	function openPPKGenerateModal() {
		ppkGenerateData = {
			owner: defaultOwnerName,
			month: new Date().getMonth() + 1,
			year: new Date().getFullYear()
		};
		ppkForm.openCreate();
	}

	function closePPKGenerateModal() {
		ppkForm.close();
	}

	async function generatePPKContributions() {
		if (!ppkGenerateData.owner) {
			ppkForm.error = 'Wybierz właściciela';
			return;
		}
		await ppkForm.submit(async () => {
			await api.post('/api/retirement/ppk-contributions/generate', ppkGenerateData);
			await invalidateAll();
		});
	}

	async function createTransaction() {
		if (!newTransactionData.account_id) {
			txForm.error = 'Wybierz konto';
			return;
		}
		await txForm.submit(async () => {
			await api.post(`/api/accounts/${newTransactionData.account_id}/transactions`, {
				amount: newTransactionData.amount,
				date: newTransactionData.date,
				owner_user_id: newTransactionData.owner_user_id
			});
			await invalidateAll();
		});
	}
</script>

<svelte:head>
	<title>Transakcje | Finansowa Forteca</title>
</svelte:head>

{#snippet transactionRow(transaction: Transaction)}
	<tr>
		<td data-label="Data zakupu">{new Date(transaction.date).toLocaleDateString('pl-PL')}</td>
		<td class="font-medium" data-label="Konto">{transaction.account_name}</td>
		<td data-label="Właściciel">{ownerName(owners, transaction.owner_user_id)}</td>
		<td class="font-semibold text-primary-600-400" data-label="Kwota"
			>{formatPLN(transaction.amount)}</td
		>
		<td class="text-right">
			<button
				type="button"
				class="btn-icon btn-icon-sm"
				aria-label="Usuń"
				onclick={() => deleteTransaction(transaction.account_id, transaction.id)}
			>
				<Trash2 size={16} />
			</button>
		</td>
	</tr>
{/snippet}

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Transakcje</h1>
		<p class="text-surface-700-300 text-sm">Historia transakcji inwestycyjnych</p>
	</div>
	<div class="flex flex-col sm:flex-row gap-2 w-full sm:w-auto">
		<button type="button" class="btn preset-tonal-surface gap-2" onclick={openPPKGenerateModal}>
			<Landmark size={16} />
			Generuj wpłaty PPK
		</button>
		<button
			type="button"
			class="btn preset-filled-primary-500 gap-2"
			onclick={openNewTransactionModal}
		>
			<Plus size={16} />
			Nowa Transakcja
		</button>
	</div>
</div>

<div class="space-y-4">
	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Search size={20} /> Filtry</h3>
		</header>
		<form
			class="space-y-4"
			onsubmit={(event) => {
				event.preventDefault();
				applyFilters();
			}}
		>
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
				<label class="label">
					<span class="font-semibold text-sm">Konto</span>
					<select class="select" bind:value={filterAccountId}>
						<option value="">Wszystkie</option>
						{#each data.accounts as account}
							<option value={account.id}>{account.name}</option>
						{/each}
					</select>
				</label>

				<label class="label">
					<span class="font-semibold text-sm">Właściciel</span>
					<select class="select" bind:value={filterOwnerUserId}>
						<option value="">Wszystkie</option>
						{#each owners as owner}
							<option value={String(owner.id)}>{owner.name}</option>
						{/each}
					</select>
				</label>

				<label class="label">
					<span class="font-semibold text-sm">Data od</span>
					<input type="date" class="input" bind:value={filterDateFrom} />
				</label>

				<label class="label">
					<span class="font-semibold text-sm">Data do</span>
					<input type="date" class="input" bind:value={filterDateTo} />
				</label>
			</div>

			<div class="flex flex-col sm:flex-row gap-2">
				<button type="submit" class="btn preset-filled-primary-500">Filtruj</button>
				<button type="button" class="btn preset-tonal-surface" onclick={clearFilters}
					>Wyczyść filtry</button
				>
			</div>
		</form>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="flex flex-col md:flex-row md:items-center md:justify-between gap-2">
			<h3 class="h3 flex items-center gap-2"><BarChart3 size={20} /> Historia transakcji</h3>
			<p class="text-sm text-surface-700-300">
				Zainwestowano łącznie: <strong class="text-primary-600-400 font-bold"
					>{formatPLN(data.transactions.total_invested)}</strong
				>
			</p>
		</header>

		{#if data.transactions.transactions.length === 0}
			<div class="text-center py-12 text-surface-700-300">
				<p>Brak transakcji</p>
			</div>
		{:else}
			<div class="table-cards">
				<SortableTable
					columns={transactionColumns}
					items={data.transactions.transactions}
					row={transactionRow}
					getKey={(t) => t.id}
				/>
			</div>
		{/if}
	</div>
</div>

<Modal
	open={txForm.open}
	title="Nowa Transakcja"
	onConfirm={createTransaction}
	onCancel={closeNewTransactionModal}
	confirmText={txForm.saving ? 'Zapisywanie...' : 'Dodaj transakcję'}
	confirmDisabled={txForm.saving}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			createTransaction();
		}}
		class="space-y-4"
	>
		{#if txForm.error}
			<div class="card preset-filled-error-500 p-3 text-sm">{txForm.error}</div>
		{/if}

		<label class="label">
			<span class="font-semibold text-sm">Konto *</span>
			<select class="select" bind:value={newTransactionData.account_id} required>
				<option value="">Wybierz konto</option>
				{#each data.accounts as account}
					<option value={account.id}>{account.name}</option>
				{/each}
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Kwota (PLN) *</span>
			<input
				type="number"
				class="input"
				bind:value={newTransactionData.amount}
				required
				min="0.01"
				step="0.01"
				placeholder="np. 5000.00"
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Data zakupu *</span>
			<input
				type="date"
				class="input"
				bind:value={newTransactionData.date}
				required
				max={new Date().toISOString().split('T')[0]}
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Właściciel *</span>
			<select class="select" bind:value={newTransactionData.owner_user_id}>
				<option value={null}>Wspólne</option>
				{#each owners as owner}
					<option value={owner.id}>{owner.name}</option>
				{/each}
			</select>
		</label>
	</form>
</Modal>

<Modal
	open={ppkForm.open}
	title="Generuj wpłaty PPK"
	onConfirm={generatePPKContributions}
	onCancel={closePPKGenerateModal}
	confirmText={ppkForm.saving ? 'Generowanie...' : 'Generuj'}
	confirmDisabled={ppkForm.saving}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			generatePPKContributions();
		}}
		class="space-y-4"
	>
		{#if ppkForm.error}
			<div class="card preset-filled-error-500 p-3 text-sm">{ppkForm.error}</div>
		{/if}

		<label class="label">
			<span class="font-semibold text-sm">Właściciel *</span>
			<select class="select" bind:value={ppkGenerateData.owner} required>
				{#each owners as owner}
					<option value={owner.name}>{owner.name}</option>
				{/each}
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Miesiąc *</span>
			<select class="select" bind:value={ppkGenerateData.month} required>
				<option value={1}>Styczeń</option>
				<option value={2}>Luty</option>
				<option value={3}>Marzec</option>
				<option value={4}>Kwiecień</option>
				<option value={5}>Maj</option>
				<option value={6}>Czerwiec</option>
				<option value={7}>Lipiec</option>
				<option value={8}>Sierpień</option>
				<option value={9}>Wrzesień</option>
				<option value={10}>Październik</option>
				<option value={11}>Listopad</option>
				<option value={12}>Grudzień</option>
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Rok *</span>
			<input
				type="number"
				class="input"
				bind:value={ppkGenerateData.year}
				required
				min="2019"
				max={new Date().getFullYear() + 1}
			/>
		</label>
	</form>
</Modal>
