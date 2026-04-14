<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN, formatDate } from '$lib/utils/format';
	import { ClipboardList, Plus, Pencil, Trash2, Wallet, CircleDollarSign } from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import { onMount } from 'svelte';
	import type { Debt, DebtPayment } from './+page';
	import type { Persona } from '$lib/types/personas';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
	$: personas = data.personas as Persona[];
	$: defaultOwner = personas.length > 0 ? personas[0].name : 'Marcin';

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
			const response = await fetch(`${apiUrl}/api/debts/${debtToDelete}`, { method: 'DELETE' });

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
			selectedDebt = null;
			paymentsData = null;
		} else {
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
				{ method: 'DELETE' }
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

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Zobowiązania</h1>
		<p class="text-surface-700-300 text-sm">Zarządzaj długami i wpłatami</p>
	</div>
	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto gap-2"
		on:click={startCreate}
	>
		<Plus size={16} />
		Nowe Zobowiązanie
	</button>
</div>

<div class="space-y-4">
	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><ClipboardList size={20} /> Lista zobowiązań</h3>
		</header>

		{#if data.debts.length === 0}
			<div class="text-center py-12 text-surface-700-300">
				<p>Brak zobowiązań</p>
			</div>
		{:else}
			<div class="table-wrap">
				<table class="table table-hover">
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
							<th class="text-right">Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.debts as debt}
							<tr>
								<td>{debt.name}</td>
								<td>{debtTypeLabels[debt.debt_type] || debt.debt_type}</td>
								<td class="font-semibold">{formatPLN(debt.initial_amount)}</td>
								<td class="font-semibold text-error-600-400">
									{#if debt.latest_balance !== null}
										<div>{formatPLN(debt.latest_balance)}</div>
										{#if debt.latest_balance_date}
											<div class="text-xs text-surface-700-300">
												na {formatDate(debt.latest_balance_date)}
											</div>
										{/if}
									{:else}
										<span class="text-surface-700-300 italic">brak danych</span>
									{/if}
								</td>
								<td class="font-semibold">{formatPLN(debt.total_paid)}</td>
								<td class="font-semibold text-error-600-400">{formatPLN(debt.interest_paid)}</td>
								<td>{debt.interest_rate}%</td>
								<td>{formatDate(debt.start_date)}</td>
								<td>{debt.account_owner}</td>
								<td class="text-right whitespace-nowrap">
									<button
										type="button"
										class="btn preset-tonal-surface btn-sm gap-1"
										aria-label="Historia wpłat"
										on:click={() => openPayments(debt)}
									>
										<CircleDollarSign size={14} />
										<span>{paymentCounts[debt.account_id] || 0}</span>
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										on:click={() => startEdit(debt)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										on:click={() => handleDelete(debt.id)}
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

	{#if selectedDebt && paymentsData}
		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header>
				<h3 class="h3 flex items-center gap-2">
					<Plus size={20} /> Dodaj wpłatę - {selectedDebt.name}
				</h3>
			</header>

			<form class="space-y-4" on:submit|preventDefault={addPayment}>
				<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
					<label class="label">
						<span class="font-semibold text-sm">Kwota</span>
						<input
							type="number"
							class="input"
							step="0.01"
							bind:value={paymentFormData.amount}
							required
						/>
					</label>
					<label class="label">
						<span class="font-semibold text-sm">Data wpłaty</span>
						<input type="date" class="input" bind:value={paymentFormData.date} required />
					</label>
					<label class="label">
						<span class="font-semibold text-sm">Kto wpłacił</span>
						<select class="select" bind:value={paymentFormData.owner}>
							{#each personas as persona}
								<option value={persona.name}>{persona.name}</option>
							{/each}
						</select>
					</label>
				</div>

				{#if paymentError}
					<div class="card preset-filled-error-500 p-3 text-sm">{paymentError}</div>
				{/if}

				<button type="submit" class="btn preset-filled-primary-500" disabled={savingPayment}>
					{savingPayment ? 'Zapisywanie...' : 'Dodaj wpłatę'}
				</button>
			</form>
		</div>

		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header>
				<h3 class="h3 flex items-center gap-2">
					<Wallet size={20} /> Historia wpłat - {selectedDebt.name}
				</h3>
			</header>

			<div class="text-sm text-surface-700-300 space-y-1">
				<p>
					Wpłacono łącznie: <strong class="text-surface-950-50"
						>{formatPLN(paymentsData.total_paid)}</strong
					>
				</p>
				<p>Liczba wpłat: {paymentsData.payment_count}</p>
			</div>

			{#if paymentsData.payments.length === 0}
				<div class="text-center py-8 text-surface-700-300">Brak wpłat</div>
			{:else}
				<div class="table-wrap">
					<table class="table table-hover">
						<thead>
							<tr>
								<th>Data wpłaty</th>
								<th>Kwota</th>
								<th>Kto wpłacił</th>
								<th class="text-right">Akcje</th>
							</tr>
						</thead>
						<tbody>
							{#each paymentsData.payments as payment}
								<tr>
									<td>{formatDate(payment.date)}</td>
									<td>{formatPLN(payment.amount)}</td>
									<td>{payment.owner}</td>
									<td class="text-right">
										<button
											type="button"
											class="btn-icon btn-icon-sm"
											aria-label="Usuń wpłatę"
											on:click={() => handleDeletePayment(payment.id)}
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
	{/if}
</div>

<Modal
	open={showForm}
	title={editingDebt ? 'Edytuj zobowiązanie' : 'Nowe zobowiązanie'}
	onConfirm={handleSubmit}
	onCancel={cancelForm}
	confirmText={saving ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={saving}
>
	<form on:submit|preventDefault={handleSubmit} class="space-y-4">
		{#if error}
			<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
		{/if}

		<label class="label">
			<span class="font-semibold text-sm">Nazwa</span>
			<input type="text" class="input" bind:value={formData.name} required />
		</label>

		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			<label class="label">
				<span class="font-semibold text-sm">Typ zobowiązania</span>
				<select class="select" bind:value={formData.debt_type} required>
					<option value="mortgage">Hipoteka</option>
					<option value="installment_0percent">Raty 0%</option>
				</select>
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Data rozpoczęcia</span>
				<input type="date" class="input" bind:value={formData.start_date} required />
			</label>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			<label class="label">
				<span class="font-semibold text-sm">Kwota początkowa (główna)</span>
				<input
					type="number"
					class="input"
					step="0.01"
					bind:value={formData.initial_amount}
					required
				/>
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Oprocentowanie (%)</span>
				<input
					type="number"
					class="input"
					step="0.01"
					bind:value={formData.interest_rate}
					required
				/>
			</label>
		</div>

		<label class="label">
			<span class="font-semibold text-sm">Notatki</span>
			<textarea class="textarea" bind:value={formData.notes} rows="3"></textarea>
		</label>
	</form>
</Modal>

<Modal
	open={showDeleteModal}
	title="Potwierdź usunięcie"
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
	confirmText="Usuń"
	confirmVariant="danger"
>
	<p>Czy na pewno chcesz usunąć to zobowiązanie?</p>
</Modal>

<Modal
	open={showDeletePaymentModal}
	title="Potwierdź usunięcie wpłaty"
	onConfirm={confirmDeletePayment}
	onCancel={cancelDeletePayment}
	confirmText="Usuń"
	confirmVariant="danger"
>
	<p>Czy na pewno chcesz usunąć tę wpłatę?</p>
</Modal>
