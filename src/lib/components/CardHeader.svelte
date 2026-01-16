<script lang="ts">
	let className = '';
	export { className as class };
</script>

<div class="card-header {className}">
	<slot />
</div>

<style>
	.card-header {
		padding: var(--size-5) var(--size-6);
		border-bottom: 1px solid var(--color-border);
	}
</style>
