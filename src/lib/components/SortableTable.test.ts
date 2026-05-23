import { describe, it, expect } from 'vitest';
import {
	parseSortParam,
	formatSortParam,
	nextSortDirection,
	compareValues,
	sortRows,
	type SortableColumn
} from './SortableTable.svelte';

describe('parseSortParam', () => {
	it('returns null for empty input', () => {
		expect(parseSortParam(null)).toBeNull();
		expect(parseSortParam('')).toBeNull();
	});

	it('parses key:asc', () => {
		expect(parseSortParam('name:asc')).toEqual({ key: 'name', direction: 'asc' });
	});

	it('parses key:desc', () => {
		expect(parseSortParam('value:desc')).toEqual({ key: 'value', direction: 'desc' });
	});

	it('returns null for invalid direction', () => {
		expect(parseSortParam('name:bogus')).toBeNull();
		expect(parseSortParam('name')).toBeNull();
	});
});

describe('formatSortParam', () => {
	it('returns empty string for null', () => {
		expect(formatSortParam(null)).toBe('');
	});

	it('serializes sort state to key:direction', () => {
		expect(formatSortParam({ key: 'amount', direction: 'desc' })).toBe('amount:desc');
	});
});

describe('nextSortDirection', () => {
	it('starts at asc for a new column', () => {
		expect(nextSortDirection(null, 'name')).toEqual({ key: 'name', direction: 'asc' });
	});

	it('switches to a different column at asc', () => {
		expect(nextSortDirection({ key: 'name', direction: 'desc' }, 'value')).toEqual({
			key: 'value',
			direction: 'asc'
		});
	});

	it('cycles asc → desc on same column', () => {
		expect(nextSortDirection({ key: 'name', direction: 'asc' }, 'name')).toEqual({
			key: 'name',
			direction: 'desc'
		});
	});

	it('cycles desc → none on same column', () => {
		expect(nextSortDirection({ key: 'name', direction: 'desc' }, 'name')).toBeNull();
	});
});

describe('compareValues', () => {
	it('compares numbers numerically', () => {
		expect(compareValues(2, 10)).toBeLessThan(0);
		expect(compareValues(10, 2)).toBeGreaterThan(0);
	});

	it('compares strings with Polish collation', () => {
		expect(compareValues('Ąla', 'Bla')).toBeLessThan(0);
	});

	it('compares dates by timestamp', () => {
		const a = new Date('2024-01-01');
		const b = new Date('2024-06-01');
		expect(compareValues(a, b)).toBeLessThan(0);
	});

	it('puts nullish values last', () => {
		expect(compareValues(null, 1)).toBeGreaterThan(0);
		expect(compareValues(1, null)).toBeLessThan(0);
		expect(compareValues(null, null)).toBe(0);
	});

	it('treats NaN as nullish (sorts last)', () => {
		expect(compareValues(NaN, 1)).toBeGreaterThan(0);
		expect(compareValues(1, NaN)).toBeLessThan(0);
		expect(compareValues(NaN, NaN)).toBe(0);
	});
});

describe('sortRows', () => {
	interface Row {
		name: string;
		amount: number;
	}
	const columns: SortableColumn<Row>[] = [
		{ key: 'name', label: 'Nazwa', sortable: true, accessor: (r) => r.name },
		{ key: 'amount', label: 'Kwota', sortable: true, accessor: (r) => r.amount }
	];
	const rows: Row[] = [
		{ name: 'B', amount: 2 },
		{ name: 'A', amount: 3 },
		{ name: 'C', amount: 1 }
	];

	it('returns original order when sort is null', () => {
		expect(sortRows(rows, columns, null)).toEqual(rows);
	});

	it('sorts ascending by accessor', () => {
		const sorted = sortRows(rows, columns, { key: 'name', direction: 'asc' });
		expect(sorted.map((r) => r.name)).toEqual(['A', 'B', 'C']);
	});

	it('sorts descending by accessor', () => {
		const sorted = sortRows(rows, columns, { key: 'amount', direction: 'desc' });
		expect(sorted.map((r) => r.amount)).toEqual([3, 2, 1]);
	});

	it('does not mutate the input array', () => {
		const copy = [...rows];
		sortRows(rows, columns, { key: 'name', direction: 'asc' });
		expect(rows).toEqual(copy);
	});

	it('returns original order when column has no accessor', () => {
		const cols: SortableColumn<Row>[] = [{ key: 'name', label: 'Nazwa', sortable: true }];
		expect(sortRows(rows, cols, { key: 'name', direction: 'asc' })).toEqual(rows);
	});

	it('returns original order for unknown sort key', () => {
		expect(sortRows(rows, columns, { key: 'unknown', direction: 'asc' })).toEqual(rows);
	});
});
