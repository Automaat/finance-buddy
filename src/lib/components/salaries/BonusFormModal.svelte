<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import type { OwnerOption } from '$lib/types/owners';
	import type { BonusType } from '$lib/types/salaries';

	export interface BonusFormData {
		date: string;
		amount: number;
		currency: string;
		type: BonusType;
		company: string;
		owner_user_id: number | null;
		contract_type: string;
		notes: string;
	}

	interface Props {
		open: boolean;
		editing: boolean;
		data: BonusFormData;
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
	title={editing ? 'Edytuj bonus' : 'Nowy bonus'}
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
			<span class="font-semibold text-sm">Data wypłaty*</span>
			<input type="date" class="input" bind:value={data.date} max={today} required />
		</label>

		<div class="grid grid-cols-3 gap-2">
			<label class="label col-span-2">
				<span class="font-semibold text-sm">Kwota*</span>
				<input type="number" class="input" bind:value={data.amount} min="0" step="0.01" required />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Waluta*</span>
				<select class="select" bind:value={data.currency} required>
					<option value="PLN">PLN</option>
					<option value="USD">USD</option>
					<option value="EUR">EUR</option>
					<option value="GBP">GBP</option>
					<option value="CHF">CHF</option>
				</select>
			</label>
		</div>

		<label class="label">
			<span class="font-semibold text-sm">Typ*</span>
			<select class="select" bind:value={data.type} required>
				<option value="annual">Roczny</option>
				<option value="signon">Powitalny</option>
				<option value="spot">Uznaniowy</option>
				<option value="retention">Retencyjny</option>
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
			<span class="font-semibold text-sm">Notatki</span>
			<input
				type="text"
				class="input"
				bind:value={data.notes}
				placeholder="np. Q4 performance bonus"
			/>
		</label>
	</form>
</Modal>
