<script lang="ts">
	import Card from '$lib/components/Card.svelte';
	import CardHeader from '$lib/components/CardHeader.svelte';
	import CardTitle from '$lib/components/CardTitle.svelte';
	import CardContent from '$lib/components/CardContent.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import type { Account } from './+page';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';

	let showForm = false;
	let editingAccount: Account | null = null;
	let showDeleteModal = false;
	let accountToDelete: number | null = null;

	const categoryLabels: Record<string, string> = {
		bank: 'Konto bankowe',
		ike: 'IKE',
		ikze: 'IKZE',
		ppk: 'PPK',
		fund: 'Fundusz',
		etf: 'ETF',
		bonds: 'Obligacje',
		stocks: 'Akcje',
		real_estate: 'Nieruchomość',
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
		owner: 'Marcin',
		currency: 'PLN'
	};

	let error = '';
	let saving = false;

	$: if (editingAccount) {
		formData = {
			name: editingAccount.name,
			type: editingAccount.type,
			category: editingAccount.category,
			owner: editingAccount.owner,
			currency: editingAccount.currency
		};
	} else if (showForm) {
		formData = {
			name: '',
			type: 'asset',
			category: 'bank',
			owner: 'Marcin',
			currency: 'PLN'
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

			const response = await fetch(endpoint, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(formData)
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

{#if showForm}
	<Card>
		<CardHeader>
			<CardTitle>{editingAccount ? '✏️ Edytuj Konto' : '➕ Nowe Konto'}</CardTitle>
		</CardHeader>
		<CardContent>
			<form on:submit|preventDefault={handleSubmit} class="account-form">
				{#if error}
					<div class="error-message">{error}</div>
				{/if}

				<div class="form-row">
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

				<div class="form-row">
					<div class="form-group">
						<label for="category">Kategoria</label>
						<select id="category" bind:value={formData.category} required>
							<optgroup label="Aktywa">
								<option value="bank">Konto bankowe</option>
								<option value="ike">IKE</option>
								<option value="ikze">IKZE</option>
								<option value="ppk">PPK</option>
								<option value="fund">Fundusz</option>
								<option value="etf">ETF</option>
								<option value="bonds">Obligacje</option>
								<option value="stocks">Akcje</option>
								<option value="real_estate">Nieruchomość</option>
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

				<div class="form-actions">
					<button type="button" class="btn btn-secondary" on:click={cancelForm} disabled={saving}>
						Anuluj
					</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>
						{saving ? 'Zapisywanie...' : editingAccount ? 'Zapisz zmiany' : 'Utwórz konto'}
					</button>
				</div>
			</form>
		</CardContent>
	</Card>
{/if}

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
			<div class="table-container">
				<table class="accounts-table">
					<thead>
						<tr>
							<th>Nazwa</th>
							<th>Kategoria</th>
							<th>Właściciel</th>
							<th>Wartość</th>
							<th>Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.assets as account}
							<tr>
								<td class="name-cell">{account.name}</td>
								<td>{categoryLabels[account.category] || account.category}</td>
								<td>{account.owner}</td>
								<td class="value-cell">{formatPLN(account.current_value)}</td>
								<td class="actions-cell">
									<button class="btn-icon" on:click={() => startEdit(account)}>✏️</button>
									<button class="btn-icon" on:click={() => handleDelete(account.id)}>🗑️</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
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
			<div class="table-container">
				<table class="accounts-table">
					<thead>
						<tr>
							<th>Nazwa</th>
							<th>Kategoria</th>
							<th>Właściciel</th>
							<th>Wartość</th>
							<th>Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.liabilities as account}
							<tr>
								<td class="name-cell">{account.name}</td>
								<td>{categoryLabels[account.category] || account.category}</td>
								<td>{account.owner}</td>
								<td class="value-cell negative">{formatPLN(account.current_value)}</td>
								<td class="actions-cell">
									<button class="btn-icon" on:click={() => startEdit(account)}>✏️</button>
									<button class="btn-icon" on:click={() => handleDelete(account.id)}>🗑️</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</CardContent>
</Card>

<Modal
	open={showDeleteModal}
	title="Potwierdzenie usunięcia"
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
>
	<p>Czy na pewno chcesz usunąć to konto?</p>
	<p>Operacja ta ustawi konto jako nieaktywne.</p>
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

	.form-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
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

	.form-actions {
		display: flex;
		gap: var(--size-3);
		justify-content: flex-end;
	}

	.btn-secondary {
		background: transparent;
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}

	.btn-secondary:hover {
		background: var(--color-accent);
	}

	.error-message {
		padding: var(--size-3);
		background: var(--nord11);
		color: var(--nord6);
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
	}

	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			gap: var(--size-4);
		}

		.form-row {
			grid-template-columns: 1fr;
		}
	}
</style>
