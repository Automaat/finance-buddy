<script lang="ts">
	interface Props {
		open: boolean;
		title: string;
		onConfirm?: () => void;
		onCancel?: () => void;
		confirmText?: string;
		cancelText?: string;
		confirmDisabled?: boolean;
		confirmVariant?: 'primary' | 'danger';
		hideFooter?: boolean;
		children?: import('svelte').Snippet;
	}

	let {
		open,
		title,
		onConfirm,
		onCancel,
		confirmText = 'Zapisz',
		cancelText = 'Anuluj',
		confirmDisabled = false,
		confirmVariant = 'primary',
		hideFooter = false,
		children
	}: Props = $props();

	function handleBackdropClick(event: MouseEvent) {
		if (event.target === event.currentTarget) onCancel?.();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (!open) return;
		if (event.key === 'Escape') {
			onCancel?.();
			return;
		}
		if (event.key === 'Enter' && !confirmDisabled) {
			// Only confirm when Enter wouldn't lose the user input that's
			// in focus: text inputs / selects are safe (single-line submit),
			// textareas insert newlines, buttons handle their own activation,
			// content-editable could have any meaning so we skip it.
			const t = event.target as HTMLElement | null;
			if (!t) {
				onConfirm?.();
				return;
			}
			const tag = t.tagName;
			const editable = t.isContentEditable;
			if (tag === 'TEXTAREA' || tag === 'BUTTON' || editable) return;
			if (tag === 'INPUT' || tag === 'SELECT') {
				event.preventDefault();
			}
			onConfirm?.();
		}
	}

	const confirmClass = $derived(
		confirmVariant === 'danger' ? 'preset-filled-error-500' : 'preset-filled-primary-500'
	);
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-surface-950/60 backdrop-blur-sm p-4"
		role="presentation"
		onclick={handleBackdropClick}
	>
		<div
			class="card preset-filled-surface-50-950 w-full max-w-lg max-h-[90vh] flex flex-col shadow-xl"
			role="dialog"
			aria-modal="true"
			aria-labelledby="modal-title"
			tabindex="-1"
		>
			<header class="flex items-center justify-between px-5 py-4 border-b border-surface-200-800">
				<h2 id="modal-title" class="h4 font-bold">{title}</h2>
				<button
					type="button"
					class="btn-icon btn-icon-sm"
					aria-label="Zamknij"
					onclick={() => onCancel?.()}
				>
					×
				</button>
			</header>

			<div class="flex-1 overflow-y-auto px-5 py-4">
				{@render children?.()}
			</div>

			{#if !hideFooter}
				<footer class="flex justify-end gap-2 px-5 py-4 border-t border-surface-200-800">
					<button type="button" class="btn preset-tonal-surface" onclick={() => onCancel?.()}>
						{cancelText}
					</button>
					<button
						type="button"
						class="btn {confirmClass}"
						disabled={confirmDisabled}
						onclick={() => onConfirm?.()}
					>
						{confirmText}
					</button>
				</footer>
			{/if}
		</div>
	</div>
{/if}
