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
	import { invalidateAll } from '$app/navigation';
	import { onMount } from 'svelte';
	import { INVESTMENT_CATEGORIES } from '$lib/constants';
	import type { Account, Transaction, TransactionsData } from './+page';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
	const defaultOwner = env.PUBLIC_DEFAULT_OWNER || 'Marcin';

	let showForm = false;
	let editingAccount: Account | null = null;
	let showDeleteModal = false;
	let accountToDelete: number | null = null;
	let transactionCounts: Record<number, number> = {};

	let showTransactionsModal = false;
	let selectedAccountId: number | null = null;
	let selectedAccountName = '';
	let selectedAccountWrapper: string | null = null;
	let transactionsData: TransactionsData | null = null;
	let transactionFormData = {
		amount: 0,
		date: new Date().toISOString().split('T')[0],
		owner: defaultOwner,
		transaction_type: null as string | null
	};
	let transactionError = '';
	let savingTransaction = false;

	const categoryLabels: Record<string, string> = {
		bank: 'Konto bankowe',
		saving_account: 'Konto oszczędnościowe',
		stock: 'Akcje',
		bond: 'Obligacje',
		gold: 'Złoto',
		real_estate: 'Nieruchomość',
		ppk: 'PPK',
		fund: 'Fundusz',
		etf: 'ETF',
		vehicle: 'Pojazd',
		mortgage: 'Hipoteka',
		installment: 'Raty',
		other: 'Inne'
	};

	function startCreate() {
		editingAccount = null;
		showForm = true;
	}

	function startEdit(account: Account) {
		editingAccount = account;
		showForm = true;
	}

	function cancelForm() {
		showForm = false;
		editingAccount = null;
	}

	let formData = {
		name: '',
		type: 'asset',
		category: 'bank',
		owner: defaultOwner,
		currency: 'PLN',
		account_wrapper: null as string | null,
		purpose: 'general',
		receives_contributions: true,
		square_meters: null as number | null
	};

	let error = '';
	let saving = false;

	$: if (editingAccount) {
		formData = {
			name: editingAccount.name,
			type: editingAccount.type,
			category: editingAccount.category,
			owner: editingAccount.owner,
			currency: editingAccount.currency,
			account_wrapper: editingAccount.account_wrapper,
			purpose: editingAccount.purpose,
			receives_contributions: editingAccount.receives_contributions,
			square_meters: editingAccount.square_meters
		};
	} else if (showForm) {
		formData = {
			name: '',
			type: 'asset',
			category: 'bank',
			owner: defaultOwner,
			currency: 'PLN',
			account_wrapper: null,
			purpose: 'general',
			receives_contributions: true,
			square_meters: null
		};
	}

	async function handleSubmit() {
		error = '';
		saving = true;

		try {
			const endpoint = editingAccount
				? `${apiUrl}/api/accounts/${editingAccount.id}`
				: `${apiUrl}/api/accounts`;
			const method = editingAccount ? 'PUT' : 'POST';

			// Conditionally include receives_contributions only for PPK accounts
			const payload =
				formData.account_wrapper === 'PPK'
					? formData
					: {
							...formData,
							receives_contributions: undefined
						};

			const response = await fetch(endpoint, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.detail || 'Failed to save account');
			}

			await invalidateAll();
			cancelForm();
		} catch (err) {
			if (err instanceof Error) {
				error = err.message;
			}
		} finally {
			saving = false;
		}
	}

	function handleDelete(accountId: number) {
		accountToDelete = accountId;
		showDeleteModal = true;
	}

	function cancelDelete() {
		showDeleteModal = false;
		accountToDelete = null;
	}

	async function confirmDelete() {
		if (!accountToDelete) return;

		try {
			const response = await fetch(`${apiUrl}/api/accounts/${accountToDelete}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				throw new Error('Failed to delete account');
			}

			await invalidateAll();
			showDeleteModal = false;
			accountToDelete = null;
		} catch (err) {
			if (err instanceof Error) {
				error = err.message;
			}
			showDeleteModal = false;
			accountToDelete = null;
		}
	}

	async function loadTransactionCounts() {
		try {
			const response = await fetch(`${apiUrl}/api/transactions/counts`);
			if (response.ok) {
				transactionCounts = await response.json();
			}
		} catch (err) {
			console.error('Failed to load transaction counts:', err);
		}
	}

	onMount(() => {
		loadTransactionCounts();
	});

	function openTransactions(
		accountId: number,
		accountName: string,
		accountWrapper: string | null = null
	) {
		selectedAccountId = accountId;
		selectedAccountName = accountName;
		selectedAccountWrapper = accountWrapper;
		showTransactionsModal = true;
		loadTransactions();
	}

	async function loadTransactions() {
		if (!selectedAccountId) return;

		try {
			const response = await fetch(`${apiUrl}/api/accounts/${selectedAccountId}/transactions`);
			if (response.ok) {
				transactionsData = await response.json();
			} else {
				const errorData = await response.json();
				transactionError = errorData.detail || 'Failed to load transactions';
			}
		} catch (err) {
			console.error('Failed to load transactions:', err);
			transactionError = 'Failed to load transactions';
		}
	}

	function closeTransactions() {
		showTransactionsModal = false;
		selectedAccountId = null;
		selectedAccountName = '';
		selectedAccountWrapper = null;
		transactionsData = null;
		transactionFormData = {
			amount: 0,
			date: new Date().toISOString().split('T')[0],
			owner: defaultOwner,
			transaction_type: null
		};
		transactionError = '';
	}

	async function addTransaction() {
		if (!selectedAccountId) return;

		transactionError = '';
		savingTransaction = true;

		try {
			const response = await fetch(`${apiUrl}/api/accounts/${selectedAccountId}/transactions`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(transactionFormData)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.detail || 'Failed to add transaction');
			}

			transactionFormData = {
				amount: 0,
				date: new Date().toISOString().split('T')[0],
				owner: defaultOwner,
				transaction_type: null
			};

			await loadTransactions();
			await loadTransactionCounts();
		} catch (err) {
			if (err instanceof Error) {
				transactionError = err.message;
			}
		} finally {
			savingTransaction = false;
		}
	}

	async function deleteTransaction(transactionId: number) {
		if (!selectedAccountId) return;

		try {
			const response = await fetch(
				`${apiUrl}/api/accounts/${selectedAccountId}/transactions/${transactionId}`,
				{
					method: 'DELETE'
				}
			);

			if (!response.ok) {
				throw new Error('Failed to delete transaction');
			}

			await loadTransactions();
			await loadTransactionCounts();
		} catch (err) {
			if (err instanceof Error) {
				transactionError = err.message;
			}
		}
	}
</script>

<svelte:head>
	<title>Konta | Finansowa Forteca</title>
</svelte:head>

<div class="page-header">
	<div>
		<h1 class="page-title">Konta</h1>
		<p class="page-description">Zarządzaj kontami aktywów i pasywów</p>
	</div>
	<button class="btn btn-primary" on:click={startCreate}>+ Nowe Konto</button>
</div>

<Card>
	<CardHeader>
		<CardTitle>💰 Aktywa</CardTitle>
	</CardHeader>
	<CardContent>
		{#if data.assets.length === 0}
			<div class="empty-state">
				<p>Brak aktywów</p>
			</div>
		{:else}
			<Table
				headers={['Nazwa', 'Kategoria', 'Właściciel', 'Wartość', 'Akcje']}
				mobileCardView
				class="accounts-table"
			>
				{#each data.assets as account}
					<tr>
						<td data-label="Nazwa" class="name-cell">{account.name}</td>
						<td data-label="Kategoria">{categoryLabels[account.category] || account.category}</td>
						<td data-label="Właściciel">{account.owner}</td>
						<td data-label="Wartość" class="value-cell">{formatPLN(account.current_value)}</td>
						<td data-label="Akcje" class="actions-cell">
							<button class="btn-icon tap-target" on:click={() => startEdit(account)}>✏️</button>
							{#if INVESTMENT_CATEGORIES.has(account.category) || account.account_wrapper}
								<button
									class="btn-icon tap-target transaction-btn"
									title="Transakcje"
									on:click={() =>
										openTransactions(account.id, account.name, account.account_wrapper)}
								>
									📊 ({transactionCounts[account.id] || 0})
								</button>
							{/if}
							<button class="btn-icon tap-target" on:click={() => handleDelete(account.id)}
								>🗑️</button
							>
						</td>
					</tr>
				{/each}
			</Table>
		{/if}
	</CardContent>
</Card>

<Card>
	<CardHeader>
		<CardTitle>📉 Pasywa</CardTitle>
	</CardHeader>
	<CardContent>
		{#if data.liabilities.length === 0}
			<div class="empty-state">
				<p>Brak pasywów</p>
			</div>
		{:else}
			<Table
				headers={['Nazwa', 'Kategoria', 'Właściciel', 'Wartość', 'Akcje']}
				mobileCardView
				class="accounts-table"
			>
				{#each data.liabilities as account}
					<tr>
						<td data-label="Nazwa" class="name-cell">{account.name}</td>
						<td data-label="Kategoria">{categoryLabels[account.category] || account.category}</td>
						<td data-label="Właściciel">{account.owner}</td>
						<td data-label="Wartość" class="value-cell negative"
							>{formatPLN(account.current_value)}</td
						>
						<td data-label="Akcje" class="actions-cell">
							<button class="btn-icon tap-target" on:click={() => startEdit(account)}>✏️</button>
							<button class="btn-icon tap-target" on:click={() => handleDelete(account.id)}
								>🗑️</button
							>
						</td>
					</tr>
				{/each}
			</Table>
		{/if}
	</CardContent>
</Card>

<Modal
	open={showForm}
	title={editingAccount ? 'Edytuj Konto' : 'Nowe Konto'}
	onConfirm={handleSubmit}
	onCancel={cancelForm}
	confirmText={saving ? 'Zapisywanie...' : editingAccount ? 'Zapisz zmiany' : 'Utwórz konto'}
	confirmDisabled={saving}
	confirmVariant="primary"
	size="large"
>
	<form on:submit|preventDefault={handleSubmit} class="account-form">
		{#if error}
			<div class="error-message">{error}</div>
		{/if}

		<div class="grid grid-cols-1 md:grid-cols-2">
			<div class="form-group">
				<label for="name">Nazwa</label>
				<input
					type="text"
					id="name"
					bind:value={formData.name}
					required
					placeholder="np. mBank Konto"
				/>
			</div>

			<div class="form-group">
				<label for="type">Typ</label>
				<select id="type" bind:value={formData.type} required>
					<option value="asset">Aktywo</option>
					<option value="liability">Pasywo</option>
				</select>
			</div>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-2">
			<div class="form-group">
				<label for="category">Kategoria</label>
				<select id="category" bind:value={formData.category} required>
					<optgroup label="Aktywa">
						<option value="bank">Konto bankowe</option>
						<option value="saving_account">Konto oszczędnościowe</option>
						<option value="stock">Akcje</option>
						<option value="bond">Obligacje</option>
						<option value="gold">Złoto</option>
						<option value="real_estate">Nieruchomość</option>
						<option value="ppk">PPK</option>
						<option value="fund">Fundusz</option>
						<option value="etf">ETF</option>
						<option value="vehicle">Pojazd</option>
					</optgroup>
					<optgroup label="Pasywa">
						<option value="mortgage">Hipoteka</option>
						<option value="installment">Raty</option>
					</optgroup>
					<option value="other">Inne</option>
				</select>
			</div>

			<div class="form-group">
				<label for="owner">Właściciel</label>
				<select id="owner" bind:value={formData.owner} required>
					<option value="Marcin">Marcin</option>
					<option value="Ewa">Ewa</option>
					<option value="Shared">Wspólne</option>
				</select>
			</div>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-2">
			<div class="form-group">
				<label for="account_wrapper">Opakowanie rachunku (opcjonalne)</label>
				<select id="account_wrapper" bind:value={formData.account_wrapper}>
					<option value={null}>Brak</option>
					<option value="IKE">IKE</option>
					<option value="IKZE">IKZE</option>
					<option value="PPK">PPK</option>
				</select>
			</div>

			<div class="form-group">
				<label for="purpose">Cel konta</label>
				<select id="purpose" bind:value={formData.purpose} required>
					<option value="general">Ogólne</option>
					<option value="retirement">Emerytura</option>
					<option value="emergency_fund">Fundusz awaryjny</option>
				</select>
			</div>
		</div>

		{#if formData.account_wrapper === 'PPK'}
			<div class="form-group">
				<label class="checkbox-label">
					<input type="checkbox" bind:checked={formData.receives_contributions} />
					<span>Konto otrzymuje wpłaty (aktywne PPK)</span>
				</label>
				<p class="form-help-text">
					Zaznacz jeśli to konto jest aktywnie używane do otrzymywania miesięcznych wpłat PPK. Stare
					konta PPK powinny mieć to odznaczone - będą tylko śledzone przez snapshoty.
				</p>
			</div>
		{/if}

		{#if formData.category === 'real_estate'}
			<div class="form-group">
				<label for="square_meters">Powierzchnia (m²)</label>
				<input
					type="number"
					id="square_meters"
					bind:value={formData.square_meters}
					min="0"
					step="0.01"
					placeholder="np. 65.50"
				/>
				<p class="form-help-text">
					Powierzchnia nieruchomości w metrach kwadratowych. Używana do obliczania metryki "Ile
					metrów mieszkania jest nasze".
				</p>
			</div>
		{/if}
	</form>
</Modal>

<Modal
	open={showDeleteModal}
	title="Potwierdzenie usunięcia"
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
>
	<p>Czy na pewno chcesz usunąć to konto?</p>
	<p>Operacja ta ustawi konto jako nieaktywne.</p>
</Modal>

<Modal
	open={showTransactionsModal}
	title="Transakcje - {selectedAccountName}"
	onCancel={closeTransactions}
	size="large"
>
	<div class="transactions-modal">
		{#if transactionError}
			<div class="error-message">{transactionError}</div>
		{/if}

		{#if transactionsData}
			<div class="transactions-header">
				<h3>Historia transakcji</h3>
				<p class="total-invested">
					Zainwestowano łącznie: <strong>{formatPLN(transactionsData.total_invested)}</strong>
				</p>
			</div>

			{#if transactionsData.transactions.length === 0}
				<div class="empty-state">
					<p>Brak transakcji</p>
				</div>
			{:else}
				<div class="table-container">
					<table class="transactions-table">
						<thead>
							<tr>
								<th>Data zakupu</th>
								<th>Kwota</th>
								<th>Właściciel</th>
								<th>Akcje</th>
							</tr>
						</thead>
						<tbody>
							{#each transactionsData.transactions as transaction}
								<tr>
									<td>{new Date(transaction.date).toLocaleDateString('pl-PL')}</td>
									<td class="value-cell">{formatPLN(transaction.amount)}</td>
									<td>{transaction.owner}</td>
									<td class="actions-cell">
										<button
											class="btn-icon"
											on:click={() => deleteTransaction(transaction.id)}
											title="Usuń"
										>
											🗑️
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}

			<div class="transaction-form">
				<h3>Dodaj transakcję</h3>
				<form on:submit|preventDefault={addTransaction}>
					<div class="grid grid-cols-1 md:grid-cols-2">
						<div class="form-group">
							<label for="transaction-amount">Kwota (PLN)</label>
							<input
								type="number"
								id="transaction-amount"
								bind:value={transactionFormData.amount}
								required
								min="0.01"
								step="0.01"
								placeholder="np. 5000.00"
							/>
						</div>

						<div class="form-group">
							<label for="transaction-date">Data zakupu</label>
							<input
								type="date"
								id="transaction-date"
								bind:value={transactionFormData.date}
								required
								max={new Date().toISOString().split('T')[0]}
							/>
						</div>

						<div class="form-group">
							<label for="transaction-owner">Właściciel</label>
							<select id="transaction-owner" bind:value={transactionFormData.owner} required>
								<option value="Marcin">Marcin</option>
								<option value="Ewa">Ewa</option>
								<option value="Shared">Wspólne</option>
							</select>
						</div>

						{#if selectedAccountWrapper}
							<div class="form-group">
								<label for="transaction-type">Typ wpłaty</label>
								<select id="transaction-type" bind:value={transactionFormData.transaction_type}>
									<option value="">Wpłata pracownika</option>
									{#if selectedAccountWrapper === 'PPK'}
										<option value="employer">Wpłata pracodawcy</option>
									{/if}
									<option value="withdrawal">Wypłata</option>
								</select>
							</div>
						{/if}
					</div>

					<button type="submit" class="btn btn-primary" disabled={savingTransaction}>
						{savingTransaction ? 'Zapisywanie...' : 'Dodaj transakcję'}
					</button>
				</form>
			</div>
		{/if}
	</div>
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

	.accounts-table {
		width: 100%;
		border-collapse: collapse;
	}

	.accounts-table thead {
		border-bottom: 2px solid var(--color-border);
	}

	.accounts-table th {
		text-align: left;
		padding: var(--size-3) var(--size-4);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.accounts-table tbody tr {
		border-bottom: 1px solid var(--color-border);
		transition: background-color 0.2s;
	}

	.accounts-table tbody tr:hover {
		background-color: var(--color-accent);
	}

	.accounts-table td {
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

	.value-cell.negative {
		color: var(--nord11);
	}

	.actions-cell {
		text-align: right;
	}

	.account-form {
		display: flex;
		flex-direction: column;
		gap: var(--size-5);
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

	.error-message {
		padding: var(--size-3);
		background: var(--nord11);
		color: var(--nord6);
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
	}

	.checkbox-label {
		display: flex;
		align-items: center;
		gap: var(--size-2);
		cursor: pointer;
		font-weight: var(--font-weight-5);
	}

	.checkbox-label input[type='checkbox'] {
		width: var(--size-4);
		height: var(--size-4);
		cursor: pointer;
	}

	.form-help-text {
		margin: var(--size-2) 0 0 0;
		font-size: var(--font-size-1);
		color: var(--color-text-secondary);
		line-height: 1.4;
	}

	.transactions-modal {
		display: flex;
		flex-direction: column;
		gap: var(--size-6);
	}

	.transactions-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-bottom: var(--size-4);
		border-bottom: 2px solid var(--color-border);
	}

	.transactions-header h3 {
		margin: 0;
		font-size: var(--font-size-4);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
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
		padding: var(--size-3) var(--size-4);
		font-size: var(--font-size-2);
	}

	.transaction-form {
		padding-top: var(--size-4);
		border-top: 2px solid var(--color-border);
	}

	.transaction-form h3 {
		margin: 0 0 var(--size-4) 0;
		font-size: var(--font-size-3);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
	}

	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			gap: var(--size-4);
		}

		.transactions-header {
			flex-direction: column;
			align-items: flex-start;
			gap: var(--size-2);
		}
	}
</style>
