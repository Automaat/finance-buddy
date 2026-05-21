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

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-surface-950/60 backdrop-blur-sm p-4"
	role="presentation"
	onclick={closeOnBackdrop}
>
	<div
		class="card preset-filled-surface-50-950 w-full max-w-lg max-h-[90vh] flex flex-col shadow-xl"
		role="dialog"
		aria-modal="true"
		aria-labelledby="new-account-modal-title"
		tabindex="-1"
	>
		<header class="flex items-center justify-between px-5 py-4 border-b border-surface-200-800">
			<h2 id="new-account-modal-title" class="h4 font-bold">Dodaj nowe konto</h2>
			<button
				type="button"
				class="btn-icon btn-icon-sm"
				aria-label="Zamknij"
				title="Zamknij"
				onclick={onClose}
			>
				×
			</button>
		</header>

		<div class="flex-1 overflow-y-auto px-5 py-4 space-y-4">
			<div class="flex flex-col gap-1">
				<label for="newAccountName" class="text-sm font-semibold">Nazwa konta *</label>
				<input
					id="newAccountName"
					type="text"
					bind:value={name}
					placeholder="np. Konto oszczędnościowe"
					class="input"
					required
				/>
			</div>

			<div class="flex flex-col gap-1">
				<label for="newAccountCategory" class="text-sm font-semibold">Kategoria *</label>
				<select id="newAccountCategory" bind:value={category} class="select">
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
				<div class="flex flex-col gap-1">
					<label for="newAccountWrapper" class="text-sm font-semibold">Wrapper *</label>
					<select id="newAccountWrapper" bind:value={wrapper} class="select">
						<option value="IKE">IKE</option>
						<option value="IKZE">IKZE</option>
						<option value="PPK">PPK</option>
					</select>
				</div>
			{/if}

			<div class="flex flex-col gap-1">
				<label for="newAccountOwner" class="text-sm font-semibold">Właściciel</label>
				<select id="newAccountOwner" bind:value={owner} class="select">
					{#each personas as persona}
						<option value={persona.name}>{persona.name}</option>
					{/each}
				</select>
			</div>

			<div class="flex flex-col gap-1">
				<label for="newAccountValue" class="text-sm font-semibold">Wartość początkowa</label>
				<input
					id="newAccountValue"
					type="number"
					step="0.01"
					bind:value
					placeholder="0.00"
					class="input"
				/>
			</div>
		</div>

		<footer class="flex justify-end gap-2 px-5 py-4 border-t border-surface-200-800">
			<button type="button" class="btn preset-tonal-surface" onclick={onClose}>Anuluj</button>
			<button
				type="button"
				class="btn preset-filled-primary-500"
				disabled={creating}
				onclick={onCreate}
			>
				{creating ? 'Tworzenie...' : 'Utwórz konto'}
			</button>
		</footer>
	</div>
</div>
