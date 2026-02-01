<script lang="ts">
	import { goto } from '$app/navigation';
	import { Card, CardHeader, CardTitle, CardContent, Table, formatPLN } from '@mskalski/home-ui';

	export let data;
</script>

<svelte:head>
	<title>Snapshots | Finansowa Forteca</title>
</svelte:head>

<div class="page-header">
	<div>
		<h1 class="page-title">Snapshots</h1>
		<p class="page-description">Historia zapisanych wartości netto</p>
	</div>
	<button class="btn btn-primary" on:click={() => goto('/snapshots/new')}> + Nowy Snapshot </button>
</div>

<Card>
	<CardHeader>
		<CardTitle>📊 Wszystkie Snapshots</CardTitle>
	</CardHeader>
	<CardContent>
		{#if data.snapshots.length === 0}
			<div class="empty-state">
				<p>Brak zapisanych snapshotów</p>
				<button class="btn btn-secondary" on:click={() => goto('/snapshots/new')}>
					Utwórz pierwszy snapshot
				</button>
			</div>
		{:else}
			<Table
				headers={['Data', 'Wartość Netto', 'Notatki', 'Akcje']}
				mobileCardView
				class="snapshots-table"
			>
				{#each data.snapshots as snapshot}
					<tr>
						<td data-label="Data" class="date-cell">
							{new Date(snapshot.date).toLocaleDateString('pl-PL', {
								year: 'numeric',
								month: 'long',
								day: 'numeric'
							})}
						</td>
						<td data-label="Wartość Netto" class="value-cell"
							>{formatPLN(snapshot.total_net_worth)}</td
						>
						<td data-label="Notatki" class="notes-cell">{snapshot.notes || '—'}</td>
						<td data-label="Akcje" class="actions-cell">
							<button
								class="btn-edit"
								on:click={() => goto(`/snapshots/${snapshot.id}/edit`)}
								title="Edytuj snapshot"
							>
								✏️ Edytuj
							</button>
						</td>
					</tr>
				{/each}
			</Table>
		{/if}
	</CardContent>
</Card>

<style>
	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
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

	.btn {
		padding: var(--size-3) var(--size-5);
		border: none;
		border-radius: var(--radius-2);
		font-weight: var(--font-weight-6);
		font-size: var(--font-size-2);
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-primary {
		background: var(--color-primary);
		color: var(--nord6);
	}

	.btn-primary:hover {
		background: var(--nord9);
	}

	.btn-secondary {
		background: transparent;
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}

	.btn-secondary:hover {
		background: var(--color-accent);
	}

	.empty-state {
		text-align: center;
		padding: var(--size-8) var(--size-4);
		color: var(--color-text-secondary);
	}

	.empty-state p {
		margin-bottom: var(--size-4);
		font-size: var(--font-size-3);
	}

	.table-container {
		overflow-x: auto;
	}

	.snapshots-table {
		width: 100%;
		border-collapse: collapse;
	}

	.snapshots-table thead {
		border-bottom: 2px solid var(--color-border);
	}

	.snapshots-table th {
		text-align: left;
		padding: var(--size-3) var(--size-4);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.snapshots-table tbody tr {
		border-bottom: 1px solid var(--color-border);
		transition: background-color 0.2s;
	}

	.snapshots-table tbody tr:hover {
		background-color: var(--color-accent);
	}

	.snapshots-table td {
		padding: var(--size-4);
		font-size: var(--font-size-2);
	}

	.date-cell {
		font-weight: var(--font-weight-6);
		color: var(--color-text);
	}

	.value-cell {
		font-weight: var(--font-weight-6);
		color: var(--color-primary);
	}

	.notes-cell {
		color: var(--color-text-secondary);
		font-style: italic;
	}

	.actions-cell {
		text-align: right;
	}

	.btn-edit {
		padding: var(--size-2) var(--size-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: transparent;
		color: var(--color-text);
		font-size: var(--font-size-1);
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-edit:hover {
		background: var(--color-primary);
		color: var(--nord6);
		border-color: var(--color-primary);
	}

	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			gap: var(--size-4);
		}

		.actions-cell {
			text-align: left;
		}
	}
</style>
