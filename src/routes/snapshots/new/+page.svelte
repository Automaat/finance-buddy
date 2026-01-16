<script lang="ts">
	import { goto } from '$app/navigation';
	import { env } from '$env/dynamic/public';
	import Card from '$lib/components/Card.svelte';
	import CardHeader from '$lib/components/CardHeader.svelte';
	import CardTitle from '$lib/components/CardTitle.svelte';
	import CardContent from '$lib/components/CardContent.svelte';

	export let data;

	// Initialize form state
	let snapshotDate = new Date().toISOString().split('T')[0];
	let notes = '';
	let loading = false;
	let error = '';

	// Separate investments from regular assets
	const investmentCategories = ['ike', 'ikze', 'ppk', 'bonds', 'stocks', 'fund', 'etf'];
	let investments = data.assets.filter((a: any) => investmentCategories.includes(a.category));
	let regularAssets = data.assets.filter((a: any) => !investmentCategories.includes(a.category));

	// Track visible accounts (initially show accounts with value > 0)
	let visibleAccountIds = new Set<number>(
		[...data.assets, ...data.liabilities]
			.filter((a: any) => a.current_value > 0)
			.map((a: any) => a.id)
	);

	// Initialize values with current values from accounts
	let values: Record<number, number> = {};
	[...data.assets, ...data.liabilities].forEach((account) => {
		values[account.id] = account.current_value;
	});

	function removeAccount(accountId: number) {
		visibleAccountIds.delete(accountId);
		visibleAccountIds = new Set(visibleAccountIds); // Trigger reactivity
	}

	function addAccount(accountId: number) {
		visibleAccountIds.add(accountId);
		visibleAccountIds = new Set(visibleAccountIds); // Trigger reactivity
	}

	// New account creation state
	let showNewAccountForm = false;
	let newAccountSection: 'investments' | 'assets' | 'liabilities' = 'assets';
	let newAccountName = '';
	let newAccountCategory = '';
	let newAccountOwner = 'Shared';
	let newAccountValue = 0;
	let creatingAccount = false;

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
				data.liabilities = [...data.liabilities, newAccount];
			}

			// Set initial value and mark as visible
			values[newAccount.id] = newAccountValue;
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
		// Set default category based on section
		if (section === 'investments') {
			newAccountCategory = 'ike';
		} else if (section === 'assets') {
			newAccountCategory = 'bank';
		} else {
			newAccountCategory = 'mortgage';
		}
		showNewAccountForm = true;
	}

	async function handleSubmit() {
		loading = true;
		error = '';

		try {
			// Build payload: visible accounts with values, hidden accounts with 0
			const allAccounts = [...investments, ...regularAssets, ...data.liabilities];
			const payload = {
				date: snapshotDate,
				notes: notes || null,
				values: allAccounts.map((account) => {
					const isVisible = visibleAccountIds.has(account.id);
					const value = isVisible ? values[account.id] : 0;
					const parsedValue = parseFloat(value.toString());
					if (Number.isNaN(parsedValue)) {
						throw new Error('Invalid value for account. Please enter a valid number.');
					}
					return {
						account_id: account.id,
						value: parsedValue
					};
				})
			};

			const response = await fetch(`${env.PUBLIC_API_URL_BROWSER}/api/snapshots`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.detail || 'Failed to create snapshot');
			}

			// Redirect to dashboard on success
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

<svelte:head>
	<title>Nowy Snapshot | Finansowa Forteca</title>
</svelte:head>

<div class="page-header">
	<div>
		<h1 class="page-title">Nowy Snapshot</h1>
		<p class="page-description">Zaktualizuj wartości wszystkich kont</p>
	</div>
</div>

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
								bind:value={values[account.id]}
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
			<CardTitle>💰 Aktywa</CardTitle>
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
							bind:value={values[account.id]}
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

	<!-- Liabilities -->
	{#if data.liabilities.length > 0}
		<Card>
			<CardHeader>
				<CardTitle>💸 Zobowiązania</CardTitle>
			</CardHeader>
			<CardContent>
				{#each data.liabilities.filter((a: any) => visibleAccountIds.has(a.id)) as account}
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
								bind:value={values[account.id]}
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
					{#if data.liabilities.filter((a: any) => !visibleAccountIds.has(a.id)).length > 0}
						<details>
							<summary>+ Pokaż ukryte konta</summary>
							<div class="add-account-list">
								{#each data.liabilities.filter((a: any) => !visibleAccountIds.has(a.id)) as account}
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
			<div class="modal" on:click|stopPropagation>
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
	.page-header {
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

	.add-account[open] summary {
		margin-bottom: var(--size-3);
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
