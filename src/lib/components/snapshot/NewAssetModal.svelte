<script lang="ts">
	interface Props {
		name?: string;
		value?: number;
		creating?: boolean;
		onCreate: () => void;
		onClose: () => void;
	}

	let {
		name = $bindable(''),
		value = $bindable(0),
		creating = false,
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
		aria-labelledby="new-asset-modal-title"
		tabindex="-1"
	>
		<header class="flex items-center justify-between px-5 py-4 border-b border-surface-200-800">
			<h2 id="new-asset-modal-title" class="h4 font-bold">Dodaj nowy majątek</h2>
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
				<label for="newAssetName" class="text-sm font-semibold">Nazwa *</label>
				<input
					id="newAssetName"
					type="text"
					bind:value={name}
					placeholder="np. Mieszkanie Poznań, Rower"
					class="input"
					required
				/>
			</div>

			<div class="flex flex-col gap-1">
				<label for="newAssetValue" class="text-sm font-semibold">Wartość początkowa</label>
				<input
					id="newAssetValue"
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
				{creating ? 'Tworzenie...' : 'Utwórz majątek'}
			</button>
		</footer>
	</div>
</div>
