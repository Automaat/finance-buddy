<script lang="ts">
	let className = '';
	export { className as class };
</script>

<div class="card-content {className}">
	<slot />
</div>

<style>
	.card-content {
		padding: var(--size-6);
	}
</style>
