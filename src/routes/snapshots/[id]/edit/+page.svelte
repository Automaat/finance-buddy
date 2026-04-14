<script lang="ts">
	import SnapshotForm from '$lib/components/SnapshotForm.svelte';

	export let data;
</script>

<svelte:head>
	<title>Edytuj Snapshot | Finansowa Forteca</title>
</svelte:head>

<div class="mb-6 space-y-1">
	<h1 class="h2">Edytuj Snapshot</h1>
	<p class="text-surface-700-300 text-sm">Zaktualizuj wartości z dnia {data.snapshot.date}</p>
</div>

<SnapshotForm
	editingSnapshot={data.snapshot}
	assets={data.assets}
	liabilities={data.liabilities}
	physicalAssets={data.physicalAssets}
	personas={data.personas}
/>
