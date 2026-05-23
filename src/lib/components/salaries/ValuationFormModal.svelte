<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import type { ValuationSource } from '$lib/types/salaries';

	export interface ValuationFormData {
		company: string;
		date: string;
		currency: string;
		fmv_per_share: number;
		fmv_low: number | null;
		fmv_high: number | null;
		source: ValuationSource;
		common_stock_discount_pct: number | null;
		notes: string;
	}

	interface Props {
		open: boolean;
		editing: boolean;
		data: ValuationFormData;
		error: string;
		saving: boolean;
		onSave: () => void;
		onCancel: () => void;
	}

	let { open, editing, data, error, saving, onSave, onCancel }: Props = $props();
</script>

<Modal
	{open}
	title={editing ? 'Edytuj wycenę' : 'Nowa wycena'}
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
			<span class="font-semibold text-sm">Firma*</span>
			<input
				type="text"
				class="input"
				bind:value={data.company}
				placeholder="Nazwa firmy"
				required
			/>
		</label>

		<div class="grid grid-cols-2 gap-2">
			<label class="label">
				<span class="font-semibold text-sm">Data wyceny*</span>
				<input type="date" class="input" bind:value={data.date} required />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Waluta*</span>
				<select class="select" bind:value={data.currency} required>
					<option value="USD">USD</option>
					<option value="EUR">EUR</option>
					<option value="PLN">PLN</option>
					<option value="GBP">GBP</option>
					<option value="CHF">CHF</option>
				</select>
			</label>
		</div>

		<label class="label">
			<span class="font-semibold text-sm">FMV per share (bazowa)*</span>
			<input
				type="number"
				class="input"
				bind:value={data.fmv_per_share}
				min="0"
				step="0.0001"
				required
			/>
		</label>

		<div class="grid grid-cols-2 gap-2">
			<label class="label">
				<span class="font-semibold text-sm">FMV low (opcjonalna)</span>
				<input type="number" class="input" bind:value={data.fmv_low} min="0" step="0.0001" />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">FMV high (opcjonalna)</span>
				<input type="number" class="input" bind:value={data.fmv_high} min="0" step="0.0001" />
			</label>
		</div>

		<label class="label">
			<span class="font-semibold text-sm">Źródło*</span>
			<select class="select" bind:value={data.source} required>
				<option value="409a">409A</option>
				<option value="preferred_round">Runda preferred</option>
				<option value="tender">Tender / wykup</option>
				<option value="estimate">Estymacja</option>
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Common stock discount (%) — opcjonalne</span>
			<input
				type="number"
				class="input"
				bind:value={data.common_stock_discount_pct}
				min="0"
				max="100"
				step="0.1"
				placeholder="np. 30"
			/>
			<span class="text-xs text-surface-700-300"
				>Stosowane przy wycenie preferred → common (zwykle 20–40%)</span
			>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Notatki</span>
			<input
				type="text"
				class="input"
				bind:value={data.notes}
				placeholder="np. Series C post-money"
			/>
		</label>
	</form>
</Modal>
