import { describe, it, expect } from 'vitest';
import { computePresetBounds, resolveRangeParams, isRangePreset, isRangeValue } from './dateRange';

const NOW = new Date('2026-05-23T12:00:00Z');

describe('isRangePreset / isRangeValue', () => {
	it('accepts known presets', () => {
		for (const v of ['1m', '3m', '6m', '1y', '3y', '5y', 'all']) {
			expect(isRangePreset(v)).toBe(true);
			expect(isRangeValue(v)).toBe(true);
		}
	});

	it('treats custom as a value but not a preset', () => {
		expect(isRangePreset('custom')).toBe(false);
		expect(isRangeValue('custom')).toBe(true);
	});

	it('rejects unknown strings', () => {
		expect(isRangePreset('foo')).toBe(false);
		expect(isRangeValue(null)).toBe(false);
	});
});

describe('computePresetBounds', () => {
	it('returns nulls for "all"', () => {
		expect(computePresetBounds('all', NOW)).toEqual({ from: null, to: null });
	});

	it('returns now-1m for "1m"', () => {
		expect(computePresetBounds('1m', NOW)).toEqual({ from: '2026-04-23', to: '2026-05-23' });
	});

	it('returns now-3m for "3m"', () => {
		expect(computePresetBounds('3m', NOW)).toEqual({ from: '2026-02-23', to: '2026-05-23' });
	});

	it('returns now-1y for "1y"', () => {
		expect(computePresetBounds('1y', NOW)).toEqual({ from: '2025-05-23', to: '2026-05-23' });
	});

	it('returns now-5y for "5y"', () => {
		expect(computePresetBounds('5y', NOW)).toEqual({ from: '2021-05-23', to: '2026-05-23' });
	});

	it('returns now-6m for "6m"', () => {
		expect(computePresetBounds('6m', NOW)).toEqual({ from: '2025-11-23', to: '2026-05-23' });
	});

	it('returns now-3y for "3y"', () => {
		expect(computePresetBounds('3y', NOW)).toEqual({ from: '2023-05-23', to: '2026-05-23' });
	});
});

describe('resolveRangeParams', () => {
	it('returns the default when no params present', () => {
		const r = resolveRangeParams(new URLSearchParams(), NOW);
		expect(r).toEqual({ range: 'all', dateFrom: null, dateTo: null });
	});

	it('honors an explicit preset', () => {
		const r = resolveRangeParams(new URLSearchParams('range=1y'), NOW);
		expect(r).toEqual({ range: '1y', dateFrom: '2025-05-23', dateTo: '2026-05-23' });
	});

	it('honors a custom range when ?range=custom is set', () => {
		const r = resolveRangeParams(
			new URLSearchParams('range=custom&date_from=2024-01-01&date_to=2024-12-31'),
			NOW
		);
		expect(r).toEqual({ range: 'custom', dateFrom: '2024-01-01', dateTo: '2024-12-31' });
	});

	it('infers custom when only date_from/date_to are set', () => {
		const r = resolveRangeParams(new URLSearchParams('date_from=2024-01-01'), NOW);
		expect(r).toEqual({ range: 'custom', dateFrom: '2024-01-01', dateTo: null });
	});

	it('preset overrides any stale date_from/date_to in the URL', () => {
		const r = resolveRangeParams(
			new URLSearchParams('range=3m&date_from=2020-01-01&date_to=2020-06-30'),
			NOW
		);
		expect(r).toEqual({ range: '3m', dateFrom: '2026-02-23', dateTo: '2026-05-23' });
	});
});
