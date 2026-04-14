<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { Plus, Landmark, Search, BarChart3, Trash2 } from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { goto, invalidateAll } from '$app/navigation';
	import type { Persona } from '$lib/types/personas';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
	$: personas = data.personas as Persona[];
	$: defaultOwner = personas.length > 0 ? personas[0].name : 'Marcin';

	let filterAccountId = data.filters.account_id || '';
	let filterOwner = data.filters.owner || '';
	let filterDateFrom = data.filters.date_from || '';
	let filterDateTo = data.filters.date_to || '';

	let showNewTransactionModal = false;
	let newTransactionData = {
		account_id: '',
		amount: 0,
		date: new Date().toISOString().split('T')[0],
		owner: defaultOwner
	};
	let transactionError = '';
	let savingTransaction = false;

	let showPPKGenerateModal = false;
	let ppkGenerateData = {
		owner: defaultOwner,
		month: new Date().getMonth() + 1,
		year: new Date().getFullYear()
	};
	let ppkGenerateError = '';
	let ppkGenerating = false;

	function applyFilters() {
		const params = new URLSearchParams();
		if (filterAccountId) params.set('account_id', filterAccountId);
		if (filterOwner) params.set('owner', filterOwner);
		if (filterDateFrom) params.set('date_from', filterDateFrom);
		if (filterDateTo) params.set('date_to', filterDateTo);

		goto(`/transactions?${params.toString()}`);
	}

	function clearFilters() {
		filterAccountId = '';
		filterOwner = '';
		filterDateFrom = '';
		filterDateTo = '';
		goto('/transactions');
	}

	async function deleteTransaction(accountId: number, transactionId: number) {
		if (!confirm('Czy na pewno chcesz usunąć tę transakcję?')) return;

		try {
			const response = await fetch(
				`${apiUrl}/api/accounts/${accountId}/transactions/${transactionId}`,
				{ method: 'DELETE' }
			);

			if (!response.ok) {
				throw new Error('Failed to delete transaction');
			}

			await invalidateAll();
		} catch (err) {
			console.error('Failed to delete transaction:', err);
			alert('Nie udało się usunąć transakcji');
		}
	}

	function openNewTransactionModal() {
		newTransactionData = {
			account_id: '',
			amount: 0,
			date: new Date().toISOString().split('T')[0],
			owner: defaultOwner
		};
		transactionError = '';
		showNewTransactionModal = true;
	}

	function closeNewTransactionModal() {
		showNewTransactionModal = false;
		transactionError = '';
	}

	function openPPKGenerateModal() {
		ppkGenerateData = {
			owner: defaultOwner,
			month: new Date().getMonth() + 1,
			year: new Date().getFullYear()
		};
		ppkGenerateError = '';
		showPPKGenerateModal = true;
	}

	function closePPKGenerateModal() {
		showPPKGenerateModal = false;
		ppkGenerateError = '';
	}

	async function generatePPKContributions() {
		ppkGenerateError = '';
		ppkGenerating = true;

		try {
			const response = await fetch(`${apiUrl}/api/retirement/ppk-contributions/generate`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(ppkGenerateData)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.detail || 'Nie udało się wygenerować wpłat PPK');
			}

			await invalidateAll();
			closePPKGenerateModal();
		} catch (err) {
			if (err instanceof Error) {
				ppkGenerateError = err.message;
			}
		} finally {
			ppkGenerating = false;
		}
	}

	async function createTransaction() {
		if (!newTransactionData.account_id) {
			transactionError = 'Wybierz konto';
			return;
		}

		transactionError = '';
		savingTransaction = true;

		try {
			const response = await fetch(
				`${apiUrl}/api/accounts/${newTransactionData.account_id}/transactions`,
				{
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						amount: newTransactionData.amount,
						date: newTransactionData.date,
						owner: newTransactionData.owner
					})
				}
			);

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.detail || 'Failed to create transaction');
			}

			await invalidateAll();
			closeNewTransactionModal();
		} catch (err) {
			if (err instanceof Error) {
				transactionError = err.message;
			}
		} finally {
			savingTransaction = false;
		}
	}
</script>

<svelte:head>
	<title>Transakcje | Finansowa Forteca</title>
</svelte:head>

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Wszystkie transakcje</h1>
		<p class="text-surface-700-300 text-sm">Historia transakcji inwestycyjnych</p>
	</div>
	<div class="flex flex-col sm:flex-row gap-2 w-full sm:w-auto">
		<button type="button" class="btn preset-tonal-surface gap-2" on:click={openPPKGenerateModal}>
			<Landmark size={16} />
			Generuj wpłaty PPK
		</button>
		<button type="button" class="btn preset-filled-primary-500 gap-2" on:click={openNewTransactionModal}>
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
		<form class="space-y-4" on:submit|preventDefault={applyFilters}>
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
					<select class="select" bind:value={filterOwner}>
						<option value="">Wszystkie</option>
						{#each personas as persona}
							<option value={persona.name}>{persona.name}</option>
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
				<button type="button" class="btn preset-tonal-surface" on:click={clearFilters}
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
			<div class="table-wrap">
				<table class="table table-hover">
					<thead>
						<tr>
							<th>Data zakupu</th>
							<th>Konto</th>
							<th>Właściciel</th>
							<th>Kwota</th>
							<th class="text-right">Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.transactions.transactions as transaction}
							<tr>
								<td>{new Date(transaction.date).toLocaleDateString('pl-PL')}</td>
								<td class="font-medium">{transaction.account_name}</td>
								<td>{transaction.owner}</td>
								<td class="font-semibold text-primary-600-400">{formatPLN(transaction.amount)}</td>
								<td class="text-right">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										on:click={() => deleteTransaction(transaction.account_id, transaction.id)}
									>
										<Trash2 size={16} />
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>

<Modal
	open={showNewTransactionModal}
	title="Nowa Transakcja"
	onConfirm={createTransaction}
	onCancel={closeNewTransactionModal}
	confirmText={savingTransaction ? 'Zapisywanie...' : 'Dodaj transakcję'}
	confirmDisabled={savingTransaction}
>
	<form on:submit|preventDefault={createTransaction} class="space-y-4">
		{#if transactionError}
			<div class="card preset-filled-error-500 p-3 text-sm">{transactionError}</div>
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
			<select class="select" bind:value={newTransactionData.owner} required>
				{#each personas as persona}
					<option value={persona.name}>{persona.name}</option>
				{/each}
			</select>
		</label>
	</form>
</Modal>

<Modal
	open={showPPKGenerateModal}
	title="Generuj wpłaty PPK"
	onConfirm={generatePPKContributions}
	onCancel={closePPKGenerateModal}
	confirmText={ppkGenerating ? 'Generowanie...' : 'Generuj'}
	confirmDisabled={ppkGenerating}
>
	<form on:submit|preventDefault={generatePPKContributions} class="space-y-4">
		{#if ppkGenerateError}
			<div class="card preset-filled-error-500 p-3 text-sm">{ppkGenerateError}</div>
		{/if}

		<label class="label">
			<span class="font-semibold text-sm">Właściciel *</span>
			<select class="select" bind:value={ppkGenerateData.owner} required>
				{#each personas as persona}
					<option value={persona.name}>{persona.name}</option>
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
