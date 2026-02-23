<script lang="ts">
	import { Card, CardHeader, CardTitle, CardContent, Modal, Table } from '@mskalski/home-ui';
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

<div class="page-header">
	<div>
		<h1 class="page-title">Ustawienia</h1>
		<p class="page-description">Zarządzaj personami i preferencjami</p>
	</div>
</div>

<Card>
	<CardHeader>
		<div class="card-header-row">
			<CardTitle>Persony</CardTitle>
			<button class="btn btn-primary" on:click={startCreate}>+ Nowa Persona</button>
		</div>
	</CardHeader>
	<CardContent>
		{#if error && !showForm}
			<div class="error-message">{error}</div>
		{/if}

		{#if data.personas.length === 0}
			<div class="empty-state">
				<p>Brak person</p>
			</div>
		{:else}
			<Table
				headers={['Nazwa', 'Składka PPK pracownika (%)', 'Składka PPK pracodawcy (%)', 'Akcje']}
				mobileCardView
			>
				{#each data.personas as persona}
					<tr>
						<td data-label="Nazwa"><strong>{persona.name}</strong></td>
						<td data-label="Składka PPK pracownika (%)">{persona.ppk_employee_rate}%</td>
						<td data-label="Składka PPK pracodawcy (%)">{persona.ppk_employer_rate}%</td>
						<td data-label="Akcje" class="actions-cell">
							<button class="btn-icon" on:click={() => startEdit(persona)}>✏️</button>
							<button class="btn-icon" on:click={() => handleDelete(persona)}>🗑️</button>
						</td>
					</tr>
				{/each}
			</Table>
		{/if}
	</CardContent>
</Card>

<Modal
	open={showForm}
	title={editingPersona ? 'Edytuj Personę' : 'Nowa Persona'}
	onConfirm={handleSubmit}
	onCancel={cancelForm}
	confirmText={saving ? 'Zapisywanie...' : editingPersona ? 'Zapisz zmiany' : 'Utwórz'}
	confirmDisabled={saving}
	confirmVariant="primary"
>
	<form on:submit|preventDefault={handleSubmit} class="persona-form">
		{#if error}
			<div class="error-message">{error}</div>
		{/if}

		<div class="form-group">
			<label for="name">Nazwa</label>
			<input type="text" id="name" bind:value={formData.name} required placeholder="np. Marcin" />
		</div>

		<div class="form-group">
			<label for="ppk_employee_rate">Składka PPK pracownika (%)</label>
			<input
				type="number"
				id="ppk_employee_rate"
				bind:value={formData.ppk_employee_rate}
				min="0.5"
				max="4.0"
				step="0.5"
				required
			/>
		</div>

		<div class="form-group">
			<label for="ppk_employer_rate">Składka PPK pracodawcy (%)</label>
			<input
				type="number"
				id="ppk_employer_rate"
				bind:value={formData.ppk_employer_rate}
				min="1.5"
				max="4.0"
				step="0.5"
				required
			/>
		</div>
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
>
	<p>Czy na pewno chcesz usunąć personę "{personaToDelete?.name}"?</p>
	<p>Usunięcie jest możliwe tylko jeśli żadne konta nie odnoszą się do tej persony.</p>
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

	.card-header-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
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

	.actions-cell {
		text-align: right;
	}

	.empty-state {
		text-align: center;
		padding: var(--size-8) var(--size-4);
		color: var(--color-text-secondary);
	}

	.error-message {
		padding: var(--size-3);
		background: var(--nord11);
		color: var(--nord6);
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
		margin-bottom: var(--size-3);
	}

	.persona-form {
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
</style>
