import { describe, it, expect } from 'vitest';
import { calculateNetWorth } from './calculations';

describe('calculateNetWorth', () => {
	it('calculates positive net worth', () => {
		expect(calculateNetWorth(100000, 30000)).toBe(70000);
	});

	it('calculates negative net worth', () => {
		expect(calculateNetWorth(30000, 100000)).toBe(-70000);
	});

	it('handles zero assets', () => {
		expect(calculateNetWorth(0, 50000)).toBe(-50000);
	});

	it('handles zero liabilities', () => {
		expect(calculateNetWorth(50000, 0)).toBe(50000);
	});

	it('handles both zero', () => {
		expect(calculateNetWorth(0, 0)).toBe(0);
	});

	it('handles large numbers', () => {
		expect(calculateNetWorth(1000000, 500000)).toBe(500000);
	});
});
