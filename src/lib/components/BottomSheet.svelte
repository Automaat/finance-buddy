<script lang="ts">
	interface Props {
		open: boolean;
		title?: string;
		onClose: () => void;
		children?: import('svelte').Snippet;
	}

	let { open, title, onClose, children }: Props = $props();

	const DISMISS_PX = 100;

	let sheetEl = $state<HTMLDivElement | null>(null);
	let dragStartY = $state<number | null>(null);
	let dragDeltaY = $state(0);

	const translate = $derived(dragDeltaY > 0 ? dragDeltaY : 0);

	$effect(() => {
		if (open) sheetEl?.focus();
	});

	function handleBackdropClick(event: MouseEvent) {
		if (event.target === event.currentTarget) onClose();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (!open) return;
		if (event.key === 'Escape') onClose();
	}

	function onPointerDown(event: PointerEvent) {
		const handle = (event.target as HTMLElement).closest('[data-sheet-drag]');
		if (!handle) return;
		dragStartY = event.clientY;
		dragDeltaY = 0;
		(event.currentTarget as Element).setPointerCapture(event.pointerId);
	}

	function onPointerMove(event: PointerEvent) {
		if (dragStartY === null) return;
		dragDeltaY = event.clientY - dragStartY;
	}

	function onPointerUp() {
		if (dragStartY === null) return;
		const shouldDismiss = dragDeltaY > DISMISS_PX;
		dragStartY = null;
		dragDeltaY = 0;
		if (shouldDismiss) onClose();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<div
		class="fixed inset-0 z-40 bg-surface-950/60 backdrop-blur-sm"
		role="presentation"
		onclick={handleBackdropClick}
	>
		<div
			bind:this={sheetEl}
			class="absolute bottom-0 left-0 right-0 bg-surface-100-900 rounded-t-2xl shadow-2xl max-h-[80vh] flex flex-col"
			style:transform="translateY({translate}px)"
			style:transition={dragStartY === null ? 'transform 0.2s ease-out' : 'none'}
			role="dialog"
			aria-modal="true"
			aria-label={title ?? 'Więcej'}
			tabindex="-1"
			onpointerdown={onPointerDown}
			onpointermove={onPointerMove}
			onpointerup={onPointerUp}
			onpointercancel={onPointerUp}
		>
			<div
				data-sheet-drag
				class="flex flex-col items-center pt-2 pb-1 cursor-grab select-none touch-none"
			>
				<div class="w-10 h-1.5 rounded-full bg-surface-300-700"></div>
				{#if title}
					<h2 class="mt-2 text-base font-semibold">{title}</h2>
				{/if}
			</div>
			<div class="flex-1 overflow-y-auto p-2 pb-[env(safe-area-inset-bottom)]">
				{@render children?.()}
			</div>
		</div>
	</div>
{/if}
