<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import type { OwnerOption } from '$lib/types/owners';

	export interface SalaryFormData {
		date: string;
		gross_amount: number;
		contract_type: string;
		company: string;
		owner_user_id: number | null;
	}

	interface Props {
		open: boolean;
		editing: boolean;
		data: SalaryFormData;
		error: string;
		saving: boolean;
		today: string;
		owners: OwnerOption[];
		onSave: () => void;
		onCancel: () => void;
	}

	let { open, editing, data, error, saving, today, owners, onSave, onCancel }: Props = $props();
</script>

<Modal
	{open}
	title={editing ? 'Edytuj wynagrodzenie' : 'Nowe wynagrodzenie'}
	onConfirm={onSave}
	{onCancel}
	confirmText={saving ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={saving}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			onSave();
		}}
		class="space-y-4"
	>
		{#if error}
			<div class="card preset-filled-error-500 p-3 text-sm">{error}</div>
		{/if}

		<label class="label">
			<span class="font-semibold text-sm">Data zmiany*</span>
			<input type="date" class="input" bind:value={data.date} max={today} required />
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Pensja brutto (PLN)*</span>
			<input
				type="number"
				class="input"
				bind:value={data.gross_amount}
				min="0"
				step="0.01"
				required
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Rodzaj umowy*</span>
			<select class="select" bind:value={data.contract_type} required>
				<option value="UOP">UOP</option>
				<option value="UZ">UZ</option>
				<option value="UoD">UoD</option>
				<option value="B2B">B2B</option>
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Firma*</span>
			<input
				type="text"
				class="input"
				bind:value={data.company}
				placeholder="Nazwa firmy"
				required
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Właściciel*</span>
			<select class="select" bind:value={data.owner_user_id} required>
				{#each owners as owner (owner.id)}
					<option value={owner.id}>{owner.name}</option>
				{/each}
			</select>
		</label>
	</form>
</Modal>
