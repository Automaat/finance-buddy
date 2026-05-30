<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import SortableTable, { type SortableColumn } from '$lib/components/SortableTable.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { Wallet, TrendingDown, Pencil, Trash2, Plus, BarChart3 } from 'lucide-svelte';
	import { resolveApiUrl } from '$lib/api';
	import { invalidateAll } from '$app/navigation';
	import { onMount, untrack } from 'svelte';
	import { INVESTMENT_CATEGORIES } from '$lib/constants';
	import type { Account, TransactionsData } from './+page';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const apiUrl = resolveApiUrl();
	let owners: OwnerOption[] = $state([]);
	const defaultOwnerUserId = $derived(owners.length > 0 ? owners[0].id : null);

	$effect(() => {
		let cancelled = false;
		Promise.resolve(data.owners).then((p) => {
			if (!cancelled) owners = (p ?? []) as OwnerOption[];
		});
		return () => {
			cancelled = true;
		};
	});

	let showForm = $state(false);
	let editingAccount: Account | null = $state(null);
	let showDeleteModal = $state(false);
	let accountToDelete: number | null = $state(null);
	let transactionCounts: Record<number, number> = $state({});

	let showTransactionsModal = $state(false);
	let selectedAccountId: number | null = $state(null);
	let selectedAccountName = $state('');
	let selectedAccountWrapper: string | null = $state(null);
	let transactionsData: TransactionsData | null = $state(null);
	let transactionFormData = $state({
		amount: 0,
		date: new Date().toISOString().split('T')[0],
		owner_user_id: untrack(() => defaultOwnerUserId),
		transaction_type: 'employee' as string
	});
	let transactionError = $state('');
	let savingTransaction = $state(false);

	interface TransactionTypeOption {
		value: string;
		label: string;
	}
	// Fallback list keeps the dropdown usable if /api/transactions/types
	// fails (network blip, backend down). Matches the canonical Go enum so
	// validation accepts every value the UI offers.
	const FALLBACK_TRANSACTION_TYPES: TransactionTypeOption[] = [
		{ value: 'employee', label: 'Wpłata pracownika' },
		{ value: 'employer', label: 'Wpłata pracodawcy' },
		{ value: 'government', label: 'Dopłata państwa' },
		{ value: 'withdrawal', label: 'Wypłata' }
	];
	let transactionTypes: TransactionTypeOption[] = $state(FALLBACK_TRANSACTION_TYPES);

	// Only PPK accounts accept employer + government contributions; for
	// non-PPK wrappers (IKE, IKZE, stocks…) hide those rows.
	const visibleTransactionTypes = $derived(
		transactionTypes.filter(
			(t) =>
				selectedAccountWrapper === 'PPK' || (t.value !== 'employer' && t.value !== 'government')
		)
	);

	onMount(async () => {
		try {
			const res = await fetch(`${apiUrl}/api/transactions/types`);
			if (res.ok) transactionTypes = (await res.json()) as TransactionTypeOption[];
		} catch (err) {
			console.error('Failed to load transaction types:', err);
		}
	});

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

	const accountColumns = $derived<SortableColumn<Account>[]>([
		{ key: 'name', label: 'Nazwa', sortable: true, accessor: (a) => a.name },
		{
			key: 'category',
			label: 'Kategoria',
			sortable: true,
			accessor: (a) => categoryLabels[a.category] ?? a.category
		},
		{
			key: 'owner',
			label: 'Właściciel',
			sortable: true,
			accessor: (a) => ownerName(owners, a.owner_user_id)
		},
		{ key: 'value', label: 'Wartość', sortable: true, accessor: (a) => a.current_value },
		{
			key: 'real_yield',
			label: 'Realne %',
			sortable: true,
			// Return null for missing rates so SortableTable's null-handling in
			// compareValues keeps rows without a tracked yield at the bottom on
			// asc and at the top on desc — consistent with other null-bearing
			// columns instead of pretending they're the smallest number.
			accessor: (a) => a.real_yield_pct
		},
		{ key: 'actions', label: 'Akcje', align: 'right' }
	]);

	const liabilityColumns = $derived<SortableColumn<Account>[]>([
		{ key: 'name', label: 'Nazwa', sortable: true, accessor: (a) => a.name },
		{
			key: 'category',
			label: 'Kategoria',
			sortable: true,
			accessor: (a) => categoryLabels[a.category] ?? a.category
		},
		{
			key: 'owner',
			label: 'Właściciel',
			sortable: true,
			accessor: (a) => ownerName(owners, a.owner_user_id)
		},
		{ key: 'value', label: 'Wartość', sortable: true, accessor: (a) => a.current_value },
		{ key: 'actions', label: 'Akcje', align: 'right' }
	]);

	function formatPct(v: number): string {
		const sign = v > 0 ? '+' : '';
		return `${sign}${v.toFixed(2)}%`;
	}

	// Color buckets per issue #573 acceptance: green > 1%, amber 0–1%, red < 0%.
	// Exactly 1.00% stays amber so the boundary case lands on the conservative
	// side ("not yet beating inflation comfortably").
	function realYieldColorClass(v: number): string {
		if (v < 0) return 'text-error-600-400';
		if (v <= 1) return 'text-warning-600-400';
		return 'text-success-600-400';
	}

	// Native `title` tooltips collapse newlines in some browsers, so we join
	// the math steps with " · " for reliable single-line rendering. We avoid
	// hardcoding the Belka percentage here so the tooltip doesn't drift when
	// the rate changes year-to-year — the canonical value lives in the
	// backend `rules` table.
	function realYieldTooltip(a: Account): string {
		if (a.interest_rate_pct == null) return '';
		const nominal = a.interest_rate_pct.toFixed(2);
		const shielded = a.account_wrapper === 'IKE' || a.account_wrapper === 'IKZE';
		const belkaPart = shielded
			? `bez podatku Belki (opakowanie ${a.account_wrapper})`
			: 'minus podatek Belki';
		if (a.cpi_yoy_pct == null || a.cpi_as_of_year == null) {
			return `Nominalne ${nominal}% · ${belkaPart} · brak danych CPI`;
		}
		const cpi = a.cpi_yoy_pct.toFixed(2);
		return `Nominalne ${nominal}% · ${belkaPart} · minus CPI ${a.cpi_as_of_year}: ${cpi}%`;
	}

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

	let formData = $state({
		name: '',
		type: 'asset',
		category: 'bank',
		owner_user_id: untrack(() => defaultOwnerUserId),
		currency: 'PLN',
		account_wrapper: null as string | null,
		purpose: 'general',
		receives_contributions: true,
		square_meters: null as number | null,
		excluded_from_fire: false,
		interest_rate_pct: null as number | null
	});

	let error = $state('');
	let saving = $state(false);

	$effect(() => {
		if (editingAccount) {
			formData = {
				name: editingAccount.name,
				type: editingAccount.type,
				category: editingAccount.category,
				owner_user_id: editingAccount.owner_user_id,
				currency: editingAccount.currency,
				account_wrapper: editingAccount.account_wrapper,
				purpose: editingAccount.purpose,
				receives_contributions: editingAccount.receives_contributions,
				square_meters: editingAccount.square_meters,
				excluded_from_fire: editingAccount.excluded_from_fire ?? false,
				interest_rate_pct: editingAccount.interest_rate_pct
			};
		} else if (showForm) {
			formData = {
				name: '',
				type: 'asset',
				category: 'bank',
				owner_user_id: defaultOwnerUserId,
				currency: 'PLN',
				account_wrapper: null,
				purpose: 'general',
				receives_contributions: true,
				square_meters: null,
				excluded_from_fire: false,
				interest_rate_pct: null
			};
		}
	});

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
				const errorBody = await response.json();
				throw new Error(errorBody.detail || 'Failed to save account');
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
			owner_user_id: defaultOwnerUserId,
			transaction_type: 'employee'
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
				owner_user_id: defaultOwnerUserId,
				transaction_type: 'employee'
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

