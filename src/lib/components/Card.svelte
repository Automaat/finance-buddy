<script lang="ts">
	let className = '';
	export { className as class };
</script>

<div class="card {className}">
	<slot />
</div>

<style>
	.card {
		background: var(--color-bg-card);
		border-radius: var(--radius-3);
		border: 1px solid var(--color-border);
		box-shadow: var(--shadow-2);
		overflow: hidden;
	}
</style>
