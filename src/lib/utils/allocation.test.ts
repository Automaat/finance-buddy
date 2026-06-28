import { describe, it, expect } from 'vitest';
import { sumsToHundred, topNWithOther, type AllocationBucket } from './allocation';

describe('sumsToHundred', () => {
	it('is true for exactly 100', () => {
		expect(sumsToHundred(100)).toBe(true);
	});

	it('absorbs float drift within the default tolerance', () => {
		expect(sumsToHundred(100.005)).toBe(true);
		expect(sumsToHundred(99.995)).toBe(true);
	});

	it('rejects values outside tolerance', () => {
		expect(sumsToHundred(99)).toBe(false);
		expect(sumsToHundred(100.5)).toBe(false);
	});

	it('supports an exact (zero-tolerance) match', () => {
		expect(sumsToHundred(100, 0)).toBe(true);
		expect(sumsToHundred(100.01, 0)).toBe(false);
	});
});

describe('topNWithOther', () => {
	const bucket = (category: string, value: number): AllocationBucket => ({
		category,
		owner_user_id: 1,
		value
	});

	it('returns positive entries unchanged when count is within the limit', () => {
		const items = [bucket('bank', 100), bucket('stock', 50)];
		expect(topNWithOther(items, 3)).toEqual(items);
	});

	it('drops non-finite and non-positive values before ranking', () => {
		expect(
			topNWithOther(
				[
					bucket('bank', 100),
					bucket('zero', 0),
					bucket('negative', -10),
					bucket('nan', Number.NaN),
					bucket('inf', Number.POSITIVE_INFINITY)
				],
				6
			)
		).toEqual([bucket('bank', 100)]);
	});

	it('sorts descending and keeps original order for ties', () => {
		expect(
			topNWithOther([bucket('first', 10), bucket('second', 20), bucket('third', 10)], 2)
		).toEqual([
			bucket('second', 20),
			bucket('first', 10),
			{ category: 'Inne', owner_user_id: null, value: 10, isOther: true }
		]);
	});

	it('omits Inne when filtering leaves no tail', () => {
		expect(topNWithOther([bucket('bank', 100), bucket('zero', 0)], 1)).toEqual([
			bucket('bank', 100)
		]);
	});

	it('groups every positive item under Inne when the limit is zero', () => {
		expect(topNWithOther([bucket('bank', 100), bucket('stock', 50)], 0)).toEqual([
			{ category: 'Inne', owner_user_id: null, value: 150, isOther: true }
		]);
	});
});
