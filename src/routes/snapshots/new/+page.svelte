<script lang="ts">
	import SnapshotForm from '$lib/components/SnapshotForm.svelte';

	export let data;
</script>

<svelte:head>
	<title>Nowy Snapshot | Finansowa Forteca</title>
</svelte:head>

<div class="mb-6 space-y-1">
	<h1 class="h2">Nowy Snapshot</h1>
	<p class="text-surface-700-300 text-sm">Zaktualizuj wartości wszystkich kont</p>
</div>

<SnapshotForm
	editingSnapshot={null}
	assets={data.assets}
	liabilities={data.liabilities}
	physicalAssets={data.physicalAssets}
	personas={data.personas}
/>
