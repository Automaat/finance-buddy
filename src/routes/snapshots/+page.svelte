<script lang="ts">
	import { goto } from '$app/navigation';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { Plus, Camera, Pencil } from 'lucide-svelte';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();
</script>

<svelte:head>
	<title>Migawki | Finansowa Forteca</title>
</svelte:head>

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Migawki</h1>
		<p class="text-surface-700-300 text-sm">Historia zapisanych wartości netto</p>
	</div>
	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto gap-2"
		onclick={() => goto('/snapshots/new')}
	>
		<Plus size={16} />
		Nowa migawka
	</button>
</div>

<div class="card preset-filled-surface-100-900 p-4 space-y-4">
	<header>
		<h3 class="h3 flex items-center gap-2"><Camera size={20} /> Wszystkie migawki</h3>
	</header>

	{#await data.snapshots}
		<div role="status" aria-live="polite" aria-label="Ładowanie migawek" class="table-wrap">
			<table class="table">
				<thead>
					<tr>
						<th>Data</th>
						<th>Wartość Netto</th>
						<th>Notatki</th>
						<th class="text-right">Akcje</th>
					</tr>
				</thead>
				<tbody>
					{#each { length: 5 } as _, i (i)}
						<tr>
							<td><Skeleton height="1rem" width="70%" /></td>
							<td><Skeleton height="1rem" width="60%" /></td>
							<td><Skeleton height="1rem" width="85%" /></td>
							<td>
								<div class="flex justify-end">
									<Skeleton height="2rem" width="5rem" rounded="md" />
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:then snapshots}
		{#if snapshots.length === 0}
			<div class="text-center py-12 space-y-4 text-surface-700-300">
				<p>Brak zapisanych migawek</p>
				<button
					type="button"
					class="btn preset-tonal-primary"
					onclick={() => goto('/snapshots/new')}
				>
					Utwórz pierwszą migawkę
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
						{#each snapshots as snapshot}
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
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										title="Edytuj migawkę"
										onclick={() => goto(`/snapshots/${snapshot.id}/edit`)}
									>
										<Pencil size={16} />
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{:catch err}
		<div class="card preset-filled-error-500 p-4">
			<p class="font-semibold">Nie udało się załadować migawek.</p>
			<p class="text-sm">{err?.message ?? 'Spróbuj ponownie później.'}</p>
		</div>
	{/await}
</div>
