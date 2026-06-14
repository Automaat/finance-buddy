<script lang="ts">
	import { untrack } from 'svelte';
	import { goto, invalidateAll } from '$app/navigation';
	import { resolveApiUrl } from '$lib/api';
	import { Wallet, Umbrella, TrendingUp, Home, CreditCard } from 'lucide-svelte';
	import type { Account, Asset, SnapshotResponse } from '$lib/types';
	import type { OwnerOption } from '$lib/types/owners';
	import { categoryLabel } from '$lib/utils/categories';
	import { round2, staleQuotes, type HoldingQuote } from '$lib/utils/quoteFreshness';
	import NewAccountModal from './snapshot/NewAccountModal.svelte';
	import NewAssetModal from './snapshot/NewAssetModal.svelte';
	import ValueRow from './snapshot/ValueRow.svelte';

	interface Props {
		editingSnapshot?: SnapshotResponse | null;
		assets: Account[];
		liabilities: Account[];
		physicalAssets: Asset[];
		owners?: OwnerOption[];
		holdings?: HoldingQuote[];
	}

	let {
		editingSnapshot = null,
		assets: assetsProp,
		liabilities: liabilitiesProp,
		physicalAssets: physicalAssetsProp,
		owners = [],
		holdings = []
	}: Props = $props();

	// Quote-freshness: investment autocalc uses the latest stored price quote.
	// Flag held positions whose quote is missing or older than this many days
	// so the user can refresh before snapshotting.
	const STALE_QUOTE_DAYS = 2;

	const staleHoldings = $derived(
		editingSnapshot ? [] : staleQuotes(holdings, Date.now(), STALE_QUOTE_DAYS)
	);

	let refreshingPrices = $state(false);

	async function refreshPrices() {
		refreshingPrices = true;
		error = '';
		try {
			const response = await fetch(`${resolveApiUrl()}/api/holdings/refresh-quotes`, {
				method: 'POST'
			});
			if (!response.ok) {
				const errorData = await response.json().catch(() => null);
				const message =
					(errorData &&
						typeof errorData === 'object' &&
						'detail' in errorData &&
						errorData.detail) ||
					'Nie udało się zaktualizować cen';
				throw new Error(String(message));
			}
			// Re-run load(): re-fetches accounts (live values recomputed with the
			// fresh quotes) and holdings; the populate effect re-fills the form.
			await invalidateAll();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Błąd aktualizacji cen';
		} finally {
			refreshingPrices = false;
		}
	}

	// Accounts/assets created in-component, merged with the prop-provided ones
	let addedAccounts = $state<Account[]>([]);
	let addedLiabilities = $state<Account[]>([]);
	let addedAssets = $state<Asset[]>([]);

	const assets = $derived([...assetsProp, ...addedAccounts]);
	const liabilities = $derived([...liabilitiesProp, ...addedLiabilities]);
	const physicalAssets = $derived([...physicalAssetsProp, ...addedAssets]);

	// Initialize form state
	let snapshotDate = $state(new Date().toISOString().split('T')[0]);
	let notes = $state('');
	let loading = $state(false);
	let error = $state('');

	// Section grouping
	const RETIREMENT_WRAPPERS = ['PPK', 'IKE', 'IKZE'] as const;
	const INVESTMENT_CATEGORIES = ['stock', 'bond', 'fund', 'etf', 'gold'];
	const FINANCIAL_CATEGORIES = ['bank', 'saving_account'];
	const MAJATEK_CATEGORIES = ['real_estate', 'vehicle', 'other'];

	const retirementAccounts = $derived(assets.filter((a) => a.account_wrapper));
	const investmentAccounts = $derived(
		assets.filter((a) => !a.account_wrapper && INVESTMENT_CATEGORIES.includes(a.category))
	);
	const financialAccounts = $derived(
		assets.filter((a) => !a.account_wrapper && FINANCIAL_CATEGORIES.includes(a.category))
	);
	const majatekAccounts = $derived(
		assets.filter((a) => !a.account_wrapper && MAJATEK_CATEGORIES.includes(a.category))
	);

	const retirementByWrapper = $derived(
		RETIREMENT_WRAPPERS.map((w) => ({
			wrapper: w,
			accounts: retirementAccounts.filter((a) => a.account_wrapper === w)
		})).filter((g) => g.accounts.length > 0)
	);

	// Track visible accounts and assets
	let visibleAccountIds = $state(new Set<number>());
	let visibleAssetIds = $state(new Set<number>());

	// Initialize values
	let accountValues: Record<number, number> = $state({});
	let assetValues: Record<number, number> = $state({});

	// Populate form from editingSnapshot or defaults
	function populateForm() {
		const nextAccountValues: Record<number, number> = {};
		const nextAssetValues: Record<number, number> = {};
		const nextVisibleAccountIds = new Set<number>();
		const nextVisibleAssetIds = new Set<number>();

		if (editingSnapshot) {
			snapshotDate = editingSnapshot.date;
			notes = editingSnapshot.notes || '';

			// Populate from snapshot values
			editingSnapshot.values.forEach((v) => {
				if (v.account_id !== null && v.account_id !== undefined) {
					nextAccountValues[v.account_id] = v.value;
					nextVisibleAccountIds.add(v.account_id);
				}
				if (v.asset_id !== null && v.asset_id !== undefined) {
					nextAssetValues[v.asset_id] = v.value;
					nextVisibleAssetIds.add(v.asset_id);
				}
			});
		} else {
			// Create mode - initialize from current values. Auto-calculated
			// investment values carry many decimals (qty × price × FX); round
			// to 2 places to match the numeric(15,2) column and the input's
			// step="0.01" so the form passes native number validation.
			[...assets, ...liabilities].forEach((account) => {
				nextAccountValues[account.id] = round2(account.current_value);
				if (account.current_value > 0) nextVisibleAccountIds.add(account.id);
			});
			physicalAssets.forEach((asset) => {
				nextAssetValues[asset.id] = round2(asset.current_value);
				if (asset.current_value > 0) nextVisibleAssetIds.add(asset.id);
			});
		}

		accountValues = nextAccountValues;
		assetValues = nextAssetValues;
		visibleAccountIds = nextVisibleAccountIds;
		visibleAssetIds = nextVisibleAssetIds;
	}

	// Reactive population
	$effect(() => {
		if (editingSnapshot || assets || liabilities || physicalAssets) {
			populateForm();
		}
	});

	function removeAccount(accountId: number) {
		visibleAccountIds.delete(accountId);
		visibleAccountIds = new Set(visibleAccountIds);
	}

	function addAccount(accountId: number) {
		visibleAccountIds.add(accountId);
		visibleAccountIds = new Set(visibleAccountIds);
	}

	function removeAsset(assetId: number) {
		visibleAssetIds.delete(assetId);
		visibleAssetIds = new Set(visibleAssetIds);
	}

	function addAsset(assetId: number) {
		visibleAssetIds.add(assetId);
		visibleAssetIds = new Set(visibleAssetIds);
	}

	type NewAccountSection = 'financial' | 'retirement' | 'investment' | 'majatek' | 'liabilities';

	// New item creation state
	let showNewAccountForm = $state(false);
	let showNewAssetForm = $state(false);
	let newAccountSection: NewAccountSection = $state('financial');
	let newAccountName = $state('');
	let newAccountCategory = $state('');
	let newAccountWrapper: '' | 'PPK' | 'IKE' | 'IKZE' = $state('');
	let newAccountOwnerUserId = $state<number | null>(
		untrack(() => (owners.length > 0 ? owners[0].id : null))
	);
	let newAccountValue = $state(0);
	let creatingAccount = $state(false);
	let newAssetName = $state('');
	let newAssetValue = $state(0);
	let creatingAsset = $state(false);

	async function createNewAccount() {
		if (!newAccountName.trim()) {
			error = 'Nazwa konta jest wymagana';
			return;
		}

		creatingAccount = true;
		error = '';

		try {
			const type = newAccountSection === 'liabilities' ? 'liability' : 'asset';
			const category = newAccountCategory || 'other';
			const account_wrapper =
				newAccountSection === 'retirement' && newAccountWrapper ? newAccountWrapper : null;
			const purpose = newAccountSection === 'retirement' ? 'retirement' : 'general';

			const response = await fetch(`${resolveApiUrl()}/api/accounts`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name: newAccountName,
					type,
					category,
					owner_user_id: newAccountOwnerUserId,
					currency: 'PLN',
					account_wrapper,
					purpose
				})
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => null);
				const message =
					(errorData &&
						typeof errorData === 'object' &&
						'detail' in errorData &&
						errorData.detail) ||
					'Failed to create account';
				throw new Error(String(message));
			}

			const newAccount: Account = await response.json();

			if (newAccountSection === 'liabilities') {
				addedLiabilities = [...addedLiabilities, newAccount];
			} else {
				addedAccounts = [...addedAccounts, newAccount];
			}

			// Set initial value and mark as visible
			accountValues[newAccount.id] = newAccountValue;
			visibleAccountIds.add(newAccount.id);
			visibleAccountIds = new Set(visibleAccountIds);

			// Reset form
			showNewAccountForm = false;
			newAccountName = '';
			newAccountCategory = '';
			newAccountWrapper = '';
			newAccountOwnerUserId = owners.length > 0 ? owners[0].id : null;
			newAccountValue = 0;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Błąd tworzenia konta';
		} finally {
			creatingAccount = false;
		}
	}

	function openNewAccountForm(section: NewAccountSection) {
		newAccountSection = section;
		newAccountWrapper = '';
		if (section === 'financial') {
			newAccountCategory = 'bank';
		} else if (section === 'retirement') {
			newAccountCategory = 'stock';
			newAccountWrapper = 'IKE';
		} else if (section === 'investment') {
			newAccountCategory = 'stock';
		} else if (section === 'majatek') {
			newAccountCategory = 'real_estate';
		} else {
			newAccountCategory = 'mortgage';
		}
		showNewAccountForm = true;
	}

	async function createNewAsset() {
		if (!newAssetName.trim()) {
			error = 'Nazwa majątku jest wymagana';
			return;
		}

		creatingAsset = true;
		error = '';

		try {
			const response = await fetch(`${resolveApiUrl()}/api/assets`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name: newAssetName
				})
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => null);
				const message =
					(errorData &&
						typeof errorData === 'object' &&
						'detail' in errorData &&
						errorData.detail) ||
					'Failed to create asset';
				throw new Error(String(message));
			}

			const newAsset: Asset = await response.json();

			addedAssets = [...addedAssets, newAsset];

			assetValues[newAsset.id] = newAssetValue;
			visibleAssetIds.add(newAsset.id);
			visibleAssetIds = new Set(visibleAssetIds);

			showNewAssetForm = false;
			newAssetName = '';
			newAssetValue = 0;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Błąd tworzenia majątku';
		} finally {
			creatingAsset = false;
		}
	}

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		loading = true;
		error = '';

		try {
			const allAccounts = [...assets, ...liabilities];
			const accountPayloads = editingSnapshot
				? allAccounts
						.filter((account) => visibleAccountIds.has(account.id))
						.map((account) => {
							const value = accountValues[account.id];
							const parsedValue = parseFloat(value.toString());
							if (Number.isNaN(parsedValue)) {
								throw new Error('Invalid value for account. Please enter a valid number.');
							}
							return {
								account_id: account.id,
								value: parsedValue
							};
						})
				: allAccounts.map((account) => {
						const isVisible = visibleAccountIds.has(account.id);
						const value = isVisible ? accountValues[account.id] : 0;
						const parsedValue = parseFloat(value.toString());
						if (Number.isNaN(parsedValue)) {
							throw new Error('Invalid value for account. Please enter a valid number.');
						}
						return {
							account_id: account.id,
							value: parsedValue
						};
					});

			const assetPayloads = editingSnapshot
				? physicalAssets
						.filter((asset) => visibleAssetIds.has(asset.id))
						.map((asset) => {
							const value = assetValues[asset.id];
							const parsedValue = parseFloat(value.toString());
							if (Number.isNaN(parsedValue)) {
								throw new Error('Invalid value for asset. Please enter a valid number.');
							}
							return {
								asset_id: asset.id,
								value: parsedValue
							};
						})
				: physicalAssets.map((asset) => {
						const isVisible = visibleAssetIds.has(asset.id);
						const value = isVisible ? assetValues[asset.id] : 0;
						const parsedValue = parseFloat(value.toString());
						if (Number.isNaN(parsedValue)) {
							throw new Error('Invalid value for asset. Please enter a valid number.');
						}
						return {
							asset_id: asset.id,
							value: parsedValue
						};
					});

			const payload = {
				date: snapshotDate,
				notes: notes || null,
				values: [...accountPayloads, ...assetPayloads]
			};

			const method = editingSnapshot ? 'PUT' : 'POST';
			const url = editingSnapshot
				? `${resolveApiUrl()}/api/snapshots/${editingSnapshot.id}`
				: `${resolveApiUrl()}/api/snapshots`;

			const response = await fetch(url, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.detail || 'Failed to save snapshot');
			}

			goto('/');
		} catch (err) {
			error = err instanceof Error ? err.message : 'An error occurred';
		} finally {
			loading = false;
		}
	}

	function accountMeta(account: Account): string {
		return `(${categoryLabel(account.category)})`;
	}

	function liabilityMeta(account: Account): string {
		return `(${categoryLabel(account.category)})`;
	}

	function closeNewAccountModal() {
		showNewAccountForm = false;
	}

	function closeNewAssetModal() {
		showNewAssetForm = false;
	}
