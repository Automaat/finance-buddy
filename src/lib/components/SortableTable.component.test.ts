import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import SortableTable, { type SortableColumn } from './SortableTable.svelte';

beforeAll(() => {
	(globalThis as { IntersectionObserver?: unknown }).IntersectionObserver = class {
		observe(): void {}
		disconnect(): void {}
		unobserve(): void {}
		takeRecords(): unknown[] {
			return [];
		}
		readonly root = null;
		readonly rootMargin = '';
		readonly thresholds: ReadonlyArray<number> = [];
	};
});

interface Row {
	id: number;
	name: string;
	amount: number;
}

interface PageState {
	url: URL;
	state: Record<string, unknown>;
}

const { pageStore, replaceStateSpy } = vi.hoisted(() => {
	let value: PageState = { url: new URL('http://localhost/test'), state: {} };
	const subs = new Set<(v: PageState) => void>();
	const store = {
		subscribe(fn: (v: PageState) => void): () => void {
			subs.add(fn);
			fn(value);
			return () => subs.delete(fn);
		},
		set(next: PageState): void {
			value = next;
			subs.forEach((fn) => fn(value));
		}
	};
	const spy = vi.fn();
	return { pageStore: store, replaceStateSpy: spy };
});

vi.mock('$app/navigation', () => ({
	replaceState: (url: URL | string, state: unknown) => {
		const nextUrl = typeof url === 'string' ? new URL(url) : url;
		replaceStateSpy(nextUrl, state);
		pageStore.set({ url: nextUrl, state: (state ?? {}) as Record<string, unknown> });
	}
}));

vi.mock('$app/stores', () => ({
	page: pageStore
}));

const columns: SortableColumn<Row>[] = [
	{ key: 'name', label: 'Nazwa', sortable: true, accessor: (r) => r.name },
	{ key: 'amount', label: 'Kwota', sortable: true, accessor: (r) => r.amount },
	{ key: 'actions', label: 'Akcje', align: 'right' }
];

const items: Row[] = [
	{ id: 1, name: 'B', amount: 20 },
	{ id: 2, name: 'A', amount: 30 },
	{ id: 3, name: 'C', amount: 10 }
];

const rowSnippet = createRawSnippet<[Row]>((getRow) => ({
	render: () => {
		const r = getRow();
		return `<tr data-testid="row-${r.id}"><td>${r.name}</td><td>${r.amount}</td><td>x</td></tr>`;
	}
}));

function setUrl(search: string) {
	pageStore.set({
		url: new URL(`http://localhost/test${search}`),
		state: {}
	});
}

describe('SortableTable component', () => {
	it('renders all column headers', () => {
		setUrl('');
		const { getByText } = render(SortableTable<Row>, {
			props: { columns, items, row: rowSnippet }
		});
		expect(getByText('Nazwa')).toBeTruthy();
		expect(getByText('Kwota')).toBeTruthy();
		expect(getByText('Akcje')).toBeTruthy();
	});

	it('renders sortable headers as buttons and non-sortable as plain th', () => {
		setUrl('');
		const { container } = render(SortableTable<Row>, {
			props: { columns, items, row: rowSnippet }
		});
		const buttons = container.querySelectorAll('thead button.sort-header');
		expect(buttons.length).toBe(2);
		const ths = container.querySelectorAll('thead th');
		expect(ths.length).toBe(3);
	});

	it('renders rows in original order when no sort param', () => {
		setUrl('');
		const { container } = render(SortableTable<Row>, {
			props: { columns, items, row: rowSnippet }
		});
		const rows = container.querySelectorAll('tbody tr');
		expect(rows[0].getAttribute('data-testid')).toBe('row-1');
		expect(rows[1].getAttribute('data-testid')).toBe('row-2');
		expect(rows[2].getAttribute('data-testid')).toBe('row-3');
	});

	it('applies asc sort from URL param', () => {
		setUrl('?sort=name:asc');
		const { container } = render(SortableTable<Row>, {
			props: { columns, items, row: rowSnippet }
		});
		const rows = container.querySelectorAll('tbody tr');
		expect(rows[0].getAttribute('data-testid')).toBe('row-2'); // A
		expect(rows[1].getAttribute('data-testid')).toBe('row-1'); // B
		expect(rows[2].getAttribute('data-testid')).toBe('row-3'); // C
	});

	it('applies desc sort from URL param', () => {
		setUrl('?sort=amount:desc');
		const { container } = render(SortableTable<Row>, {
			props: { columns, items, row: rowSnippet }
		});
		const rows = container.querySelectorAll('tbody tr');
		expect(rows[0].getAttribute('data-testid')).toBe('row-2'); // 30
		expect(rows[1].getAttribute('data-testid')).toBe('row-1'); // 20
		expect(rows[2].getAttribute('data-testid')).toBe('row-3'); // 10
	});

	it('respects a custom paramName', () => {
		setUrl('?sortA=name:asc&sort=amount:desc');
		const { container } = render(SortableTable<Row>, {
			props: { columns, items, row: rowSnippet, paramName: 'sortA' }
		});
		const rows = container.querySelectorAll('tbody tr');
		expect(rows[0].getAttribute('data-testid')).toBe('row-2'); // A
	});

	it('cycles sort state via replaceState on header click', async () => {
		setUrl('');
		replaceStateSpy.mockClear();
		const { container } = render(SortableTable<Row>, {
			props: { columns, items, row: rowSnippet }
		});
		const nameButton = container.querySelector('thead button.sort-header') as HTMLButtonElement;
		expect(nameButton).toBeTruthy();

		await fireEvent.click(nameButton);
		expect(replaceStateSpy).toHaveBeenCalledTimes(1);
		expect((replaceStateSpy.mock.calls[0][0] as URL).searchParams.get('sort')).toBe('name:asc');

		await fireEvent.click(nameButton);
		expect((replaceStateSpy.mock.calls[1][0] as URL).searchParams.get('sort')).toBe('name:desc');

		await fireEvent.click(nameButton);
		expect((replaceStateSpy.mock.calls[2][0] as URL).searchParams.get('sort')).toBeNull();
	});

	it('sets aria-sort on sorted column', () => {
		setUrl('?sort=name:asc');
		const { container } = render(SortableTable<Row>, {
			props: { columns, items, row: rowSnippet }
		});
		const ths = container.querySelectorAll('thead th');
		expect(ths[0].getAttribute('aria-sort')).toBe('ascending');
		expect(ths[1].getAttribute('aria-sort')).toBe('none');
		expect(ths[2].getAttribute('aria-sort')).toBeNull();
	});

	it('renders emptyMessage when items are empty', () => {
		setUrl('');
		const { getByText } = render(SortableTable<Row>, {
			props: { columns, items: [], row: rowSnippet, emptyMessage: 'Brak danych' }
		});
		expect(getByText('Brak danych')).toBeTruthy();
	});
});
