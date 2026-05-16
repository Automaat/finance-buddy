<script lang="ts">
	export let inputId: string;
	export let label: string;
	export let meta = '';
	export let value: number;
	export let onRemove: () => void;
</script>

<div class="form-group-with-remove">
	<div class="form-group">
		<label for={inputId} class="form-label">
			{label}
			{#if meta}
				<span class="label-meta">{meta}</span>
			{/if}
		</label>
		<input
			id={inputId}
			type="number"
			step="0.01"
			bind:value
			placeholder="0.00"
			class="form-input"
		/>
	</div>
	<button type="button" class="btn-remove" on:click={onRemove} title="Usuń pole"> × </button>
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

	.label-meta {
		color: var(--color-text-secondary);
		font-weight: var(--font-weight-4);
		font-size: var(--font-size-1);
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

	.form-group-with-remove {
		display: flex;
		gap: var(--size-2);
		align-items: flex-start;
		margin-bottom: var(--size-4);
	}

	.form-group-with-remove .form-group {
		flex: 1;
		margin-bottom: 0;
	}

	.btn-remove {
		flex-shrink: 0;
		width: var(--tap-target-min);
		height: var(--tap-target-min);
		margin-top: 28px;
		padding: 0;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: transparent;
		color: var(--color-text-secondary);
		font-size: var(--font-size-4);
		line-height: 1;
		cursor: pointer;
		transition: all 0.2s;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.btn-remove:hover {
		background: var(--nord11);
		color: var(--nord6);
		border-color: var(--nord11);
	}

	@media (max-width: 640px) {
		.btn-remove {
			margin-top: 32px;
		}
	}
</style>
