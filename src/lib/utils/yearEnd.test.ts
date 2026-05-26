import { describe, it, expect } from 'vitest';
import { countdownTier, daysLabel, daysUntilYearEnd } from './yearEnd';

describe('daysUntilYearEnd', () => {
	it('counts 365 days on Jan 1 of a non-leap year', () => {
		expect(daysUntilYearEnd(new Date(2026, 0, 1, 0, 0, 0))).toBe(365);
	});

	it('counts 366 days on Jan 1 of a leap year', () => {
		expect(daysUntilYearEnd(new Date(2028, 0, 1, 0, 0, 0))).toBe(366);
	});

	it('counts 2 days on Dec 30 (today + Dec 31)', () => {
		expect(daysUntilYearEnd(new Date(2026, 11, 30, 12, 0, 0))).toBe(2);
	});

	it('counts 1 day on Dec 31 (today is the last chance)', () => {
		expect(daysUntilYearEnd(new Date(2026, 11, 31, 23, 59, 59, 999))).toBe(1);
	});

	it('counts ~half year on Jul 1', () => {
		expect(daysUntilYearEnd(new Date(2026, 6, 1, 0, 0, 0))).toBe(184);
	});
});

describe('countdownTier', () => {
	it('returns maxed when limit reached regardless of month', () => {
		expect(countdownTier(new Date(2026, 0, 15), 100)).toBe('maxed');
		expect(countdownTier(new Date(2026, 11, 15), 150)).toBe('maxed');
	});

	it('returns safe in Q1-Q2 (Jan-Jun)', () => {
		expect(countdownTier(new Date(2026, 0, 1), 0)).toBe('safe');
		expect(countdownTier(new Date(2026, 5, 30), 99)).toBe('safe');
	});

	it('returns warn in Q3 (Jul-Sep)', () => {
		expect(countdownTier(new Date(2026, 6, 1), 0)).toBe('warn');
		expect(countdownTier(new Date(2026, 8, 30), 50)).toBe('warn');
	});

	it('returns urgent in Q4 (Oct-Dec) when not maxed', () => {
		expect(countdownTier(new Date(2026, 9, 1), 50)).toBe('urgent');
		expect(countdownTier(new Date(2026, 11, 31), 99.9)).toBe('urgent');
	});
});

describe('daysLabel', () => {
	it('uses singular for 1', () => {
		expect(daysLabel(1)).toBe('dzień');
	});

	it('uses plural for 0', () => {
		expect(daysLabel(0)).toBe('dni');
	});

	it('uses plural for 2-4', () => {
		expect(daysLabel(2)).toBe('dni');
		expect(daysLabel(4)).toBe('dni');
	});

	it('uses plural for larger numbers', () => {
		expect(daysLabel(5)).toBe('dni');
		expect(daysLabel(100)).toBe('dni');
		expect(daysLabel(365)).toBe('dni');
	});
});