{#snippet assetRow(account: Account)}
	<tr>
		<td class="font-medium" data-label="Nazwa">
			{account.name}
			{#if account.excluded_from_fire}
				<span
					class="chip preset-tonal-surface text-xs ml-1"
					title="Wykluczone z liczby FIRE i powiązanych metryk"
				>
					poza FIRE
				</span>
			{/if}
		</td>
		<td data-label="Kategoria">{categoryLabels[account.category] || account.category}</td>
		<td data-label="Właściciel">{ownerName(owners, account.owner_user_id)}</td>
		<td class="font-semibold text-primary-600-400" data-label="Wartość">
			{formatPLN(account.current_value)}
		</td>
		<td class="whitespace-nowrap" data-label="Realne %">
			{#if account.real_yield_pct != null}
				<span
					class="font-semibold {realYieldColorClass(account.real_yield_pct)}"
					title={realYieldTooltip(account)}
				>
					{formatPct(account.real_yield_pct)}
				</span>
			{:else if account.interest_rate_pct != null}
				<span class="text-surface-700-300" title={realYieldTooltip(account)}>
					{account.interest_rate_pct.toFixed(2)}%
				</span>
			{:else}
				<span class="text-surface-500-500" aria-hidden="true">—</span>
			{/if}
		</td>
		<td class="text-right whitespace-nowrap">
			<button
				type="button"
				class="btn-icon btn-icon-sm"
				aria-label="Edytuj"
				onclick={() => startEdit(account)}
			>
				<Pencil size={16} />
			</button>
			{#if INVESTMENT_CATEGORIES.has(account.category) || account.account_wrapper}
				<button
					type="button"
					class="btn preset-tonal-surface btn-sm gap-1"
					aria-label="Transakcje"
					onclick={() => openTransactions(account.id, account.name, account.account_wrapper)}
				>
					<BarChart3 size={14} />
					<span>{transactionCounts[account.id] || 0}</span>
				</button>
			{/if}
			<button
				type="button"
				class="btn-icon btn-icon-sm"
				aria-label="Usuń"
				onclick={() => handleDelete(account.id)}
			>
				<Trash2 size={16} />
			</button>
		</td>
	</tr>
{/snippet}

{#snippet liabilityRow(account: Account)}
	<tr>
		<td class="font-medium" data-label="Nazwa">{account.name}</td>
		<td data-label="Kategoria">{categoryLabels[account.category] || account.category}</td>
		<td data-label="Właściciel">{ownerName(owners, account.owner_user_id)}</td>
		<td class="font-semibold text-error-600-400" data-label="Wartość">
			{formatPLN(account.current_value)}
		</td>
		<td class="text-right whitespace-nowrap">
			<button
				type="button"
				class="btn-icon btn-icon-sm"
				aria-label="Edytuj"
				onclick={() => startEdit(account)}
			>
				<Pencil size={16} />
			</button>
			<button
				type="button"
				class="btn-icon btn-icon-sm"
				aria-label="Usuń"
				onclick={() => handleDelete(account.id)}
			>
				<Trash2 size={16} />
			</button>
		</td>
	</tr>
{/snippet}

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Konta</h1>
		<p class="text-surface-700-300 text-sm">Zarządzaj kontami aktywów i pasywów</p>
	</div>
	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto gap-2"
		onclick={startCreate}
	>
		<Plus size={16} />
		Nowe Konto
	</button>
</div>

{#await data.accountsData}
	<div role="status" aria-live="polite" aria-label="Ładowanie kont" class="space-y-4">
		{#each [{ icon: Wallet, title: 'Aktywa', isAsset: true }, { icon: TrendingDown, title: 'Pasywa', isAsset: false }] as section}
			<div class="card preset-filled-surface-100-900 p-4 space-y-4">
				<header>
					<h3 class="h3 flex items-center gap-2">
						<section.icon size={20} />
						{section.title}
					</h3>
				</header>
				<div class="table-wrap">
					<table class="table">
						<thead>
							<tr>
								<th>Nazwa</th>
								<th>Kategoria</th>
								<th>Właściciel</th>
								<th>Wartość</th>
								{#if section.isAsset}
									<th>Realne %</th>
								{/if}
								<th class="text-right">Akcje</th>
							</tr>
						</thead>
						<tbody>
							{#each { length: 5 } as _, i (i)}
								<tr>
									<td><Skeleton height="1rem" width="70%" /></td>
									<td><Skeleton height="1rem" width="60%" /></td>
									<td><Skeleton height="1rem" width="50%" /></td>
									<td><Skeleton height="1rem" width="65%" /></td>
									{#if section.isAsset}
										<td><Skeleton height="1rem" width="40%" /></td>
									{/if}
									<td>
										<div class="flex justify-end gap-2">
											<Skeleton height="2rem" width="2rem" rounded="md" />
											<Skeleton height="2rem" width="2rem" rounded="md" />
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/each}
	</div>
{:then accounts}
	<div class="space-y-4">
		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header>
				<h3 class="h3 flex items-center gap-2"><Wallet size={20} /> Aktywa</h3>
			</header>
			{#if accounts.assets.length === 0}
				<div class="text-center py-12 text-surface-700-300"><p>Brak aktywów</p></div>
			{:else}
				<div class="table-cards">
					<SortableTable
						columns={accountColumns}
						items={accounts.assets}
						row={assetRow}
						paramName="sortA"
						getKey={(a) => a.id}
					/>
				</div>
			{/if}
		</div>

		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header>
				<h3 class="h3 flex items-center gap-2"><TrendingDown size={20} /> Pasywa</h3>
			</header>
			{#if accounts.liabilities.length === 0}
				<div class="text-center py-12 text-surface-700-300"><p>Brak pasywów</p></div>
			{:else}
				<div class="table-cards">
					<SortableTable
						columns={liabilityColumns}
						items={accounts.liabilities}
						row={liabilityRow}
						paramName="sortL"
						getKey={(a) => a.id}
					/>
				</div>
			{/if}
		</div>
	</div>
{:catch err}
	<div class="card preset-filled-error-500 p-4">
		<p class="font-semibold">Nie udało się załadować kont.</p>
		<p class="text-sm">{err?.message ?? 'Spróbuj ponownie później.'}</p>
	</div>
{/await}

<Modal
	open={showForm}
	title={editingAccount ? 'Edytuj Konto' : 'Nowe Konto'}
	onConfirm={handleSubmit}
	onCancel={cancelForm}
	confirmText={saving ? 'Zapisywanie...' : editingAccount ? 'Zapisz zmiany' : 'Utwórz konto'}
	confirmDisabled={saving}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			handleSubmit();
		}}
		class="space-y-4"
	>
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
				<select class="select" bind:value={formData.owner_user_id}>
					<option value={null}>Wspólne</option>
					{#each owners as owner}
						<option value={owner.id}>{owner.name}</option>
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

		{#if formData.type === 'asset' && (formData.category === 'bank' || formData.category === 'saving_account' || formData.category === 'bond')}
			<label class="label">
				<span class="font-semibold text-sm">Oprocentowanie nominalne (%)</span>
				<input
					type="number"
					class="input"
					bind:value={formData.interest_rate_pct}
					min="0"
					max="50"
					step="0.01"
					placeholder="np. 5.35"
				/>
				<span class="text-xs text-surface-700-300 italic">
					Roczna stopa nominalna. Tabela kont wyliczy realną stopę po inflacji i 19% Belce (gdy nie
					w opakowaniu IKE/IKZE).
				</span>
			</label>
		{/if}

		{#if formData.type === 'asset'}
			<label class="flex items-center gap-2">
				<input type="checkbox" class="checkbox" bind:checked={formData.excluded_from_fire} />
				<span class="text-sm">Wyklucz z FIRE</span>
			</label>
			<p class="text-xs text-surface-700-300 italic">
				Wartość tego konta nie wlicza się do liczby FIRE ani powiązanych metryk (Coast / Barista /
				Lean / Fat / Bridge / projekcja). Zaznacz dla mieszkania, w którym mieszkasz, lub innych
				aktywów, których nie będziesz spieniężać na emeryturze.
			</p>
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
									<td>{ownerName(owners, transaction.owner_user_id)}</td>
									<td class="text-right">
										<button
											type="button"
											class="btn-icon btn-icon-sm"
											aria-label="Usuń"
											onclick={() => deleteTransaction(transaction.id)}
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
				<form
					onsubmit={(event) => {
						event.preventDefault();
						addTransaction();
					}}
					class="space-y-4"
				>
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
							<select class="select" bind:value={transactionFormData.owner_user_id}>
								<option value={null}>Wspólne</option>
								{#each owners as owner}
									<option value={owner.id}>{owner.name}</option>
								{/each}
							</select>
						</label>

						{#if selectedAccountWrapper}
							<label class="label">
								<span class="font-semibold text-sm">Typ wpłaty</span>
								<select class="select" bind:value={transactionFormData.transaction_type}>
									{#each visibleTransactionTypes as t}
										<option value={t.value}>{t.label}</option>
									{/each}
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
