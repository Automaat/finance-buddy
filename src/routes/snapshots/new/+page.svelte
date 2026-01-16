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

	// Initialize values with current values from accounts
	let values: Record<number, number> = {};
	[...data.assets, ...data.liabilities].forEach((account) => {
		values[account.id] = account.current_value;
	});

	async function handleSubmit() {
		loading = true;
		error = '';

		try {
			// Build request payload
			const payload = {
				date: snapshotDate,
				notes: notes || null,
				values: Object.entries(values).map(([accountId, value]) => ({
					account_id: parseInt(accountId),
					value: parseFloat(value.toString()) || 0
				}))
			};

			const response = await fetch(`${env.PUBLIC_API_URL}/api/snapshots`, {
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
		other: 'Inne'
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

	<!-- Assets -->
	<Card>
		<CardHeader>
			<CardTitle>💰 Aktywa</CardTitle>
		</CardHeader>
		<CardContent>
			{#each data.assets as account}
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
						required
						class="form-input"
					/>
				</div>
			{/each}
		</CardContent>
	</Card>

	<!-- Liabilities -->
	{#if data.liabilities.length > 0}
		<Card>
			<CardHeader>
				<CardTitle>💸 Zobowiązania</CardTitle>
			</CardHeader>
			<CardContent>
				{#each data.liabilities as account}
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
							required
							class="form-input"
						/>
					</div>
				{/each}
			</CardContent>
		</Card>
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
</style>