</script>

<form onsubmit={handleSubmit} class="max-w-[800px] flex flex-col gap-6">
	<!-- Quote freshness warning -->
	{#if staleHoldings.length > 0}
		<div class="card preset-filled-warning-500 p-4 space-y-2">
			<p class="text-sm font-semibold">⚠️ Notowania inwestycji mogą być nieaktualne</p>
			<ul class="text-sm list-disc list-inside space-y-0.5">
				{#each staleHoldings as q}
					<li>
						{q.name}:
						{#if q.date}
							{q.date} ({q.daysOld} dni temu)
						{:else}
							brak notowania
						{/if}
					</li>
				{/each}
			</ul>
			<button
				type="button"
				class="btn btn-sm preset-filled-surface-50-950"
				onclick={refreshPrices}
				disabled={refreshingPrices}
			>
				{refreshingPrices ? 'Aktualizowanie...' : '🔄 Aktualizuj ceny'}
			</button>
		</div>
	{/if}

	<!-- Date & Notes -->
	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="space-y-1">
			<h3 class="h3">Data Snapshot</h3>
		</header>
		<div class="space-y-4">
			<div class="flex flex-col gap-1">
				<label for="date" class="text-sm font-semibold">Data</label>
				<input id="date" type="date" bind:value={snapshotDate} required class="input" />
			</div>

			<div class="flex flex-col gap-1">
				<label for="notes" class="text-sm font-semibold">Notatki (opcjonalne)</label>
				<input
					id="notes"
					type="text"
					bind:value={notes}
					placeholder="Dodaj notatkę..."
					class="input"
				/>
			</div>
		</div>
	</div>

	<!-- Konta finansowe -->
	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="space-y-1">
			<h3 class="h3 flex items-center gap-2"><Wallet size={20} /> Konta finansowe</h3>
		</header>
		<div class="space-y-4">
			{#each financialAccounts.filter((a) => visibleAccountIds.has(a.id)) as account}
				<ValueRow
					inputId="account-{account.id}"
					label={account.name}
					meta={accountMeta(account)}
					bind:value={accountValues[account.id]}
					onRemove={() => removeAccount(account.id)}
				/>
			{/each}

			<div class="mt-4 p-3 border border-dashed border-surface-300-700 rounded-container">
				{#if financialAccounts.filter((a) => !visibleAccountIds.has(a.id)).length > 0}
					<details>
						<summary
							class="cursor-pointer text-primary-500 font-semibold text-sm select-none hover:text-primary-600-400"
						>
							+ Pokaż ukryte konta
						</summary>
						<div class="flex flex-col gap-2 mt-2">
							{#each financialAccounts.filter((a) => !visibleAccountIds.has(a.id)) as account}
								<button
									type="button"
									class="btn preset-tonal-surface w-full text-left text-sm"
									onclick={() => addAccount(account.id)}
								>
									{account.name}
									<span class="text-surface-600-400 font-normal ml-1"
										>({categoryLabel(account.category)})</span
									>
								</button>
							{/each}
						</div>
					</details>
				{/if}
				<button
					type="button"
					class="btn w-full mt-2 border-2 border-dashed border-primary-500 text-primary-500 hover:preset-filled-primary-500"
					onclick={() => openNewAccountForm('financial')}
				>
					+ Dodaj nowe konto
				</button>
			</div>
		</div>
	</div>

	<!-- Emerytura -->
	{#if retirementByWrapper.length > 0}
		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header class="space-y-1">
				<h3 class="h3 flex items-center gap-2"><Umbrella size={20} /> Emerytura</h3>
			</header>
			<div class="space-y-4">
				{#each retirementByWrapper as group}
					<h3
						class="text-sm font-bold text-surface-600-400 uppercase tracking-wide mt-4 mb-2 pb-1 border-b border-surface-200-800"
					>
						{group.wrapper}
					</h3>
					{#each group.accounts.filter((a) => visibleAccountIds.has(a.id)) as account}
						<ValueRow
							inputId="account-{account.id}"
							label={account.name}
							meta={accountMeta(account)}
							bind:value={accountValues[account.id]}
							onRemove={() => removeAccount(account.id)}
						/>
					{/each}
				{/each}

				<div class="mt-4 p-3 border border-dashed border-surface-300-700 rounded-container">
					{#if retirementAccounts.filter((a) => !visibleAccountIds.has(a.id)).length > 0}
						<details>
							<summary
								class="cursor-pointer text-primary-500 font-semibold text-sm select-none hover:text-primary-600-400"
							>
								+ Pokaż ukryte konta
							</summary>
							<div class="flex flex-col gap-2 mt-2">
								{#each retirementAccounts.filter((a) => !visibleAccountIds.has(a.id)) as account}
									<button
										type="button"
										class="btn preset-tonal-surface w-full text-left text-sm"
										onclick={() => addAccount(account.id)}
									>
										{account.name}
										<span class="text-surface-600-400 font-normal ml-1"
											>({account.account_wrapper})</span
										>
									</button>
								{/each}
							</div>
						</details>
					{/if}
					<button
						type="button"
						class="btn w-full mt-2 border-2 border-dashed border-primary-500 text-primary-500 hover:preset-filled-primary-500"
						onclick={() => openNewAccountForm('retirement')}
					>
						+ Dodaj nowe konto
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Inwestycje -->
	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="space-y-1">
			<h3 class="h3 flex items-center gap-2"><TrendingUp size={20} /> Inwestycje</h3>
		</header>
		<div class="space-y-4">
			{#each investmentAccounts.filter((a) => visibleAccountIds.has(a.id)) as account}
				<ValueRow
					inputId="account-{account.id}"
					label={account.name}
					meta={accountMeta(account)}
					bind:value={accountValues[account.id]}
					onRemove={() => removeAccount(account.id)}
				/>
			{/each}

			<div class="mt-4 p-3 border border-dashed border-surface-300-700 rounded-container">
				{#if investmentAccounts.filter((a) => !visibleAccountIds.has(a.id)).length > 0}
					<details>
						<summary
							class="cursor-pointer text-primary-500 font-semibold text-sm select-none hover:text-primary-600-400"
						>
							+ Pokaż ukryte konta
						</summary>
						<div class="flex flex-col gap-2 mt-2">
							{#each investmentAccounts.filter((a) => !visibleAccountIds.has(a.id)) as account}
								<button
									type="button"
									class="btn preset-tonal-surface w-full text-left text-sm"
									onclick={() => addAccount(account.id)}
								>
									{account.name}
									<span class="text-surface-600-400 font-normal ml-1"
										>({categoryLabel(account.category)})</span
									>
								</button>
							{/each}
						</div>
					</details>
				{/if}
				<button
					type="button"
					class="btn w-full mt-2 border-2 border-dashed border-primary-500 text-primary-500 hover:preset-filled-primary-500"
					onclick={() => openNewAccountForm('investment')}
				>
					+ Dodaj nowe konto
				</button>
			</div>
		</div>
	</div>

	<!-- Majątek -->
	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="space-y-1">
			<h3 class="h3 flex items-center gap-2"><Home size={20} /> Majątek</h3>
		</header>
		<div class="space-y-4">
			{#each majatekAccounts.filter((a) => visibleAccountIds.has(a.id)) as account}
				<ValueRow
					inputId="account-{account.id}"
					label={account.name}
					meta={accountMeta(account)}
					bind:value={accountValues[account.id]}
					onRemove={() => removeAccount(account.id)}
				/>
			{/each}

			{#each physicalAssets.filter((a) => visibleAssetIds.has(a.id)) as asset}
				<ValueRow
					inputId="asset-{asset.id}"
					label={asset.name}
					bind:value={assetValues[asset.id]}
					onRemove={() => removeAsset(asset.id)}
				/>
			{/each}

			<div class="mt-4 p-3 border border-dashed border-surface-300-700 rounded-container">
				{#if majatekAccounts.filter((a) => !visibleAccountIds.has(a.id)).length > 0 || physicalAssets.filter((a) => !visibleAssetIds.has(a.id)).length > 0}
					<details>
						<summary
							class="cursor-pointer text-primary-500 font-semibold text-sm select-none hover:text-primary-600-400"
						>
							+ Pokaż ukryty majątek
						</summary>
						<div class="flex flex-col gap-2 mt-2">
							{#each majatekAccounts.filter((a) => !visibleAccountIds.has(a.id)) as account}
								<button
									type="button"
									class="btn preset-tonal-surface w-full text-left text-sm"
									onclick={() => addAccount(account.id)}
								>
									{account.name}
									<span class="text-surface-600-400 font-normal ml-1"
										>({categoryLabel(account.category)})</span
									>
								</button>
							{/each}
							{#each physicalAssets.filter((a) => !visibleAssetIds.has(a.id)) as asset}
								<button
									type="button"
									class="btn preset-tonal-surface w-full text-left text-sm"
									onclick={() => addAsset(asset.id)}
								>
									{asset.name}
								</button>
							{/each}
						</div>
					</details>
				{/if}
				<button
					type="button"
					class="btn w-full mt-2 border-2 border-dashed border-primary-500 text-primary-500 hover:preset-filled-primary-500"
					onclick={() => openNewAccountForm('majatek')}
				>
					+ Dodaj nowe konto
				</button>
				<button
					type="button"
					class="btn w-full mt-2 border-2 border-dashed border-primary-500 text-primary-500 hover:preset-filled-primary-500"
					onclick={() => (showNewAssetForm = true)}
				>
					+ Dodaj nowy majątek
				</button>
			</div>
		</div>
	</div>

	<!-- Liabilities -->
	{#if liabilities.length > 0}
		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header class="space-y-1">
				<h3 class="h3 flex items-center gap-2"><CreditCard size={20} /> Zobowiązania</h3>
			</header>
			<div class="space-y-4">
				{#each liabilities.filter((a) => visibleAccountIds.has(a.id)) as account}
					<ValueRow
						inputId="account-{account.id}"
						label={account.name}
						meta={liabilityMeta(account)}
						bind:value={accountValues[account.id]}
						onRemove={() => removeAccount(account.id)}
					/>
				{/each}

				<div class="mt-4 p-3 border border-dashed border-surface-300-700 rounded-container">
					{#if liabilities.filter((a) => !visibleAccountIds.has(a.id)).length > 0}
						<details>
							<summary
								class="cursor-pointer text-primary-500 font-semibold text-sm select-none hover:text-primary-600-400"
							>
								+ Pokaż ukryte konta
							</summary>
							<div class="flex flex-col gap-2 mt-2">
								{#each liabilities.filter((a) => !visibleAccountIds.has(a.id)) as account}
									<button
										type="button"
										class="btn preset-tonal-surface w-full text-left text-sm"
										onclick={() => addAccount(account.id)}
									>
										{account.name}
										<span class="text-surface-600-400 font-normal ml-1"
											>({categoryLabel(account.category)})</span
										>
									</button>
								{/each}
							</div>
						</details>
					{/if}
					<button
						type="button"
						class="btn w-full mt-2 border-2 border-dashed border-primary-500 text-primary-500 hover:preset-filled-primary-500"
						onclick={() => openNewAccountForm('liabilities')}
					>
						+ Dodaj nowe konto
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- New Account Modal -->
	{#if showNewAccountForm}
		<NewAccountModal
			section={newAccountSection}
			bind:name={newAccountName}
			bind:category={newAccountCategory}
			bind:wrapper={newAccountWrapper}
			bind:ownerUserId={newAccountOwnerUserId}
			bind:value={newAccountValue}
			creating={creatingAccount}
			{owners}
			onCreate={createNewAccount}
			onClose={closeNewAccountModal}
		/>
	{/if}

	<!-- New Asset Modal -->
	{#if showNewAssetForm}
		<NewAssetModal
			bind:name={newAssetName}
			bind:value={newAssetValue}
			creating={creatingAsset}
			onCreate={createNewAsset}
			onClose={closeNewAssetModal}
		/>
	{/if}

	<!-- Error Message -->
	{#if error}
		<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
	{/if}

	<!-- Submit Buttons -->
	<div class="flex flex-col-reverse sm:flex-row gap-3">
		<button type="submit" disabled={loading} class="btn preset-filled-primary-500 sm:flex-1 w-full">
			{loading ? 'Zapisywanie...' : '💾 Zapisz Snapshot'}
		</button>
		<button
			type="button"
			onclick={() => goto('/')}
			class="btn preset-tonal-surface w-full sm:w-auto"
		>
			Anuluj
		</button>
	</div>
</form>
