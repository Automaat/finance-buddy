<script lang="ts">
	type NewAccountSection = 'financial' | 'retirement' | 'investment' | 'majatek' | 'liabilities';

	interface Props {
		section: NewAccountSection;
		name?: string;
		category?: string;
		wrapper?: '' | 'PPK' | 'IKE' | 'IKZE';
		owner?: string;
		value?: number;
		creating?: boolean;
		personas?: Array<{ id: number; name: string }>;
		onCreate: () => void;
		onClose: () => void;
	}

	let {
		section,
		name = $bindable(''),
		category = $bindable(''),
		wrapper = $bindable(''),
		owner = $bindable(''),
		value = $bindable(0),
		creating = false,
		personas = [],
		onCreate,
		onClose
	}: Props = $props();

	function closeOnBackdrop(event: MouseEvent) {
		if (event.target === event.currentTarget) onClose();
	}

	function closeOnEscape(event: KeyboardEvent) {
		if (event.key === 'Escape') onClose();
	}
</script>

<svelte:window onkeydown={closeOnEscape} />

<div class="modal-overlay" role="presentation" onclick={closeOnBackdrop}>
	<div
		class="modal"
		role="dialog"
		aria-modal="true"
		aria-labelledby="new-account-modal-title"
		tabindex="-1"
	>
		<div class="modal-header">
			<h2 id="new-account-modal-title">Dodaj nowe konto</h2>
			<button type="button" class="btn-close" onclick={onClose} title="Zamknij"> × </button>
		</div>
		<div class="modal-content">
			<div class="form-group">
				<label for="newAccountName" class="form-label">Nazwa konta *</label>
				<input
					id="newAccountName"
					type="text"
					bind:value={name}
					placeholder="np. Konto oszczędnościowe"
					class="form-input"
					required
				/>
			</div>

			<div class="form-group">
				<label for="newAccountCategory" class="form-label">Kategoria *</label>
				<select id="newAccountCategory" bind:value={category} class="form-input">
					{#if section === 'financial'}
						<option value="bank">Konto bankowe</option>
						<option value="saving_account">Konto oszczędnościowe</option>
					{:else if section === 'retirement'}
						<option value="stock">Akcje</option>
						<option value="bond">Obligacje</option>
						<option value="fund">Fundusz</option>
						<option value="etf">ETF</option>
						<option value="ppk">PPK</option>
					{:else if section === 'investment'}
						<option value="stock">Akcje</option>
						<option value="bond">Obligacje</option>
						<option value="fund">Fundusz</option>
						<option value="etf">ETF</option>
						<option value="gold">Złoto</option>
						<option value="other">Inne</option>
					{:else if section === 'majatek'}
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

			{#if section === 'retirement'}
				<div class="form-group">
					<label for="newAccountWrapper" class="form-label">Wrapper *</label>
					<select id="newAccountWrapper" bind:value={wrapper} class="form-input">
						<option value="IKE">IKE</option>
						<option value="IKZE">IKZE</option>
						<option value="PPK">PPK</option>
					</select>
				</div>
			{/if}

			<div class="form-group">
				<label for="newAccountOwner" class="form-label">Właściciel</label>
				<select id="newAccountOwner" bind:value={owner} class="form-input">
					{#each personas as persona}
						<option value={persona.name}>{persona.name}</option>
					{/each}
				</select>
			</div>

			<div class="form-group">
				<label for="newAccountValue" class="form-label">Wartość początkowa</label>
				<input
					id="newAccountValue"
					type="number"
					step="0.01"
					bind:value
					placeholder="0.00"
					class="form-input"
				/>
			</div>
		</div>
		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={onClose}> Anuluj </button>
			<button type="button" class="btn btn-primary" disabled={creating} onclick={onCreate}>
				{creating ? 'Tworzenie...' : 'Utwórz konto'}
			</button>
		</div>
	</div>
</div>

<style>
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

	.form-input {
		width: 100%;
		padding: var(--size-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: var(--color-bg);
		color: var(--color-text);
		font-size: var(--font-size-2);
		font-family: inherit;
		transition: all 0.2s;
		min-height: var(--tap-target-min);
	}

	.form-input:focus {
		outline: none;
		border-color: var(--color-primary);
		box-shadow: 0 0 0 2px rgba(94, 129, 172, 0.2);
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
		width: var(--tap-target-min);
		height: var(--tap-target-min);
		padding: 0;
		border: none;
		background: transparent;
		color: var(--color-text-secondary);
		font-size: var(--font-size-5);
		line-height: 1;
		cursor: pointer;
		transition: all 0.2s;
		display: inline-flex;
		align-items: center;
		justify-content: center;
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
