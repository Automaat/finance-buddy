<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { Plus, Pencil, Trash2 } from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import type { Persona } from '$lib/types/personas';

	export let data: { personas: Persona[] };

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';

	let showForm = false;
	let editingPersona: Persona | null = null;
	let showDeleteModal = false;
	let personaToDelete: Persona | null = null;
	let error = '';
	let saving = false;

	let formData = {
		name: '',
		ppk_employee_rate: 2.0,
		ppk_employer_rate: 1.5
	};

	function startCreate() {
		editingPersona = null;
		formData = { name: '', ppk_employee_rate: 2.0, ppk_employer_rate: 1.5 };
		showForm = true;
	}

	function startEdit(persona: Persona) {
		editingPersona = persona;
		formData = {
			name: persona.name,
			ppk_employee_rate: persona.ppk_employee_rate,
			ppk_employer_rate: persona.ppk_employer_rate
		};
		showForm = true;
	}

	function cancelForm() {
		showForm = false;
		editingPersona = null;
		error = '';
	}

	async function handleSubmit() {
		error = '';
		saving = true;

		try {
			const endpoint = editingPersona
				? `${apiUrl}/api/personas/${editingPersona.id}`
				: `${apiUrl}/api/personas`;
			const method = editingPersona ? 'PUT' : 'POST';

			const response = await fetch(endpoint, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(formData)
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.detail || 'Failed to save persona');
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

	function handleDelete(persona: Persona) {
		personaToDelete = persona;
		showDeleteModal = true;
	}

	async function confirmDelete() {
		if (!personaToDelete) return;

		try {
			const response = await fetch(`${apiUrl}/api/personas/${personaToDelete.id}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.detail || 'Failed to delete persona');
			}

			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) {
				error = err.message;
			}
		} finally {
			showDeleteModal = false;
			personaToDelete = null;
		}
	}
</script>

<svelte:head>
	<title>Ustawienia | Finansowa Forteca</title>
</svelte:head>

<div class="mb-6 space-y-1">
	<h1 class="h2">Ustawienia</h1>
	<p class="text-surface-700-300 text-sm">Zarządzaj personami i preferencjami</p>
</div>

<div class="card preset-filled-surface-100-900 p-4 space-y-4">
	<header class="flex items-center justify-between flex-wrap gap-3">
		<h3 class="h3">Persony</h3>
		<button type="button" class="btn preset-filled-primary-500 gap-2" on:click={startCreate}>
			<Plus size={16} />
			Nowa Persona
		</button>
	</header>

	{#if error && !showForm}
		<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
	{/if}

	{#if data.personas.length === 0}
		<div class="text-center py-12 text-surface-700-300">
			<p>Brak person</p>
		</div>
	{:else}
		<div class="table-wrap">
			<table class="table table-hover">
				<thead>
					<tr>
						<th>Nazwa</th>
						<th>Składka PPK pracownika (%)</th>
						<th>Składka PPK pracodawcy (%)</th>
						<th class="text-right">Akcje</th>
					</tr>
				</thead>
				<tbody>
					{#each data.personas as persona}
						<tr>
							<td><strong>{persona.name}</strong></td>
							<td>{persona.ppk_employee_rate}%</td>
							<td>{persona.ppk_employer_rate}%</td>
							<td class="text-right whitespace-nowrap">
								<button
									type="button"
									class="btn-icon btn-icon-sm"
									aria-label="Edytuj"
									on:click={() => startEdit(persona)}
								>
									<Pencil size={16} />
								</button>
								<button
									type="button"
									class="btn-icon btn-icon-sm"
									aria-label="Usuń"
									on:click={() => handleDelete(persona)}
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

<Modal
	open={showForm}
	title={editingPersona ? 'Edytuj Personę' : 'Nowa Persona'}
	onConfirm={handleSubmit}
	onCancel={cancelForm}
	confirmText={saving ? 'Zapisywanie...' : editingPersona ? 'Zapisz zmiany' : 'Utwórz'}
	confirmDisabled={saving}
>
	<form on:submit|preventDefault={handleSubmit} class="space-y-4">
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
				placeholder="np. Marcin"
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Składka PPK pracownika (%)</span>
			<input
				type="number"
				class="input"
				bind:value={formData.ppk_employee_rate}
				min="0.5"
				max="4.0"
				step="0.5"
				required
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Składka PPK pracodawcy (%)</span>
			<input
				type="number"
				class="input"
				bind:value={formData.ppk_employer_rate}
				min="1.5"
				max="4.0"
				step="0.5"
				required
			/>
		</label>
	</form>
</Modal>

<Modal
	open={showDeleteModal}
	title="Potwierdzenie usunięcia"
	onConfirm={confirmDelete}
	onCancel={() => {
		showDeleteModal = false;
		personaToDelete = null;
	}}
	confirmText="Usuń"
	confirmVariant="danger"
>
	<p class="mb-2">Czy na pewno chcesz usunąć personę "{personaToDelete?.name}"?</p>
	<p class="text-sm text-surface-700-300">
		Usunięcie jest możliwe tylko jeśli żadne konta nie odnoszą się do tej persony.
	</p>
</Modal>
