<script lang="ts">
	let className = '';
	export { className as class };
</script>

<h3 class="card-title {className}">
	<slot />
</h3>

<style>
	.card-title {
		font-size: var(--font-size-2);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		margin: 0;
	}
</style>
