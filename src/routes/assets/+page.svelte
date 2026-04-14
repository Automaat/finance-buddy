<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { Home, Pencil, Plus, Trash2 } from 'lucide-svelte';
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

	let formData = { name: '' };
	let error = '';
	let saving = false;

	$: if (editingAsset) {
		formData = { name: editingAsset.name };
	} else if (showForm) {
		formData = { name: '' };
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
			const response = await fetch(`${apiUrl}/api/assets/${assetToDelete}`, { method: 'DELETE' });

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

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Majątek</h1>
		<p class="text-surface-700-300 text-sm">
			Zarządzaj majątkiem fizycznym (nieruchomości, pojazdy, sprzęt)
		</p>
	</div>
	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto gap-2"
		on:click={startCreate}
	>
		<Plus size={16} />
		Nowy Majątek
	</button>
</div>

<div class="space-y-4">
	{#if showForm}
		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header>
				<h3 class="h3 flex items-center gap-2">
					{#if editingAsset}
						<Pencil size={20} /> Edytuj Majątek
					{:else}
						<Plus size={20} /> Nowy Majątek
					{/if}
				</h3>
			</header>
			<form class="space-y-4" on:submit|preventDefault={handleSubmit}>
				{#if error}
					<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
				{/if}

				<label class="label">
					<span class="font-semibold text-sm">Nazwa</span>
					<input
						type="text"
						class="input"
						bind:value={formData.name}
						required
						placeholder="np. Mieszkanie Poznań, Rower"
					/>
				</label>

				<div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-2">
					<button
						type="button"
						class="btn preset-tonal-surface"
						on:click={cancelForm}
						disabled={saving}>Anuluj</button
					>
					<button type="submit" class="btn preset-filled-primary-500" disabled={saving}>
						{saving ? 'Zapisywanie...' : editingAsset ? 'Zapisz zmiany' : 'Utwórz majątek'}
					</button>
				</div>
			</form>
		</div>
	{/if}

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Home size={20} /> Majątek</h3>
		</header>

		{#if data.assets.length === 0}
			<div class="text-center py-12 text-surface-700-300">
				<p>Brak majątku</p>
			</div>
		{:else}
			<div class="table-wrap">
				<table class="table table-hover">
					<thead>
						<tr>
							<th>Nazwa</th>
							<th>Wartość</th>
							<th class="text-right">Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.assets as asset}
							<tr>
								<td class="font-medium">{asset.name}</td>
								<td class="font-semibold text-primary-600-400">{formatPLN(asset.current_value)}</td>
								<td class="text-right whitespace-nowrap">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										on:click={() => startEdit(asset)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										on:click={() => handleDelete(asset.id)}
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
	open={showDeleteModal}
	title="Potwierdzenie usunięcia"
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
	confirmText="Usuń"
	confirmVariant="danger"
>
	<p class="mb-2">Czy na pewno chcesz usunąć ten majątek?</p>
	<p class="text-sm text-surface-700-300">Operacja ta ustawi majątek jako nieaktywny.</p>
</Modal>
