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
	import type { Asset } from './+page';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';

	let showForm = false;
	let editingAsset: Asset | null = null;
	let showDeleteModal = false;
	let assetToDelete: number | null = null;

	function startCreate() {
		editingAsset = null;
		showForm = true;
	}

	function startEdit(asset: Asset) {
		editingAsset = asset;
		showForm = true;
	}

	function cancelForm() {
		showForm = false;
		editingAsset = null;
	}

	let formData = {
		name: ''
	};

	let error = '';
	let saving = false;

	$: if (editingAsset) {
		formData = {
			name: editingAsset.name
		};
	} else if (showForm) {
		formData = {
			name: ''
		};
	}

	async function handleSubmit() {
		error = '';
		saving = true;

		try {
			const endpoint = editingAsset
				? `${apiUrl}/api/assets/${editingAsset.id}`
				: `${apiUrl}/api/assets`;
			const method = editingAsset ? 'PUT' : 'POST';

			const response = await fetch(endpoint, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(formData)
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.detail || 'Failed to save asset');
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

	function handleDelete(assetId: number) {
		assetToDelete = assetId;
		showDeleteModal = true;
	}

	function cancelDelete() {
		showDeleteModal = false;
		assetToDelete = null;
	}

	async function confirmDelete() {
		if (!assetToDelete) return;

		try {
			const response = await fetch(`${apiUrl}/api/assets/${assetToDelete}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				throw new Error('Failed to delete asset');
			}

			await invalidateAll();
			showDeleteModal = false;
			assetToDelete = null;
		} catch (err) {
			if (err instanceof Error) {
				error = err.message;
			}
			showDeleteModal = false;
			assetToDelete = null;
		}
	}
</script>

<svelte:head>
	<title>Majątek | Finansowa Forteca</title>
</svelte:head>

<div class="page-header">
	<div>
		<h1 class="page-title">Majątek</h1>
		<p class="page-description">Zarządzaj majątkiem fizycznym (nieruchomości, pojazdy, sprzęt)</p>
	</div>
	<button class="btn btn-primary" on:click={startCreate}>+ Nowy Majątek</button>
</div>

{#if showForm}
	<Card>
		<CardHeader>
			<CardTitle>{editingAsset ? '✏️ Edytuj Majątek' : '➕ Nowy Majątek'}</CardTitle>
		</CardHeader>
		<CardContent>
			<form on:submit|preventDefault={handleSubmit} class="asset-form">
				{#if error}
					<div class="error-message">{error}</div>
				{/if}

				<div class="form-group">
					<label for="name">Nazwa</label>
					<input
						type="text"
						id="name"
						bind:value={formData.name}
						required
						placeholder="np. Mieszkanie Poznań, Rower"
					/>
				</div>

				<div class="form-actions">
					<button type="button" class="btn btn-secondary" on:click={cancelForm} disabled={saving}>
						Anuluj
					</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>
						{saving ? 'Zapisywanie...' : editingAsset ? 'Zapisz zmiany' : 'Utwórz majątek'}
					</button>
				</div>
			</form>
		</CardContent>
	</Card>
{/if}

<Card>
	<CardHeader>
		<CardTitle>🏠 Majątek</CardTitle>
	</CardHeader>
	<CardContent>
		{#if data.assets.length === 0}
			<div class="empty-state">
				<p>Brak majątku</p>
			</div>
		{:else}
			<Table headers={['Nazwa', 'Wartość', 'Akcje']} mobileCardView class="assets-table">
				{#each data.assets as asset}
					<tr>
						<td data-label="Nazwa" class="name-cell">{asset.name}</td>
						<td data-label="Wartość" class="value-cell">{formatPLN(asset.current_value)}</td>
						<td data-label="Akcje" class="actions-cell">
							<button class="btn-icon tap-target" on:click={() => startEdit(asset)}>✏️</button>
							<button class="btn-icon tap-target" on:click={() => handleDelete(asset.id)}>🗑️</button
							>
						</td>
					</tr>
				{/each}
			</Table>
		{/if}
	</CardContent>
</Card>

<Modal
	open={showDeleteModal}
	title="Potwierdzenie usunięcia"
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
>
	<p>Czy na pewno chcesz usunąć ten majątek?</p>
	<p>Operacja ta ustawi majątek jako nieaktywny.</p>
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

	.assets-table {
		width: 100%;
		border-collapse: collapse;
	}

	.assets-table thead {
		border-bottom: 2px solid var(--color-border);
	}

	.assets-table th {
		text-align: left;
		padding: var(--size-3) var(--size-4);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.assets-table tbody tr {
		border-bottom: 1px solid var(--color-border);
		transition: background-color 0.2s;
	}

	.assets-table tbody tr:hover {
		background-color: var(--color-accent);
	}

	.assets-table td {
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

	.actions-cell {
		text-align: right;
	}

	.asset-form {
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

	.form-group input {
		padding: var(--size-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: var(--color-background);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.form-group input:focus {
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
	}
</style>
