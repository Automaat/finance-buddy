<script lang="ts">
	import { goto } from '$app/navigation';
	import { env } from '$env/dynamic/public';
	import { Card, CardHeader, CardTitle, CardContent } from '@mskalski/home-ui';
	import type { SnapshotResponse } from '$lib/types';

	export let editingSnapshot: SnapshotResponse | null = null;
	export let assets: any[];
	export let liabilities: any[];
	export let physicalAssets: any[];

	// Initialize form state
	let snapshotDate = new Date().toISOString().split('T')[0];
	let notes = '';
	let loading = false;
	let error = '';

	// Separate investments from regular assets
	const investmentCategories = ['ike', 'ikze', 'ppk', 'bonds', 'stocks', 'fund', 'etf'];
	$: investments = assets.filter((a: any) => investmentCategories.includes(a.category));
	$: regularAssets = assets.filter((a: any) => !investmentCategories.includes(a.category));

	// Track visible accounts and assets
	let visibleAccountIds = new Set<number>();
	let visibleAssetIds = new Set<number>();

	// Initialize values
	let accountValues: Record<number, number> = {};
	let assetValues: Record<number, number> = {};

	// Populate form from editingSnapshot or defaults
	function populateForm() {
		if (editingSnapshot) {
			snapshotDate = editingSnapshot.date;
			notes = editingSnapshot.notes || '';

			// Reset values
			accountValues = {};
			assetValues = {};
			visibleAccountIds = new Set();
			visibleAssetIds = new Set();

			// Populate from snapshot values
			editingSnapshot.values.forEach((v) => {
				if (v.account_id !== null && v.account_id !== undefined) {
					accountValues[v.account_id] = v.value;
					visibleAccountIds.add(v.account_id);
				}
				if (v.asset_id !== null && v.asset_id !== undefined) {
					assetValues[v.asset_id] = v.value;
					visibleAssetIds.add(v.asset_id);
				}
			});
			visibleAccountIds = new Set(visibleAccountIds);
			visibleAssetIds = new Set(visibleAssetIds);
		} else {
			// Create mode - initialize from current values
			visibleAccountIds = new Set(
				[...assets, ...liabilities].filter((a: any) => a.current_value > 0).map((a: any) => a.id)
			);
			visibleAssetIds = new Set(
				physicalAssets.filter((a: any) => a.current_value > 0).map((a: any) => a.id)
			);

			[...assets, ...liabilities].forEach((account) => {
				accountValues[account.id] = account.current_value;
			});
			physicalAssets.forEach((asset: any) => {
				assetValues[asset.id] = asset.current_value;
			});
		}
	}

	// Reactive population
	$: if (editingSnapshot || assets || liabilities || physicalAssets) {
		populateForm();
	}

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

	// New item creation state
	let showNewAccountForm = false;
	let showNewAssetForm = false;
	let newAccountSection: 'investments' | 'assets' | 'liabilities' = 'assets';
	let newAccountName = '';
	let newAccountCategory = '';
	let newAccountOwner = 'Shared';
	let newAccountValue = 0;
	let creatingAccount = false;
	let newAssetName = '';
	let newAssetValue = 0;
	let creatingAsset = false;

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

			const response = await fetch(`${env.PUBLIC_API_URL_BROWSER}/api/accounts`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name: newAccountName,
					type,
					category,
					owner: newAccountOwner,
					currency: 'PLN'
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

			const newAccount = await response.json();

			// Add to appropriate section
			if (newAccountSection === 'investments') {
				investments = [...investments, newAccount];
			} else if (newAccountSection === 'assets') {
				regularAssets = [...regularAssets, newAccount];
			} else {
				liabilities = [...liabilities, newAccount];
			}

			// Set initial value and mark as visible
			accountValues[newAccount.id] = newAccountValue;
			visibleAccountIds.add(newAccount.id);
			visibleAccountIds = new Set(visibleAccountIds);

			// Reset form
			showNewAccountForm = false;
			newAccountName = '';
			newAccountCategory = '';
			newAccountOwner = 'Shared';
			newAccountValue = 0;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Błąd tworzenia konta';
		} finally {
			creatingAccount = false;
		}
	}

	function openNewAccountForm(section: 'investments' | 'assets' | 'liabilities') {
		newAccountSection = section;
		if (section === 'investments') {
			newAccountCategory = 'ike';
		} else if (section === 'assets') {
			newAccountCategory = 'bank';
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
			const response = await fetch(`${env.PUBLIC_API_URL_BROWSER}/api/assets`, {
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

			const newAsset = await response.json();

			physicalAssets = [...physicalAssets, newAsset];

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

	async function handleSubmit() {
		loading = true;
		error = '';

		try {
			const allAccounts = [...investments, ...regularAssets, ...liabilities];
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
						.filter((asset: any) => visibleAssetIds.has(asset.id))
						.map((asset: any) => {
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
				: physicalAssets.map((asset: any) => {
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
				? `${env.PUBLIC_API_URL_BROWSER}/api/snapshots/${editingSnapshot.id}`
				: `${env.PUBLIC_API_URL_BROWSER}/api/snapshots`;

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

	const categoryLabels: Record<string, string> = {
		bank: 'Konto bankowe',
		ike: 'IKE',
		ikze: 'IKZE',
		ppk: 'PPK',
		bonds: 'Obligacje',
		stocks: 'Akcje',
		real_estate: 'Nieruchomości',
		vehicle: 'Pojazd',
		mortgage: 'Hipoteka',
		installment: 'Raty',
		other: 'Inne',
		fund: 'Fundusz',
		etf: 'ETF'
	};
</script>

<form on:submit|preventDefault={handleSubmit} class="snapshot-form">
	<!-- Date & Notes -->
	<Card>
		<CardHeader>
			<CardTitle>Data Snapshot</CardTitle>
		</CardHeader>
		<CardContent>
			<div class="form-group">
				<label for="date" class="form-label">Data</label>
				<input id="date" type="date" bind:value={snapshotDate} required class="form-input" />
			</div>

			<div class="form-group">
				<label for="notes" class="form-label">Notatki (opcjonalne)</label>
				<input
					id="notes"
					type="text"
					bind:value={notes}
					placeholder="Dodaj notatkę..."
					class="form-input"
				/>
			</div>
		</CardContent>
	</Card>

	<!-- Investments -->
	{#if investments.length > 0}
		<Card>
			<CardHeader>
				<CardTitle>📈 Inwestycje</CardTitle>
			</CardHeader>
			<CardContent>
				{#each investments.filter((a: any) => visibleAccountIds.has(a.id)) as account}
					<div class="form-group-with-remove">
						<div class="form-group">
							<label for="account-{account.id}" class="form-label">
								{account.name}
								<span class="label-meta"
									>({categoryLabels[account.category] || account.category})</span
								>
							</label>
							<input
								id="account-{account.id}"
								type="number"
								step="0.01"
								bind:value={accountValues[account.id]}
								placeholder="0.00"
								class="form-input"
							/>
						</div>
						<button
							type="button"
							class="btn-remove"
							on:click={() => removeAccount(account.id)}
							title="Usuń pole"
						>
							×
						</button>
					</div>
				{/each}

				<div class="add-account">
					{#if investments.filter((a: any) => !visibleAccountIds.has(a.id)).length > 0}
						<details>
							<summary>+ Pokaż ukryte konta</summary>
							<div class="add-account-list">
								{#each investments.filter((a: any) => !visibleAccountIds.has(a.id)) as account}
									<button
										type="button"
										class="btn-add-account"
										on:click={() => addAccount(account.id)}
									>
										{account.name}
										<span class="label-meta"
											>({categoryLabels[account.category] || account.category})</span
										>
									</button>
								{/each}
							</div>
						</details>
					{/if}
					<button
						type="button"
						class="btn-new-account"
						on:click={() => openNewAccountForm('investments')}
					>
						+ Dodaj nowe konto
					</button>
				</div>
			</CardContent>
		</Card>
	{/if}

	<!-- Regular Assets -->
	<Card>
		<CardHeader>
			<CardTitle>💰 Konta finansowe</CardTitle>
		</CardHeader>
		<CardContent>
			{#each regularAssets.filter((a: any) => visibleAccountIds.has(a.id)) as account}
				<div class="form-group-with-remove">
					<div class="form-group">
						<label for="account-{account.id}" class="form-label">
							{account.name}
							<span class="label-meta">({categoryLabels[account.category]})</span>
						</label>
						<input
							id="account-{account.id}"
							type="number"
							step="0.01"
							bind:value={accountValues[account.id]}
							placeholder="0.00"
							class="form-input"
						/>
					</div>
					<button
						type="button"
						class="btn-remove"
						on:click={() => removeAccount(account.id)}
						title="Usuń pole"
					>
						×
					</button>
				</div>
			{/each}

			<div class="add-account">
				{#if regularAssets.filter((a: any) => !visibleAccountIds.has(a.id)).length > 0}
					<details>
						<summary>+ Pokaż ukryte konta</summary>
						<div class="add-account-list">
							{#each regularAssets.filter((a: any) => !visibleAccountIds.has(a.id)) as account}
								<button
									type="button"
									class="btn-add-account"
									on:click={() => addAccount(account.id)}
								>
									{account.name}
									<span class="label-meta">({categoryLabels[account.category]})</span>
								</button>
							{/each}
						</div>
					</details>
				{/if}
				<button type="button" class="btn-new-account" on:click={() => openNewAccountForm('assets')}>
					+ Dodaj nowe konto
				</button>
			</div>
		</CardContent>
	</Card>

	<!-- Physical Assets -->
	<Card>
		<CardHeader>
			<CardTitle>🏠 Majątek</CardTitle>
		</CardHeader>
		<CardContent>
			{#each physicalAssets.filter((a: any) => visibleAssetIds.has(a.id)) as asset}
				<div class="form-group-with-remove">
					<div class="form-group">
						<label for="asset-{asset.id}" class="form-label">{asset.name}</label>
						<input
							id="asset-{asset.id}"
							type="number"
							step="0.01"
							bind:value={assetValues[asset.id]}
							placeholder="0.00"
							class="form-input"
						/>
					</div>
					<button
						type="button"
						class="btn-remove"
						on:click={() => removeAsset(asset.id)}
						title="Usuń pole"
					>
						×
					</button>
				</div>
			{/each}

			<div class="add-account">
				{#if physicalAssets.filter((a: any) => !visibleAssetIds.has(a.id)).length > 0}
					<details>
						<summary>+ Pokaż ukryty majątek</summary>
						<div class="add-account-list">
							{#each physicalAssets.filter((a: any) => !visibleAssetIds.has(a.id)) as asset}
								<button type="button" class="btn-add-account" on:click={() => addAsset(asset.id)}>
									{asset.name}
								</button>
							{/each}
						</div>
					</details>
				{/if}
				<button type="button" class="btn-new-account" on:click={() => (showNewAssetForm = true)}>
					+ Dodaj nowy majątek
				</button>
			</div>
		</CardContent>
	</Card>

	<!-- Liabilities -->
	{#if liabilities.length > 0}
		<Card>
			<CardHeader>
				<CardTitle>💸 Zobowiązania</CardTitle>
			</CardHeader>
			<CardContent>
				{#each liabilities.filter((a: any) => visibleAccountIds.has(a.id)) as account}
					<div class="form-group-with-remove">
						<div class="form-group">
							<label for="account-{account.id}" class="form-label">
								{account.name}
								<span class="label-meta">({categoryLabels[account.category]})</span>
							</label>
							<input
								id="account-{account.id}"
								type="number"
								step="0.01"
								bind:value={accountValues[account.id]}
								placeholder="0.00"
								class="form-input"
							/>
						</div>
						<button
							type="button"
							class="btn-remove"
							on:click={() => removeAccount(account.id)}
							title="Usuń pole"
						>
							×
						</button>
					</div>
				{/each}

				<div class="add-account">
					{#if liabilities.filter((a: any) => !visibleAccountIds.has(a.id)).length > 0}
						<details>
							<summary>+ Pokaż ukryte konta</summary>
							<div class="add-account-list">
								{#each liabilities.filter((a: any) => !visibleAccountIds.has(a.id)) as account}
									<button
										type="button"
										class="btn-add-account"
										on:click={() => addAccount(account.id)}
									>
										{account.name}
										<span class="label-meta">({categoryLabels[account.category]})</span>
									</button>
								{/each}
							</div>
						</details>
					{/if}
					<button
						type="button"
						class="btn-new-account"
						on:click={() => openNewAccountForm('liabilities')}
					>
						+ Dodaj nowe konto
					</button>
				</div>
			</CardContent>
		</Card>
	{/if}

	<!-- New Account Modal -->
	{#if showNewAccountForm}
		<div class="modal-overlay" on:click={() => (showNewAccountForm = false)}>
			<div class="modal" role="dialog" on:click|stopPropagation>
				<div class="modal-header">
					<h2>Dodaj nowe konto</h2>
					<button
						type="button"
						class="btn-close"
						on:click={() => (showNewAccountForm = false)}
						title="Zamknij"
					>
						×
					</button>
				</div>
				<div class="modal-content">
					<div class="form-group">
						<label for="newAccountName" class="form-label">Nazwa konta *</label>
						<input
							id="newAccountName"
							type="text"
							bind:value={newAccountName}
							placeholder="np. Konto oszczędnościowe"
							class="form-input"
							required
						/>
					</div>

					<div class="form-group">
						<label for="newAccountCategory" class="form-label">Kategoria *</label>
						<select id="newAccountCategory" bind:value={newAccountCategory} class="form-input">
							{#if newAccountSection === 'investments'}
								<option value="ike">IKE</option>
								<option value="ikze">IKZE</option>
								<option value="ppk">PPK</option>
								<option value="bonds">Obligacje</option>
								<option value="stocks">Akcje</option>
								<option value="fund">Fundusz</option>
								<option value="etf">ETF</option>
								<option value="other">Inne</option>
							{:else if newAccountSection === 'assets'}
								<option value="bank">Konto bankowe</option>
								<option value="real_estate">Nieruchomości</option>
								<option value="vehicle">Pojazd</option>
								<option value="other">Inne</option>
							{:else}
								<option value="mortgage">Hipoteka</option>
								<option value="installment">Raty</option>
								<option value="other">Inne</option>
							{/if}
						</select>
					</div>

					<div class="form-group">
						<label for="newAccountOwner" class="form-label">Właściciel</label>
						<select id="newAccountOwner" bind:value={newAccountOwner} class="form-input">
							<option value="Shared">Wspólne</option>
							<option value="Marcin">Marcin</option>
							<option value="Ewa">Ewa</option>
						</select>
					</div>

					<div class="form-group">
						<label for="newAccountValue" class="form-label">Wartość początkowa</label>
						<input
							id="newAccountValue"
							type="number"
							step="0.01"
							bind:value={newAccountValue}
							placeholder="0.00"
							class="form-input"
						/>
					</div>
				</div>
				<div class="modal-footer">
					<button
						type="button"
						class="btn btn-secondary"
						on:click={() => (showNewAccountForm = false)}
					>
						Anuluj
					</button>
					<button
						type="button"
						class="btn btn-primary"
						disabled={creatingAccount}
						on:click={createNewAccount}
					>
						{creatingAccount ? 'Tworzenie...' : 'Utwórz konto'}
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- New Asset Modal -->
	{#if showNewAssetForm}
		<div class="modal-overlay" on:click={() => (showNewAssetForm = false)}>
			<div class="modal" role="dialog" on:click|stopPropagation>
				<div class="modal-header">
					<h2>Dodaj nowy majątek</h2>
					<button
						type="button"
						class="btn-close"
						on:click={() => (showNewAssetForm = false)}
						title="Zamknij"
					>
						×
					</button>
				</div>
				<div class="modal-content">
					<div class="form-group">
						<label for="newAssetName" class="form-label">Nazwa *</label>
						<input
							id="newAssetName"
							type="text"
							bind:value={newAssetName}
							placeholder="np. Mieszkanie Poznań, Rower"
							class="form-input"
							required
						/>
					</div>

					<div class="form-group">
						<label for="newAssetValue" class="form-label">Wartość początkowa</label>
						<input
							id="newAssetValue"
							type="number"
							step="0.01"
							bind:value={newAssetValue}
							placeholder="0.00"
							class="form-input"
						/>
					</div>
				</div>
				<div class="modal-footer">
					<button
						type="button"
						class="btn btn-secondary"
						on:click={() => (showNewAssetForm = false)}
					>
						Anuluj
					</button>
					<button
						type="button"
						class="btn btn-primary"
						disabled={creatingAsset}
						on:click={createNewAsset}
					>
						{creatingAsset ? 'Tworzenie...' : 'Utwórz majątek'}
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Error Message -->
	{#if error}
		<div class="error-message">{error}</div>
	{/if}

	<!-- Submit Buttons -->
	<div class="button-group">
		<button type="submit" disabled={loading} class="btn btn-primary">
			{loading ? 'Zapisywanie...' : '💾 Zapisz Snapshot'}
		</button>
		<button type="button" on:click={() => goto('/')} class="btn btn-secondary"> Anuluj </button>
	</div>
</form>

<style>
	.snapshot-form {
		max-width: 800px;
		display: flex;
		flex-direction: column;
		gap: var(--size-6);
	}

	.form-group {
		margin-bottom: var(--size-4);
	}

	.form-group:last-child {
		margin-bottom: 0;
	}

	.form-label {
		display: block;
		font-weight: var(--font-weight-6);
		margin-bottom: var(--size-2);
		color: var(--color-text);
	}

	.label-meta {
		color: var(--color-text-secondary);
		font-weight: var(--font-weight-4);
		font-size: var(--font-size-1);
	}

	.form-input {
		width: 100%;
		padding: var(--size-2) var(--size-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: var(--color-bg);
		color: var(--color-text);
		font-size: var(--font-size-2);
		font-family: inherit;
		transition: all 0.2s;
	}

	.form-input:focus {
		outline: none;
		border-color: var(--color-primary);
		box-shadow: 0 0 0 2px rgba(94, 129, 172, 0.2);
	}

	.error-message {
		padding: var(--size-3);
		background: var(--nord11);
		color: var(--nord6);
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
	}

	.button-group {
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

	.btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.btn-primary {
		background: var(--color-primary);
		color: var(--nord6);
		flex: 1;
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--nord9);
	}

	.btn-secondary {
		background: transparent;
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}

	.btn-secondary:hover {
		background: var(--color-accent);
	}

	.form-group-with-remove {
		display: flex;
		gap: var(--size-2);
		align-items: flex-start;
		margin-bottom: var(--size-4);
	}

	.form-group-with-remove .form-group {
		flex: 1;
		margin-bottom: 0;
	}

	.btn-remove {
		flex-shrink: 0;
		width: 32px;
		height: 32px;
		margin-top: 28px;
		padding: 0;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: transparent;
		color: var(--color-text-secondary);
		font-size: var(--font-size-4);
		line-height: 1;
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-remove:hover {
		background: var(--nord11);
		color: var(--nord6);
		border-color: var(--nord11);
	}

	.add-account {
		margin-top: var(--size-4);
		padding: var(--size-3);
		border: 1px dashed var(--color-border);
		border-radius: var(--radius-2);
	}

	.add-account summary {
		cursor: pointer;
		color: var(--color-primary);
		font-weight: var(--font-weight-6);
		font-size: var(--font-size-2);
		user-select: none;
	}

	.add-account summary:hover {
		color: var(--nord9);
	}

	.add-account-list {
		display: flex;
		flex-direction: column;
		gap: var(--size-2);
	}

	.btn-add-account {
		width: 100%;
		padding: var(--size-2) var(--size-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: var(--color-bg);
		color: var(--color-text);
		font-size: var(--font-size-2);
		text-align: left;
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-add-account:hover {
		background: var(--color-accent);
		border-color: var(--color-primary);
	}

	.btn-new-account {
		width: 100%;
		padding: var(--size-3);
		margin-top: var(--size-2);
		border: 2px dashed var(--color-primary);
		border-radius: var(--radius-2);
		background: transparent;
		color: var(--color-primary);
		font-size: var(--font-size-2);
		font-weight: var(--font-weight-6);
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-new-account:hover {
		background: var(--color-primary);
		color: var(--nord6);
	}

	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: var(--size-4);
	}

	.modal {
		background: var(--color-bg);
		border-radius: var(--radius-2);
		max-width: 500px;
		width: 100%;
		box-shadow: var(--shadow-6);
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: var(--size-4);
		border-bottom: 1px solid var(--color-border);
	}

	.modal-header h2 {
		margin: 0;
		font-size: var(--font-size-4);
		font-weight: var(--font-weight-7);
		color: var(--color-text);
	}

	.btn-close {
		width: 32px;
		height: 32px;
		padding: 0;
		border: none;
		background: transparent;
		color: var(--color-text-secondary);
		font-size: var(--font-size-5);
		line-height: 1;
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-close:hover {
		color: var(--nord11);
	}

	.modal-content {
		padding: var(--size-4);
	}

	.modal-footer {
		display: flex;
		gap: var(--size-3);
		padding: var(--size-4);
		border-top: 1px solid var(--color-border);
	}

	.modal-footer .btn {
		flex: 1;
	}
</style>
