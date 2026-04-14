<script lang="ts">
	import { goto } from '$app/navigation';
	import { formatPLN } from '$lib/utils/format';
	import { Plus, Camera, Pencil } from 'lucide-svelte';

	export let data;
</script>

<svelte:head>
	<title>Snapshots | Finansowa Forteca</title>
</svelte:head>

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Snapshots</h1>
		<p class="text-surface-700-300 text-sm">Historia zapisanych wartości netto</p>
	</div>
	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto gap-2"
		on:click={() => goto('/snapshots/new')}
	>
		<Plus size={16} />
		Nowy Snapshot
	</button>
</div>

<div class="card preset-filled-surface-100-900 p-4 space-y-4">
	<header>
		<h3 class="h3 flex items-center gap-2"><Camera size={20} /> Wszystkie Snapshots</h3>
	</header>

	{#if data.snapshots.length === 0}
		<div class="text-center py-12 space-y-4 text-surface-700-300">
			<p>Brak zapisanych snapshotów</p>
			<button
				type="button"
				class="btn preset-tonal-primary"
				on:click={() => goto('/snapshots/new')}
			>
				Utwórz pierwszy snapshot
			</button>
		</div>
	{:else}
		<div class="table-wrap">
			<table class="table table-hover">
				<thead>
					<tr>
						<th>Data</th>
						<th>Wartość Netto</th>
						<th>Notatki</th>
						<th class="text-right">Akcje</th>
					</tr>
				</thead>
				<tbody>
					{#each data.snapshots as snapshot}
						<tr>
							<td class="font-medium">
								{new Date(snapshot.date).toLocaleDateString('pl-PL', {
									year: 'numeric',
									month: 'long',
									day: 'numeric'
								})}
							</td>
							<td class="font-semibold text-primary-600-400"
								>{formatPLN(snapshot.total_net_worth)}</td
							>
							<td class="italic text-surface-700-300">{snapshot.notes || '—'}</td>
							<td class="text-right">
								<button
									type="button"
									class="btn btn-sm preset-tonal-primary gap-1"
									on:click={() => goto(`/snapshots/${snapshot.id}/edit`)}
									title="Edytuj snapshot"
								>
									<Pencil size={14} />
									Edytuj
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
