<script lang="ts" module>
	export type SortDirection = 'asc' | 'desc';

	export interface SortableColumn<T> {
		key: string;
		label: string;
		sortable?: boolean;
		accessor?: (row: T) => string | number | Date | null | undefined;
		align?: 'left' | 'right' | 'center';
		thClass?: string;
	}

	export interface SortState {
		key: string;
		direction: SortDirection;
	}

	export function parseSortParam(raw: string | null): SortState | null {
		if (!raw) return null;
		const [key, dir] = raw.split(':');
		if (!key || (dir !== 'asc' && dir !== 'desc')) return null;
		return { key, direction: dir };
	}

	export function formatSortParam(sort: SortState | null): string {
		return sort ? `${sort.key}:${sort.direction}` : '';
	}

	export function nextSortDirection(current: SortState | null, key: string): SortState | null {
		if (!current || current.key !== key) return { key, direction: 'asc' };
		if (current.direction === 'asc') return { key, direction: 'desc' };
		return null;
	}

	export function compareValues(a: unknown, b: unknown): number {
		const aNull = a === null || a === undefined || (typeof a === 'number' && Number.isNaN(a));
		const bNull = b === null || b === undefined || (typeof b === 'number' && Number.isNaN(b));
		if (aNull && bNull) return 0;
		if (aNull) return 1;
		if (bNull) return -1;
		if (typeof a === 'number' && typeof b === 'number') return a - b;
		if (a instanceof Date && b instanceof Date) return a.getTime() - b.getTime();
		return String(a).localeCompare(String(b), 'pl');
	}

	export function sortRows<T>(
		items: T[],
		columns: SortableColumn<T>[],
		sort: SortState | null
	): T[] {
		if (!sort) return items;
		const col = columns.find((c) => c.key === sort.key);
		if (!col?.accessor) return items;
		const accessor = col.accessor;
		const factor = sort.direction === 'asc' ? 1 : -1;
		return [...items].sort((a, b) => factor * compareValues(accessor(a), accessor(b)));
	}
</script>

<script lang="ts" generics="T">
	import { replaceState } from '$app/navigation';
	import { page } from '$app/stores';
	import type { Snippet } from 'svelte';

	interface Props {
		columns: SortableColumn<T>[];
		items: T[];
		row: Snippet<[T]>;
		paramName?: string;
		emptyMessage?: string;
		tableClass?: string;
		getKey?: (item: T, index: number) => string | number;
	}

	let {
		columns,
		items,
		row,
		paramName = 'sort',
		emptyMessage,
		tableClass = 'table table-hover',
		getKey
	}: Props = $props();

	const sort = $derived(parseSortParam($page.url.searchParams.get(paramName)));
	const sortedItems = $derived(sortRows(items, columns, sort));

	function cycleSort(key: string): void {
		const url = new URL($page.url);
		const next = nextSortDirection(sort, key);
		if (next) url.searchParams.set(paramName, formatSortParam(next));
		else url.searchParams.delete(paramName);
		replaceState(url, $page.state);
	}

	let sentinel: HTMLDivElement;
	let stuck = $state(false);

	$effect(() => {
		if (!sentinel) return;
		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) stuck = !entry.isIntersecting;
			},
			{ threshold: 0 }
		);
		observer.observe(sentinel);
		return () => observer.disconnect();
	});
</script>

<div class="sortable-table">
	<div bind:this={sentinel} class="sticky-sentinel" aria-hidden="true"></div>
	<table class={tableClass}>
		<thead class:is-stuck={stuck}>
			<tr>
				{#each columns as col (col.key)}
					{@const isSorted = sort?.key === col.key}
					{@const ariaSort = isSorted
						? sort?.direction === 'asc'
							? 'ascending'
							: 'descending'
						: 'none'}
					<th
						class={col.thClass}
						class:text-right={col.align === 'right'}
						class:text-center={col.align === 'center'}
						aria-sort={col.sortable ? ariaSort : undefined}
						scope="col"
					>
						{#if col.sortable}
							<button
								type="button"
								class="sort-header"
								class:is-sorted={isSorted}
								class:align-right={col.align === 'right'}
								class:align-center={col.align === 'center'}
								onclick={() => cycleSort(col.key)}
							>
								<span>{col.label}</span>
								<span class="sort-indicator" aria-hidden="true">
									{#if isSorted && sort?.direction === 'asc'}
										▲
									{:else if isSorted && sort?.direction === 'desc'}
										▼
									{:else}
										<span class="sort-placeholder">⇅</span>
									{/if}
								</span>
							</button>
						{:else}
							{col.label}
						{/if}
					</th>
				{/each}
			</tr>
		</thead>
		<tbody>
			{#if sortedItems.length === 0 && emptyMessage}
				<tr>
					<td colspan={columns.length} class="text-center py-8 text-surface-700-300">
						{emptyMessage}
					</td>
				</tr>
			{:else}
				{#each sortedItems as item, i (getKey ? getKey(item, i) : i)}
					{@render row(item)}
				{/each}
			{/if}
		</tbody>
	</table>
</div>

<style>
	.sortable-table {
		position: relative;
		width: 100%;
	}

	.sticky-sentinel {
		height: 1px;
		width: 100%;
	}

	thead {
		position: sticky;
		top: 0;
		z-index: 10;
		background-color: var(--color-surface-100-900);
		transition: box-shadow 150ms ease;
	}

	thead.is-stuck {
		box-shadow: 0 2px 4px rgb(0 0 0 / 0.18);
	}

	.sort-header {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		cursor: pointer;
		background: transparent;
		border: 0;
		padding: 0;
		font: inherit;
		color: inherit;
		text-align: inherit;
	}

	.sort-header.align-right {
		flex-direction: row-reverse;
	}

	.sort-header.align-center {
		justify-content: center;
	}

	.sort-indicator {
		display: inline-flex;
		min-width: 0.85em;
		justify-content: center;
	}

	.sort-placeholder {
		opacity: 0.35;
	}

	.sort-header:hover .sort-placeholder {
		opacity: 0.7;
	}

	.sort-header:focus-visible {
		outline: 2px solid var(--color-primary-500);
		outline-offset: 2px;
		border-radius: 2px;
	}
</style>
