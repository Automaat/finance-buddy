<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { Wallet, TrendingDown, Pencil, Trash2, Plus, BarChart3 } from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import { onMount } from 'svelte';
	import { INVESTMENT_CATEGORIES } from '$lib/constants';
	import type { Account, TransactionsData } from './+page';
	import type { Persona } from '$lib/types/personas';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
	$: personas = data.personas as Persona[];
	$: defaultOwner = personas.length > 0 ? personas[0].name : 'Marcin';

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

			const payload =
				formData.account_wrapper === 'PPK'
					? formData
					: { ...formData, receives_contributions: undefined };

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
				{ method: 'DELETE' }
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

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Konta</h1>
		<p class="text-surface-700-300 text-sm">Zarządzaj kontami aktywów i pasywów</p>
	</div>
	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto gap-2"
		on:click={startCreate}
	>
		<Plus size={16} />
		Nowe Konto
	</button>
</div>

<div class="space-y-4">
	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Wallet size={20} /> Aktywa</h3>
		</header>
		{#if data.assets.length === 0}
			<div class="text-center py-12 text-surface-700-300"><p>Brak aktywów</p></div>
		{:else}
			<div class="table-wrap">
				<table class="table table-hover">
					<thead>
						<tr>
							<th>Nazwa</th>
							<th>Kategoria</th>
							<th>Właściciel</th>
							<th>Wartość</th>
							<th class="text-right">Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.assets as account}
							<tr>
								<td class="font-medium">{account.name}</td>
								<td>{categoryLabels[account.category] || account.category}</td>
								<td>{account.owner}</td>
								<td class="font-semibold text-primary-600-400"
									>{formatPLN(account.current_value)}</td
								>
								<td class="text-right whitespace-nowrap">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										on:click={() => startEdit(account)}
									>
										<Pencil size={16} />
									</button>
									{#if INVESTMENT_CATEGORIES.has(account.category) || account.account_wrapper}
										<button
											type="button"
											class="btn preset-tonal-surface btn-sm gap-1"
											aria-label="Transakcje"
											on:click={() =>
												openTransactions(account.id, account.name, account.account_wrapper)}
										>
											<BarChart3 size={14} />
											<span>{transactionCounts[account.id] || 0}</span>
										</button>
									{/if}
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										on:click={() => handleDelete(account.id)}
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

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><TrendingDown size={20} /> Pasywa</h3>
		</header>
		{#if data.liabilities.length === 0}
			<div class="text-center py-12 text-surface-700-300"><p>Brak pasywów</p></div>
		{:else}
			<div class="table-wrap">
				<table class="table table-hover">
					<thead>
						<tr>
							<th>Nazwa</th>
							<th>Kategoria</th>
							<th>Właściciel</th>
							<th>Wartość</th>
							<th class="text-right">Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.liabilities as account}
							<tr>
								<td class="font-medium">{account.name}</td>
								<td>{categoryLabels[account.category] || account.category}</td>
								<td>{account.owner}</td>
								<td class="font-semibold text-error-600-400">{formatPLN(account.current_value)}</td>
								<td class="text-right whitespace-nowrap">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										on:click={() => startEdit(account)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										on:click={() => handleDelete(account.id)}
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
	open={showForm}
	title={editingAccount ? 'Edytuj Konto' : 'Nowe Konto'}
	onConfirm={handleSubmit}
	onCancel={cancelForm}
	confirmText={saving ? 'Zapisywanie...' : editingAccount ? 'Zapisz zmiany' : 'Utwórz konto'}
	confirmDisabled={saving}
>
	<form on:submit|preventDefault={handleSubmit} class="space-y-4">
		{#if error}
			<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
		{/if}

		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			<label class="label">
				<span class="font-semibold text-sm">Nazwa</span>
				<input
					type="text"
					class="input"
					bind:value={formData.name}
					required
					placeholder="np. mBank Konto"
				/>
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Typ</span>
				<select class="select" bind:value={formData.type} required>
					<option value="asset">Aktywo</option>
					<option value="liability">Pasywo</option>
				</select>
			</label>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			<label class="label">
				<span class="font-semibold text-sm">Kategoria</span>
				<select class="select" bind:value={formData.category} required>
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
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Właściciel</span>
				<select class="select" bind:value={formData.owner} required>
					{#if !personas.some((p) => p.name === formData.owner)}
						<option value={formData.owner}>{formData.owner}</option>
					{/if}
					{#each personas as persona}
						<option value={persona.name}>{persona.name}</option>
					{/each}
				</select>
			</label>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			<label class="label">
				<span class="font-semibold text-sm">Opakowanie rachunku (opcjonalne)</span>
				<select class="select" bind:value={formData.account_wrapper}>
					<option value={null}>Brak</option>
					<option value="IKE">IKE</option>
					<option value="IKZE">IKZE</option>
					<option value="PPK">PPK</option>
				</select>
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Cel konta</span>
				<select class="select" bind:value={formData.purpose} required>
					<option value="general">Ogólne</option>
					<option value="retirement">Emerytura</option>
					<option value="emergency_fund">Fundusz awaryjny</option>
				</select>
			</label>
		</div>

		{#if formData.account_wrapper === 'PPK'}
			<label class="flex items-center gap-2">
				<input type="checkbox" class="checkbox" bind:checked={formData.receives_contributions} />
				<span class="text-sm">Konto otrzymuje wpłaty (aktywne PPK)</span>
			</label>
			<p class="text-xs text-surface-700-300 italic">
				Zaznacz jeśli to konto jest aktywnie używane do otrzymywania miesięcznych wpłat PPK. Stare
				konta PPK powinny mieć to odznaczone - będą tylko śledzone przez snapshoty.
			</p>
		{/if}

		{#if formData.category === 'real_estate'}
			<label class="label">
				<span class="font-semibold text-sm">Powierzchnia (m²)</span>
				<input
					type="number"
					class="input"
					bind:value={formData.square_meters}
					min="0"
					step="0.01"
					placeholder="np. 65.50"
				/>
				<span class="text-xs text-surface-700-300 italic">
					Powierzchnia nieruchomości w metrach kwadratowych. Używana do obliczania metryki "Ile
					metrów mieszkania jest nasze".
				</span>
			</label>
		{/if}
	</form>
</Modal>

<Modal
	open={showDeleteModal}
	title="Potwierdzenie usunięcia"
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
	confirmText="Usuń"
	confirmVariant="danger"
>
	<p class="mb-2">Czy na pewno chcesz usunąć to konto?</p>
	<p class="text-sm text-surface-700-300">Operacja ta ustawi konto jako nieaktywne.</p>
</Modal>

<Modal
	open={showTransactionsModal}
	title="Transakcje - {selectedAccountName}"
	onCancel={closeTransactions}
	hideFooter
>
	<div class="space-y-4">
		{#if transactionError}
			<div class="card preset-filled-error-500 p-3 text-sm">{transactionError}</div>
		{/if}

		{#if transactionsData}
			<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
				<h3 class="h4">Historia transakcji</h3>
				<p class="text-sm text-surface-700-300">
					Zainwestowano łącznie: <strong class="text-primary-600-400"
						>{formatPLN(transactionsData.total_invested)}</strong
					>
				</p>
			</div>

			{#if transactionsData.transactions.length === 0}
				<div class="text-center py-8 text-surface-700-300"><p>Brak transakcji</p></div>
			{:else}
				<div class="table-wrap">
					<table class="table table-hover">
						<thead>
							<tr>
								<th>Data zakupu</th>
								<th>Kwota</th>
								<th>Właściciel</th>
								<th class="text-right">Akcje</th>
							</tr>
						</thead>
						<tbody>
							{#each transactionsData.transactions as transaction}
								<tr>
									<td>{new Date(transaction.date).toLocaleDateString('pl-PL')}</td>
									<td class="font-semibold text-primary-600-400">{formatPLN(transaction.amount)}</td
									>
									<td>{transaction.owner}</td>
									<td class="text-right">
										<button
											type="button"
											class="btn-icon btn-icon-sm"
											aria-label="Usuń"
											on:click={() => deleteTransaction(transaction.id)}
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

			<div class="space-y-3 pt-4 border-t border-surface-200-800">
				<h3 class="h4">Dodaj transakcję</h3>
				<form on:submit|preventDefault={addTransaction} class="space-y-4">
					<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
						<label class="label">
							<span class="font-semibold text-sm">Kwota (PLN)</span>
							<input
								type="number"
								class="input"
								bind:value={transactionFormData.amount}
								required
								min="0.01"
								step="0.01"
								placeholder="np. 5000.00"
							/>
						</label>

						<label class="label">
							<span class="font-semibold text-sm">Data zakupu</span>
							<input
								type="date"
								class="input"
								bind:value={transactionFormData.date}
								required
								max={new Date().toISOString().split('T')[0]}
							/>
						</label>

						<label class="label">
							<span class="font-semibold text-sm">Właściciel</span>
							<select class="select" bind:value={transactionFormData.owner} required>
								{#each personas as persona}
									<option value={persona.name}>{persona.name}</option>
								{/each}
							</select>
						</label>

						{#if selectedAccountWrapper}
							<label class="label">
								<span class="font-semibold text-sm">Typ wpłaty</span>
								<select class="select" bind:value={transactionFormData.transaction_type}>
									<option value="">Wpłata pracownika</option>
									{#if selectedAccountWrapper === 'PPK'}
										<option value="employer">Wpłata pracodawcy</option>
									{/if}
									<option value="withdrawal">Wypłata</option>
								</select>
							</label>
						{/if}
					</div>

					<button type="submit" class="btn preset-filled-primary-500" disabled={savingTransaction}>
						{savingTransaction ? 'Zapisywanie...' : 'Dodaj transakcję'}
					</button>
				</form>
			</div>
		{/if}
	</div>
</Modal>
