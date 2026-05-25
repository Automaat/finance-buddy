import { describe, it, expect } from 'vitest';
import { limitFromRules, monthsLeftIn, optimizeIKZE } from './ikzeOptimizer';

describe('limitFromRules', () => {
	it('returns employee IKZE limit for known year', () => {
		expect(limitFromRules(2026, 'employee')).toBe(11304);
	});

	it('returns B2B IKZE limit for known year', () => {
		expect(limitFromRules(2026, 'b2b')).toBe(16956);
	});

	it('returns null for unknown year', () => {
		expect(limitFromRules(1999, 'employee')).toBeNull();
	});
});

describe('monthsLeftIn', () => {
	it('returns 12 when current date is before the target year', () => {
		expect(monthsLeftIn(2027, new Date(2026, 5, 1))).toBe(12);
	});

	it('returns 0 when current date is after the target year', () => {
		expect(monthsLeftIn(2025, new Date(2026, 5, 1))).toBe(0);
	});

	it('returns 12 in January of the target year', () => {
		expect(monthsLeftIn(2026, new Date(2026, 0, 15))).toBe(12);
	});

	it('returns 1 in December of the target year', () => {
		expect(monthsLeftIn(2026, new Date(2026, 11, 31))).toBe(1);
	});
});

describe('optimizeIKZE', () => {
	it('splits remaining limit across remaining months', () => {
		const result = optimizeIKZE({
			year: 2026,
			limitKind: 'employee',
			alreadyContributed: 0,
			marginalTaxRate: 0.32,
			now: new Date(2026, 0, 1)
		});
		expect(result.annualTarget).toBe(11304);
		expect(result.monthsLeft).toBe(12);
		expect(result.monthlyTarget).toBeCloseTo(942, 0);
		expect(result.refundEstimate).toBeCloseTo(11304 * 0.32, 2);
		expect(result.annualRefund).toBeCloseTo(11304 * 0.32, 2);
	});

	it('subtracts already contributed amounts from the monthly target', () => {
		const result = optimizeIKZE({
			year: 2026,
			limitKind: 'employee',
			alreadyContributed: 5304,
			marginalTaxRate: 0.12,
			now: new Date(2026, 5, 1)
		});
		expect(result.remaining).toBe(6000);
		expect(result.monthsLeft).toBe(7);
		expect(result.monthlyTarget).toBeCloseTo(6000 / 7, 2);
		// refundEstimate is the *remaining* upside, not the full-year refund.
		expect(result.refundEstimate).toBeCloseTo(6000 * 0.12, 2);
		expect(result.annualRefund).toBeCloseTo(11304 * 0.12, 2);
	});

	it('uses the B2B limit when requested', () => {
		const result = optimizeIKZE({
			year: 2026,
			limitKind: 'b2b',
			alreadyContributed: 0,
			marginalTaxRate: 0.19,
			now: new Date(2026, 0, 1)
		});
		expect(result.annualTarget).toBe(16956);
		expect(result.refundEstimate).toBeCloseTo(16956 * 0.19, 2);
		expect(result.annualRefund).toBeCloseTo(16956 * 0.19, 2);
	});

	it('reports zero refundEstimate once the limit is fully used', () => {
		const result = optimizeIKZE({
			year: 2026,
			limitKind: 'employee',
			alreadyContributed: 11304,
			marginalTaxRate: 0.32,
			now: new Date(2026, 0, 1)
		});
		expect(result.remaining).toBe(0);
		expect(result.refundEstimate).toBe(0);
		expect(result.annualRefund).toBeCloseTo(11304 * 0.32, 2);
	});

	it('clamps already contributed at the limit (remaining never negative)', () => {
		const result = optimizeIKZE({
			year: 2026,
			limitKind: 'employee',
			alreadyContributed: 20000,
			marginalTaxRate: 0.12,
			now: new Date(2026, 5, 1)
		});
		expect(result.remaining).toBe(0);
		expect(result.monthlyTarget).toBe(0);
	});

	it('returns zero monthly target after the year ends', () => {
		const result = optimizeIKZE({
			year: 2025,
			limitKind: 'employee',
			alreadyContributed: 1000,
			marginalTaxRate: 0.12,
			now: new Date(2026, 1, 1),
			limitOverride: 9388
		});
		expect(result.monthsLeft).toBe(0);
		expect(result.monthlyTarget).toBe(0);
	});

	it('honors limit overrides for historical years missing from rules', () => {
		const result = optimizeIKZE({
			year: 2024,
			limitKind: 'employee',
			alreadyContributed: 0,
			marginalTaxRate: 0.32,
			now: new Date(2024, 0, 1),
			limitOverride: 9388.8
		});
		expect(result.annualTarget).toBe(9388.8);
		expect(result.limitSource).toBe('override');
	});
});
