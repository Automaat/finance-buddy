<script lang="ts">
	import SnapshotForm from '$lib/components/SnapshotForm.svelte';

	export let data;
</script>

<svelte:head>
	<title>Nowy Snapshot | Finansowa Forteca</title>
</svelte:head>

<div class="page-header">
	<div>
		<h1 class="page-title">Nowy Snapshot</h1>
		<p class="page-description">Zaktualizuj wartości wszystkich kont</p>
	</div>
</div>

<SnapshotForm
	editingSnapshot={null}
	assets={data.assets}
	liabilities={data.liabilities}
	physicalAssets={data.physicalAssets}
	personas={data.personas}
/>

<style>
	.page-header {
		margin-bottom: var(--size-6);
	}

	.page-title {
		font-size: var(--font-size-6);
		font-weight: var(--font-weight-7);
		color: var(--color-text);
		margin: 0 0 var(--size-2) 0;
	}

	.page-description {
		color: var(--color-text-secondary);
		font-size: var(--font-size-2);
		margin: 0;
	}
</style>
