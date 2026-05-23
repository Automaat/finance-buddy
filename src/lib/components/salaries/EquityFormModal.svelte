<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { Plus, Trash2 } from 'lucide-svelte';
	import type { OwnerOption } from '$lib/types/owners';
	import type {
		CustomVestingEvent,
		EquityGrantType,
		EquityTaxTreatment,
		VestingFrequency
	} from '$lib/types/salaries';

	export interface EquityFormData {
		grant_date: string;
		type: EquityGrantType;
		company: string;
		owner_user_id: number | null;
		total_shares: number;
		strike_price: number | null;
		currency: string;
		vest_start_date: string;
		vest_cliff_months: number;
		vest_total_months: number;
		vest_frequency: VestingFrequency;
		preset: string;
		vest_custom_schedule: CustomVestingEvent[] | null;
		requires_liquidity_event: boolean;
		liquidity_event_date: string | null;
		tax_treatment: EquityTaxTreatment;
		notes: string;
	}

	export interface VestingPreset {
		label: string;
		cliff: number;
		total: number;
		frequency: VestingFrequency;
		custom: CustomVestingEvent[] | null;
	}

	interface Props {
		open: boolean;
		editing: boolean;
		data: EquityFormData;
		error: string;
		saving: boolean;
		today: string;
		owners: OwnerOption[];
		vestingPresets: Record<string, VestingPreset>;
		taxTreatmentLabels: Record<EquityTaxTreatment, string>;
		onApplyPreset: (key: string) => void;
		onSave: () => void;
		onCancel: () => void;
	}

	let {
		open,
		editing,
		data,
		error,
		saving,
		today,
		owners,
		vestingPresets,
		taxTreatmentLabels,
		onApplyPreset,
		onSave,
		onCancel
	}: Props = $props();
</script>

<Modal
	{open}
	title={editing ? 'Edytuj grant' : 'Nowy grant'}
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

		<div class="grid grid-cols-2 gap-2">
			<label class="label">
				<span class="font-semibold text-sm">Typ*</span>
				<select class="select" bind:value={data.type} required>
					<option value="rsu">RSU</option>
					<option value="option">Opcje</option>
				</select>
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Data grantu*</span>
				<input type="date" class="input" bind:value={data.grant_date} max={today} required />
			</label>
		</div>

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

		<div class="grid grid-cols-2 gap-2">
			<label class="label">
				<span class="font-semibold text-sm">Liczba akcji*</span>
				<input
					type="number"
					class="input"
					bind:value={data.total_shares}
					min="1"
					step="1"
					required
				/>
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

		{#if data.type === 'option'}
			<label class="label">
				<span class="font-semibold text-sm">Strike price (cena wykonania)*</span>
				<input
					type="number"
					class="input"
					bind:value={data.strike_price}
					min="0"
					step="0.0001"
					required
				/>
			</label>
		{/if}

		<fieldset class="card preset-tonal-surface p-3 space-y-3">
			<legend class="font-semibold text-sm px-1">Harmonogram vestingu</legend>

			<label class="label">
				<span class="font-semibold text-sm">Schemat</span>
				<select class="select" bind:value={data.preset} onchange={() => onApplyPreset(data.preset)}>
					{#each Object.entries(vestingPresets) as [key, preset]}
						<option value={key}>{preset.label}</option>
					{/each}
				</select>
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Data startu vestingu*</span>
				<input type="date" class="input" bind:value={data.vest_start_date} required />
			</label>

			<div class="grid grid-cols-3 gap-2">
				<label class="label">
					<span class="font-semibold text-sm">Cliff (msc)</span>
					<input type="number" class="input" bind:value={data.vest_cliff_months} min="0" step="1" />
				</label>
				<label class="label">
					<span class="font-semibold text-sm">Całość (msc)*</span>
					<input
						type="number"
						class="input"
						bind:value={data.vest_total_months}
						min="1"
						step="1"
						required
					/>
				</label>
				<label class="label">
					<span class="font-semibold text-sm">Częstotliwość</span>
					<select class="select" bind:value={data.vest_frequency}>
						<option value="monthly">Miesięczna</option>
						<option value="quarterly">Kwartalna</option>
						<option value="yearly">Roczna</option>
					</select>
				</label>
			</div>

			{#if data.preset === 'custom'}
				<div class="text-xs text-surface-700-300">
					Niestandardowy harmonogram: lista zdarzeń (miesiąc + % od całości).
				</div>
				{#each data.vest_custom_schedule ?? [] as event, idx (idx)}
					<div class="grid grid-cols-[1fr,1fr,auto] gap-2 items-end">
						<label class="label">
							<span class="text-xs">Miesiąc</span>
							<input type="number" class="input" bind:value={event.month} min="0" step="1" />
						</label>
						<label class="label">
							<span class="text-xs">% od całości</span>
							<input type="number" class="input" bind:value={event.pct} min="0" step="0.1" />
						</label>
						<button
							type="button"
							class="btn-icon btn-icon-sm"
							aria-label="Usuń wiersz"
							onclick={() => {
								data.vest_custom_schedule =
									data.vest_custom_schedule?.filter((_, i) => i !== idx) ?? null;
							}}
						>
							<Trash2 size={14} />
						</button>
					</div>
				{/each}
				<button
					type="button"
					class="btn preset-tonal-surface btn-sm"
					onclick={() => {
						const next = [...(data.vest_custom_schedule ?? []), { month: 0, pct: 0 }];
						data.vest_custom_schedule = next;
					}}
				>
					<Plus size={14} /> Dodaj zdarzenie
				</button>
			{/if}
		</fieldset>

		<fieldset class="card preset-tonal-surface p-3 space-y-3">
			<legend class="font-semibold text-sm px-1">Liquidity event (double-trigger)</legend>
			<label class="flex items-center gap-2 cursor-pointer">
				<input type="checkbox" class="checkbox" bind:checked={data.requires_liquidity_event} />
				<span class="text-sm">Wymaga liquidity event (IPO / akwizycja)</span>
			</label>
			{#if data.requires_liquidity_event}
				<label class="label">
					<span class="font-semibold text-sm">Data liquidity event</span>
					<input type="date" class="input" bind:value={data.liquidity_event_date} />
					<span class="text-xs text-surface-700-300"
						>Puste = jeszcze nie wystąpiło. Vested = 0 dopóki nie wystąpi.</span
					>
				</label>
			{/if}
		</fieldset>

		<label class="label">
			<span class="font-semibold text-sm">Traktowanie podatkowe (PL)</span>
			<select class="select" bind:value={data.tax_treatment}>
				{#each Object.entries(taxTreatmentLabels) as [key, label]}
					<option value={key}>{label}</option>
				{/each}
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Notatki</span>
			<input
				type="text"
				class="input"
				bind:value={data.notes}
				placeholder="np. ESOP 2024, double-trigger RSU"
			/>
		</label>
	</form>
</Modal>
