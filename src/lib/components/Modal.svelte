<script lang="ts">
	export let open = false;
	export let title = '';
	export let onConfirm: () => void;
	export let onCancel: () => void;

	function handleOverlayClick(event: MouseEvent) {
		if (event.target === event.currentTarget) {
			onCancel();
		}
	}
</script>

{#if open}
	<div class="modal-overlay" on:click={handleOverlayClick} role="presentation">
		<div class="modal-dialog" role="dialog" aria-modal="true" aria-labelledby="modal-title">
			<div class="modal-header">
				<h2 id="modal-title" class="modal-title">{title}</h2>
			</div>
			<div class="modal-content">
				<slot />
			</div>
			<div class="modal-actions">
				<button type="button" class="btn btn-secondary" on:click={onCancel}>Anuluj</button>
				<button type="button" class="btn btn-danger" on:click={onConfirm}>Potwierdź</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: var(--size-4);
	}

	.modal-dialog {
		background: var(--color-background);
		border-radius: var(--radius-3);
		box-shadow: var(--shadow-6);
		max-width: 500px;
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: var(--size-4);
		padding: var(--size-6);
	}

	.modal-header {
		border-bottom: 1px solid var(--color-border);
		padding-bottom: var(--size-3);
	}

	.modal-title {
		font-size: var(--font-size-4);
		font-weight: var(--font-weight-7);
		color: var(--color-text);
		margin: 0;
	}

	.modal-content {
		color: var(--color-text);
		font-size: var(--font-size-2);
		line-height: 1.6;
	}

	.modal-actions {
		display: flex;
		gap: var(--size-3);
		justify-content: flex-end;
		padding-top: var(--size-3);
		border-top: 1px solid var(--color-border);
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

	.btn-secondary {
		background: transparent;
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}

	.btn-secondary:hover {
		background: var(--color-accent);
	}

	.btn-danger {
		background: var(--nord11);
		color: var(--nord6);
	}

	.btn-danger:hover {
		background: var(--nord12);
	}

	@media (max-width: 768px) {
		.modal-dialog {
			max-width: 100%;
			margin: var(--size-4);
		}
	}
</style>
