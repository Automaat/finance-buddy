<script lang="ts">
	import Card from '$lib/components/Card.svelte';
	import CardHeader from '$lib/components/CardHeader.svelte';
	import CardTitle from '$lib/components/CardTitle.svelte';
	import CardContent from '$lib/components/CardContent.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN, formatDate } from '$lib/utils/format';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import { onMount } from 'svelte';
	import type { Debt, DebtPayment } from './+page';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
	const defaultOwner = env.PUBLIC_DEFAULT_OWNER || 'Marcin';

	let showForm = false;
	let editingDebt: Debt | null = null;
	let showDeleteModal = false;
	let debtToDelete: number | null = null;
	let paymentCounts: Record<number, number> = {};

	let selectedDebt: Debt | null = null;
	let paymentsData: {
		payments: DebtPayment[];
		total_paid: number;
		payment_count: number;
	} | null = null;
	let paymentFormData = {
		amount: 0,
		date: new Date().toISOString().split('T')[0],
		owner: defaultOwner
	};
	let paymentError = '';
	let savingPayment = false;
	let showDeletePaymentModal = false;
	let paymentToDelete: number | null = null;

	const debtTypeLabels: Record<string, string> = {
		mortgage: 'Hipoteka',
		installment_0percent: 'Raty 0%'
	};

	function startCreate() {
		editingDebt = null;
		showForm = true;
	}

	function startEdit(debt: Debt) {
		editingDebt = debt;
		showForm = true;
	}

	function cancelForm() {
		showForm = false;
		editingDebt = null;
	}

	let formData = {
		name: '',
		debt_type: 'mortgage',
		start_date: new Date().toISOString().split('T')[0],
		initial_amount: 0,
		interest_rate: 0,
		currency: 'PLN',
		notes: null as string | null
	};

	let error = '';
	let saving = false;

	$: if (editingDebt) {
		formData = {
			name: editingDebt.name,
			debt_type: editingDebt.debt_type,
			start_date: editingDebt.start_date,
			initial_amount: editingDebt.initial_amount,
			interest_rate: editingDebt.interest_rate,
			currency: editingDebt.currency,
			notes: editingDebt.notes
		};
	} else if (showForm) {
		formData = {
			name: '',
			debt_type: 'mortgage',
			start_date: new Date().toISOString().split('T')[0],
			initial_amount: 0,
			interest_rate: 0,
			currency: 'PLN',
			notes: null
		};
	}

	async function handleSubmit() {
		error = '';
		saving = true;

		try {
			let endpoint: string;
			let method: string;

			if (editingDebt) {
				endpoint = `${apiUrl}/api/debts/${editingDebt.id}`;
				method = 'PUT';
			} else {
				const accountOwner = defaultOwner;
				const tempAccount = {
					name: formData.name,
					type: 'liability',
					category: formData.debt_type === 'mortgage' ? 'mortgage' : 'installment',
					owner: accountOwner,
					currency: formData.currency
				};

				const accountResponse = await fetch(`${apiUrl}/api/accounts`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(tempAccount)
				});

				if (!accountResponse.ok) {
					const errorData = await accountResponse.json();
					throw new Error(errorData.detail || 'Failed to create account');
				}

				const accountData = await accountResponse.json();
				endpoint = `${apiUrl}/api/accounts/${accountData.id}/debts`;
				method = 'POST';
			}

			const response = await fetch(endpoint, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(formData)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.detail || 'Failed to save debt');
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

	function handleDelete(debtId: number) {
		debtToDelete = debtId;
		showDeleteModal = true;
	}

	function cancelDelete() {
		showDeleteModal = false;
		debtToDelete = null;
	}

	async function confirmDelete() {
		if (!debtToDelete) return;

		try {
			const response = await fetch(`${apiUrl}/api/debts/${debtToDelete}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				throw new Error('Failed to delete debt');
			}

			await invalidateAll();
			showDeleteModal = false;
			debtToDelete = null;
		} catch (err) {
			if (err instanceof Error) {
				error = err.message;
			}
			showDeleteModal = false;
			debtToDelete = null;
		}
	}

	async function loadPaymentCounts() {
		try {
			const response = await fetch(`${apiUrl}/api/payments/counts`);
			if (response.ok) {
				paymentCounts = await response.json();
			}
		} catch (err) {
			console.error('Failed to load payment counts:', err);
		}
	}

	onMount(() => {
		loadPaymentCounts();
	});

	async function openPayments(debt: Debt) {
		if (selectedDebt?.id === debt.id) {
			// Toggle off if clicking same debt
			selectedDebt = null;
			paymentsData = null;
		} else {
			// Show payments for this debt
			selectedDebt = debt;
			await loadPayments();
		}
	}

	async function loadPayments() {
		if (!selectedDebt) return;

		try {
			const response = await fetch(`${apiUrl}/api/accounts/${selectedDebt.account_id}/payments`);
			if (response.ok) {
				paymentsData = await response.json();
			} else {
				const errorData = await response.json();
				paymentError = errorData.detail || 'Failed to load payments';
			}
		} catch (err) {
			console.error('Failed to load payments:', err);
			paymentError = 'Failed to load payments';
		}
	}

	async function addPayment() {
		if (!selectedDebt) return;

		paymentError = '';
		savingPayment = true;

		try {
			const response = await fetch(`${apiUrl}/api/accounts/${selectedDebt.account_id}/payments`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(paymentFormData)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.detail || 'Failed to add payment');
			}

			paymentFormData = {
				amount: 0,
				date: new Date().toISOString().split('T')[0],
				owner: defaultOwner
			};

			await loadPayments();
			await loadPaymentCounts();
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) {
				paymentError = err.message;
			}
		} finally {
			savingPayment = false;
		}
	}

	function handleDeletePayment(paymentId: number) {
		paymentToDelete = paymentId;
		showDeletePaymentModal = true;
	}

	function cancelDeletePayment() {
		showDeletePaymentModal = false;
		paymentToDelete = null;
	}

	async function confirmDeletePayment() {
		if (!selectedDebt || !paymentToDelete) return;

		try {
			const response = await fetch(
				`${apiUrl}/api/accounts/${selectedDebt.account_id}/payments/${paymentToDelete}`,
				{
					method: 'DELETE'
				}
			);

			if (!response.ok) {
				throw new Error('Failed to delete payment');
			}

			await loadPayments();
			await loadPaymentCounts();
			await invalidateAll();
			showDeletePaymentModal = false;
			paymentToDelete = null;
		} catch (err) {
			if (err instanceof Error) {
				paymentError = err.message;
			}
			showDeletePaymentModal = false;
			paymentToDelete = null;
		}
	}
</script>

<svelte:head>
	<title>Zobowiązania | Finansowa Forteca</title>
</svelte:head>

<div class="page-header">
	<div>
		<h1 class="page-title">Zobowiązania</h1>
		<p class="page-description">Zarządzaj długami i wpłatami</p>
	</div>
	<button class="btn btn-primary" on:click={startCreate}>+ Nowe Zobowiązanie</button>
</div>

<Card>
	<CardHeader>
		<CardTitle>📋 Lista zobowiązań</CardTitle>
	</CardHeader>
	<CardContent>
		{#if data.debts.length === 0}
			<div class="empty-state">
				<p>Brak zobowiązań</p>
			</div>
		{:else}
			<div class="table-container">
				<table class="data-table">
					<thead>
						<tr>
							<th>Nazwa</th>
							<th>Typ</th>
							<th>Kwota początkowa</th>
							<th>Pozostało do spłaty</th>
							<th>Wpłacono łącznie</th>
							<th>Odsetki</th>
							<th>Oprocentowanie</th>
							<th>Data rozpoczęcia</th>
							<th>Właściciel</th>
							<th>Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.debts as debt}
							<tr>
								<td>{debt.name}</td>
								<td>{debtTypeLabels[debt.debt_type] || debt.debt_type}</td>
								<td class="value-cell">{formatPLN(debt.initial_amount)}</td>
								<td class="value-cell negative">
									{#if debt.latest_balance !== null}
										<div>{formatPLN(debt.latest_balance)}</div>
										{#if debt.latest_balance_date}
											<div class="amount-detail">na {formatDate(debt.latest_balance_date)}</div>
										{/if}
									{:else}
										<span class="text-muted">brak danych</span>
									{/if}
								</td>
								<td class="value-cell">{formatPLN(debt.total_paid)}</td>
								<td class="value-cell negative">{formatPLN(debt.interest_paid)}</td>
								<td>{debt.interest_rate}%</td>
								<td>{formatDate(debt.start_date)}</td>
								<td>{debt.account_owner}</td>
								<td class="actions-cell">
									<button
										class="btn-icon"
										title="Historia wpłat"
										on:click={() => openPayments(debt)}
									>
										💰 ({paymentCounts[debt.account_id] || 0})
									</button>
									<button class="btn-icon" title="Edytuj" on:click={() => startEdit(debt)}>
										✏️
									</button>
									<button class="btn-icon" title="Usuń" on:click={() => handleDelete(debt.id)}>
										🗑️
									</button>
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
	open={showForm}
	title={editingDebt ? 'Edytuj zobowiązanie' : 'Nowe zobowiązanie'}
	onConfirm={handleSubmit}
	onCancel={cancelForm}
	confirmText={saving ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={saving}
	confirmVariant="primary"
	size="medium"
>
	<form on:submit|preventDefault={handleSubmit}>
		{#if error}
			<div class="error-message">{error}</div>
		{/if}

		<div class="form-group">
			<label for="name">Nazwa</label>
			<input id="name" type="text" bind:value={formData.name} required />
		</div>

		<div class="form-row">
			<div class="form-group">
				<label for="debt_type">Typ zobowiązania</label>
				<select id="debt_type" bind:value={formData.debt_type} required>
					<option value="mortgage">Hipoteka</option>
					<option value="installment_0percent">Raty 0%</option>
				</select>
			</div>

			<div class="form-group">
				<label for="start_date">Data rozpoczęcia</label>
				<input id="start_date" type="date" bind:value={formData.start_date} required />
			</div>
		</div>

		<div class="form-row">
			<div class="form-group">
				<label for="initial_amount">Kwota początkowa (główna)</label>
				<input
					id="initial_amount"
					type="number"
					step="0.01"
					bind:value={formData.initial_amount}
					required
				/>
			</div>

			<div class="form-group">
				<label for="interest_rate">Oprocentowanie (%)</label>
				<input
					id="interest_rate"
					type="number"
					step="0.01"
					bind:value={formData.interest_rate}
					required
				/>
			</div>
		</div>

		<div class="form-group">
			<label for="notes">Notatki</label>
			<textarea id="notes" bind:value={formData.notes} rows="3"></textarea>
		</div>
	</form>
</Modal>

<Modal
	open={showDeleteModal}
	title="Potwierdź usunięcie"
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
	confirmText="Usuń"
	size="small"
>
	<p>Czy na pewno chcesz usunąć to zobowiązanie?</p>
</Modal>

{#if selectedDebt && paymentsData}
	<Card>
		<CardHeader>
			<CardTitle>➕ Dodaj wpłatę - {selectedDebt.name}</CardTitle>
		</CardHeader>
		<CardContent>
			<form on:submit|preventDefault={addPayment}>
				<div class="form-row">
					<div class="form-group">
						<label for="payment_amount">Kwota</label>
						<input
							id="payment_amount"
							type="number"
							step="0.01"
							bind:value={paymentFormData.amount}
							required
						/>
					</div>
					<div class="form-group">
						<label for="payment_date">Data wpłaty</label>
						<input id="payment_date" type="date" bind:value={paymentFormData.date} required />
					</div>
					<div class="form-group">
						<label for="payment_owner">Kto wpłacił</label>
						<select id="payment_owner" bind:value={paymentFormData.owner}>
							<option value="Marcin">Marcin</option>
							<option value="Ewa">Ewa</option>
							<option value="Shared">Wspólne</option>
						</select>
					</div>
				</div>

				{#if paymentError}
					<div class="error-message">{paymentError}</div>
				{/if}

				<button type="submit" class="btn btn-primary" disabled={savingPayment}>
					{savingPayment ? 'Zapisywanie...' : 'Dodaj wpłatę'}
				</button>
			</form>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>💰 Historia wpłat - {selectedDebt.name}</CardTitle>
		</CardHeader>
		<CardContent>
			<div class="payments-summary">
				<p>Wpłacono łącznie: <strong>{formatPLN(paymentsData.total_paid)}</strong></p>
				<p>Liczba wpłat: {paymentsData.payment_count}</p>
			</div>

			<div class="table-container">
				<table class="data-table">
					<thead>
						<tr>
							<th>Data wpłaty</th>
							<th>Kwota</th>
							<th>Kto wpłacił</th>
							<th>Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#if paymentsData.payments.length === 0}
							<tr>
								<td colspan="4" class="empty-state">Brak wpłat</td>
							</tr>
						{:else}
							{#each paymentsData.payments as payment}
								<tr>
									<td>{formatDate(payment.date)}</td>
									<td>{formatPLN(payment.amount)}</td>
									<td>{payment.owner}</td>
									<td class="actions-cell">
										<button
											class="btn-icon"
											title="Usuń wpłatę"
											on:click={() => handleDeletePayment(payment.id)}
										>
											🗑️
										</button>
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		</CardContent>
	</Card>
{/if}

<Modal
	open={showDeletePaymentModal}
	title="Potwierdź usunięcie wpłaty"
	onConfirm={confirmDeletePayment}
	onCancel={cancelDeletePayment}
	confirmText="Usuń"
	size="small"
>
	<p>Czy na pewno chcesz usunąć tę wpłatę?</p>
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

	.btn-primary:hover:not(:disabled) {
		background: var(--nord9);
	}

	.btn-primary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-secondary {
		background: transparent;
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}

	.btn-secondary:hover {
		background: var(--color-accent);
	}

	.btn-danger {
		background: var(--nord11);
		color: var(--nord6);
	}

	.btn-danger:hover {
		background: var(--nord12);
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

	.data-table {
		width: 100%;
		border-collapse: collapse;
	}

	.data-table thead {
		border-bottom: 2px solid var(--color-border);
	}

	.data-table th {
		text-align: left;
		padding: var(--size-3) var(--size-4);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.data-table tbody tr {
		border-bottom: 1px solid var(--color-border);
		transition: background-color 0.2s;
	}

	.data-table tbody tr:hover {
		background-color: var(--color-accent);
	}

	.data-table td {
		padding: var(--size-4);
		font-size: var(--font-size-2);
	}

	.value-cell {
		font-weight: var(--font-weight-6);
		color: var(--color-primary);
	}

	.value-cell.negative {
		color: var(--nord11);
	}

	.text-muted {
		color: var(--color-text-2);
		font-style: italic;
		font-size: var(--font-size-1);
	}

	.actions-cell {
		text-align: right;
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
	.form-group select,
	.form-group textarea {
		padding: var(--size-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: var(--color-background);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.form-group textarea {
		font-family: inherit;
		resize: vertical;
	}

	.form-group input:focus,
	.form-group select:focus,
	.form-group textarea:focus {
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

	.modal-actions {
		display: flex;
		gap: var(--size-3);
		justify-content: flex-end;
		margin-top: var(--size-4);
	}

	.payments-summary {
		background: var(--color-surface-2);
		padding: var(--size-4);
		border-radius: var(--radius-2);
		margin-bottom: var(--size-4);
	}

	.payments-summary p {
		margin: var(--size-2) 0;
		font-size: var(--font-size-2);
		color: var(--color-text-secondary);
	}

	.payments-summary strong {
		color: var(--color-primary);
		font-weight: var(--font-weight-7);
	}

	.payment-form {
		margin-top: var(--size-6);
		padding-top: var(--size-4);
		border-top: 2px solid var(--color-border);
	}

	.payment-form h3 {
		margin: 0 0 var(--size-4) 0;
		font-size: var(--font-size-3);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
	}

	.negative {
		color: var(--nord11);
	}

	.amount-detail {
		font-size: var(--font-size-1);
		color: var(--color-text-secondary);
		font-weight: var(--font-weight-4);
		margin-top: var(--size-1);
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
