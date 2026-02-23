<script lang="ts">
	import {
		Card,
		CardHeader,
		CardTitle,
		CardContent,
		Modal,
		Table,
		formatPLN
	} from '@mskalski/home-ui';
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
		if (!confirm('Czy na pewno chcesz usunąć tę transakcję?')) {
			return;
		}

		try {
			const response = await fetch(
				`${apiUrl}/api/accounts/${accountId}/transactions/${transactionId}`,
				{
					method: 'DELETE'
				}
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

<div class="page-header">
	<div>
		<h1 class="page-title">Wszystkie transakcje</h1>
		<p class="page-description">Historia transakcji inwestycyjnych</p>
	</div>
	<div class="header-buttons">
		<button class="btn btn-secondary" on:click={openPPKGenerateModal}>🏦 Generuj wpłaty PPK</button>
		<button class="btn btn-primary" on:click={openNewTransactionModal}>+ Nowa Transakcja</button>
	</div>
</div>

<Card>
	<CardHeader>
		<CardTitle>🔍 Filtry</CardTitle>
	</CardHeader>
	<CardContent>
		<form class="filters-form" on:submit|preventDefault={applyFilters}>
			<div class="filters-row">
				<div class="form-group">
					<label for="filter-account">Konto</label>
					<select id="filter-account" bind:value={filterAccountId}>
						<option value="">Wszystkie</option>
						{#each data.accounts as account}
							<option value={account.id}>{account.name}</option>
						{/each}
					</select>
				</div>

				<div class="form-group">
					<label for="filter-owner">Właściciel</label>
					<select id="filter-owner" bind:value={filterOwner}>
						<option value="">Wszystkie</option>
						{#each personas as persona}
							<option value={persona.name}>{persona.name}</option>
						{/each}
					</select>
				</div>

				<div class="form-group">
					<label for="filter-date-from">Data od</label>
					<input type="date" id="filter-date-from" bind:value={filterDateFrom} />
				</div>

				<div class="form-group">
					<label for="filter-date-to">Data do</label>
					<input type="date" id="filter-date-to" bind:value={filterDateTo} />
				</div>
			</div>

			<div class="filters-actions">
				<button type="submit" class="btn btn-primary">Filtruj</button>
				<button type="button" class="btn btn-secondary" on:click={clearFilters}>
					Wyczyść filtry
				</button>
			</div>
		</form>
	</CardContent>
</Card>

<Card>
	<CardHeader>
		<div class="card-header-content">
			<CardTitle>📊 Historia transakcji</CardTitle>
			<p class="total-invested">
				Zainwestowano łącznie: <strong>{formatPLN(data.transactions.total_invested)}</strong>
			</p>
		</div>
	</CardHeader>
	<CardContent>
		{#if data.transactions.transactions.length === 0}
			<div class="empty-state">
				<p>Brak transakcji</p>
			</div>
		{:else}
			<Table
				headers={['Data zakupu', 'Konto', 'Właściciel', 'Kwota', 'Akcje']}
				mobileCardView
				class="transactions-table"
			>
				{#each data.transactions.transactions as transaction}
					<tr>
						<td data-label="Data zakupu"
							>{new Date(transaction.date).toLocaleDateString('pl-PL')}</td
						>
						<td data-label="Konto" class="name-cell">{transaction.account_name}</td>
						<td data-label="Właściciel">{transaction.owner}</td>
						<td data-label="Kwota" class="value-cell">{formatPLN(transaction.amount)}</td>
						<td data-label="Akcje" class="actions-cell">
							<button
								class="btn-icon tap-target"
								on:click={() => deleteTransaction(transaction.account_id, transaction.id)}
								title="Usuń"
							>
								🗑️
							</button>
						</td>
					</tr>
				{/each}
			</Table>
		{/if}
	</CardContent>
</Card>

<Modal
	open={showNewTransactionModal}
	title="Nowa Transakcja"
	onConfirm={createTransaction}
	onCancel={closeNewTransactionModal}
	confirmText={savingTransaction ? 'Zapisywanie...' : 'Dodaj transakcję'}
	confirmDisabled={savingTransaction}
	confirmVariant="primary"
>
	<form on:submit|preventDefault={createTransaction} class="transaction-form">
		{#if transactionError}
			<div class="error-message">{transactionError}</div>
		{/if}

		<div class="form-group">
			<label for="transaction-account">Konto *</label>
			<select id="transaction-account" bind:value={newTransactionData.account_id} required>
				<option value="">Wybierz konto</option>
				{#each data.accounts as account}
					<option value={account.id}>{account.name}</option>
				{/each}
			</select>
		</div>

		<div class="form-group">
			<label for="transaction-amount">Kwota (PLN) *</label>
			<input
				type="number"
				id="transaction-amount"
				bind:value={newTransactionData.amount}
				required
				min="0.01"
				step="0.01"
				placeholder="np. 5000.00"
			/>
		</div>

		<div class="form-group">
			<label for="transaction-date">Data zakupu *</label>
			<input
				type="date"
				id="transaction-date"
				bind:value={newTransactionData.date}
				required
				max={new Date().toISOString().split('T')[0]}
			/>
		</div>

		<div class="form-group">
			<label for="transaction-owner">Właściciel *</label>
			<select id="transaction-owner" bind:value={newTransactionData.owner} required>
				{#each personas as persona}
					<option value={persona.name}>{persona.name}</option>
				{/each}
			</select>
		</div>
	</form>
</Modal>

<Modal
	open={showPPKGenerateModal}
	title="Generuj wpłaty PPK"
	onConfirm={generatePPKContributions}
	onCancel={closePPKGenerateModal}
	confirmText={ppkGenerating ? 'Generowanie...' : 'Generuj'}
	confirmDisabled={ppkGenerating}
	confirmVariant="primary"
>
	<form on:submit|preventDefault={generatePPKContributions} class="transaction-form">
		{#if ppkGenerateError}
			<div class="error-message">{ppkGenerateError}</div>
		{/if}

		<div class="form-group">
			<label for="ppk-owner">Właściciel *</label>
			<select id="ppk-owner" bind:value={ppkGenerateData.owner} required>
				{#each personas as persona}
					<option value={persona.name}>{persona.name}</option>
				{/each}
			</select>
		</div>

		<div class="form-group">
			<label for="ppk-month">Miesiąc *</label>
			<select id="ppk-month" bind:value={ppkGenerateData.month} required>
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
		</div>

		<div class="form-group">
			<label for="ppk-year">Rok *</label>
			<input
				type="number"
				id="ppk-year"
				bind:value={ppkGenerateData.year}
				required
				min="2019"
				max={new Date().getFullYear() + 1}
			/>
		</div>
	</form>
</Modal>

<style>
	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: var(--size-6);
	}

	.page-title {
		font-size: var(--font-size-6);
		font-weight: var(--font-weight-7);
		color: var(--color-text);
		margin: 0 0 var(--size-2) 0;
	}

	.page-description {
		color: var(--color-text-secondary);
		font-size: var(--font-size-2);
		margin: 0;
	}

	.card-header-content {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
	}

	.total-invested {
		margin: 0;
		font-size: var(--font-size-3);
		color: var(--color-text-secondary);
	}

	.total-invested strong {
		color: var(--color-primary);
		font-weight: var(--font-weight-7);
	}

	.filters-form {
		display: flex;
		flex-direction: column;
		gap: var(--size-5);
	}

	.filters-row {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: var(--size-4);
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: var(--size-2);
	}

	.form-group label {
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.form-group input,
	.form-group select {
		padding: var(--size-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: var(--color-background);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.form-group input:focus,
	.form-group select:focus {
		outline: none;
		border-color: var(--color-primary);
	}

	.filters-actions {
		display: flex;
		gap: var(--size-3);
	}

	.btn {
		padding: var(--size-3) var(--size-5);
		border: none;
		border-radius: var(--radius-2);
		font-weight: var(--font-weight-6);
		font-size: var(--font-size-2);
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-primary {
		background: var(--color-primary);
		color: var(--nord6);
	}

	.btn-primary:hover {
		background: var(--nord9);
	}

	.btn-secondary {
		background: var(--color-surface);
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}

	.btn-secondary:hover {
		background: var(--color-accent);
	}

	.btn-icon {
		background: transparent;
		border: none;
		cursor: pointer;
		font-size: var(--font-size-3);
		padding: var(--size-2);
		transition: transform 0.2s;
	}

	.btn-icon:hover {
		transform: scale(1.2);
	}

	.empty-state {
		text-align: center;
		padding: var(--size-8) var(--size-4);
		color: var(--color-text-secondary);
	}

	.table-container {
		overflow-x: auto;
	}

	.transactions-table {
		width: 100%;
		border-collapse: collapse;
	}

	.transactions-table thead {
		border-bottom: 2px solid var(--color-border);
	}

	.transactions-table th {
		text-align: left;
		padding: var(--size-3) var(--size-4);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.transactions-table tbody tr {
		border-bottom: 1px solid var(--color-border);
		transition: background-color 0.2s;
	}

	.transactions-table tbody tr:hover {
		background-color: var(--color-accent);
	}

	.transactions-table td {
		padding: var(--size-4);
		font-size: var(--font-size-2);
	}

	.name-cell {
		font-weight: var(--font-weight-6);
		color: var(--color-text);
	}

	.value-cell {
		font-weight: var(--font-weight-6);
		color: var(--color-primary);
	}

	.actions-cell {
		text-align: right;
	}

	.transaction-form {
		display: flex;
		flex-direction: column;
		gap: var(--size-4);
	}

	.error-message {
		padding: var(--size-3);
		background: var(--nord11);
		color: var(--nord6);
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
	}

	.header-buttons {
		display: flex;
		gap: var(--size-3);
	}

	@media (max-width: 1024px) {
		.filters-row {
			grid-template-columns: repeat(2, 1fr);
		}
	}

	@media (max-width: 768px) {
		.filters-row {
			grid-template-columns: 1fr;
		}

		.card-header-content {
			flex-direction: column;
			align-items: flex-start;
			gap: var(--size-2);
		}
	}
</style>
