import { describe, it, expect } from 'vitest';
import { sumsToHundred } from './allocation';

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
