import { describe, it, expect } from 'vitest';
import {
	calculateChange,
	formatDate,
	formatNumber,
	formatPLN,
	formatPercent,
	formatSignedPLN,
	formatSignedPercent
} from './format';

describe('formatSignedPLN', () => {
	it('prefixes positive value with +', () => {
		const result = formatSignedPLN(1234);
		expect(result).toMatch(/^\+/);
		expect(result).toContain('1');
	});

	it('prefixes negative value with − (U+2212)', () => {
		const result = formatSignedPLN(-500);
		expect(result.startsWith('−')).toBe(true);
	});

	it('formats zero without sign prefix', () => {
		const result = formatSignedPLN(0);
		expect(result.startsWith('+')).toBe(false);
		expect(result.startsWith('−')).toBe(false);
	});

	it('returns — for null', () => {
		expect(formatSignedPLN(null)).toBe('—');
	});

	it('returns — for undefined', () => {
		expect(formatSignedPLN(undefined)).toBe('—');
	});

	it('returns — for NaN', () => {
		expect(formatSignedPLN(NaN)).toBe('—');
	});
});

describe('formatSignedPercent', () => {
	it('prefixes positive value with +', () => {
		const result = formatSignedPercent(20);
		expect(result).toMatch(/^\+/);
	});

	it('prefixes negative value with − (U+2212)', () => {
		const result = formatSignedPercent(-10);
		expect(result.startsWith('−')).toBe(true);
	});

	it('formats zero without sign prefix', () => {
		const result = formatSignedPercent(0);
		expect(result.startsWith('+')).toBe(false);
		expect(result.startsWith('−')).toBe(false);
	});

	it('returns — for null', () => {
		expect(formatSignedPercent(null)).toBe('—');
	});

	it('returns — for undefined', () => {
		expect(formatSignedPercent(undefined)).toBe('—');
	});

	it('returns — for NaN', () => {
		expect(formatSignedPercent(NaN)).toBe('—');
	});

	it('does not double-round values near a display-rounding boundary', () => {
		// Regression guard: no pre-rounding allowed — same boundary hazard as formatPercent.
		expect(formatSignedPercent(1.04996)).toBe(formatSignedPercent(1.0));
		expect(formatSignedPercent(-1.04996)).toBe(formatSignedPercent(-1.0));
	});
});

describe('formatPLN', () => {
	it('formats a number as PLN currency', () => {
		expect(formatPLN(1234)).toContain('1');
	});

	it('returns — for null, undefined and NaN', () => {
		expect(formatPLN(null)).toBe('—');
		expect(formatPLN(undefined)).toBe('—');
		expect(formatPLN(NaN)).toBe('—');
	});
});

describe('formatPercent', () => {
	it('formats a number as a percentage', () => {
		expect(formatPercent(50)).toMatch(/50/);
	});

	it('suppresses float precision noise from 0.07*100', () => {
		// 0.07 * 100 === 7.000000000000001 in JS
		expect(formatPercent(7.000000000000001)).toBe(formatPercent(7));
	});

	it('does not double-round values near a display-rounding boundary', () => {
		// Regression guard: if pre-rounding to 4 decimals were added, 1.04996 → 1.0500
		// and Intl would round up to 1,1% — wrong. Let Intl round directly to 1,0%.
		expect(formatPercent(1.04996)).toBe(formatPercent(1.0));
	});

	it('returns — for null, undefined and NaN', () => {
		expect(formatPercent(null)).toBe('—');
		expect(formatPercent(undefined)).toBe('—');
		expect(formatPercent(NaN)).toBe('—');
	});
});

describe('formatDate', () => {
	it('formats an ISO date string', () => {
		expect(formatDate('2024-03-15')).toMatch(/2024/);
	});

	it('formats a Date instance', () => {
		expect(formatDate(new Date('2024-03-15'))).toMatch(/2024/);
	});

	it('returns — for empty and invalid input', () => {
		expect(formatDate(null)).toBe('—');
		expect(formatDate(undefined)).toBe('—');
		expect(formatDate('')).toBe('—');
		expect(formatDate('not-a-date')).toBe('—');
	});
});

describe('formatNumber', () => {
	it('uses comma as decimal separator', () => {
		expect(formatNumber(1.5, 1)).toContain(',');
		expect(formatNumber(1.5, 1)).not.toContain('.');
	});

	it('contains the correct digits', () => {
		expect(formatNumber(1234.5, 1)).toMatch(/1.*234.*5/);
	});

	it('rounds to specified decimal places', () => {
		expect(formatNumber(1.234, 2)).toBe('1,23');
	});

	it('returns — for null, undefined and NaN', () => {
		expect(formatNumber(null)).toBe('—');
		expect(formatNumber(undefined)).toBe('—');
		expect(formatNumber(NaN)).toBe('—');
	});

	it('defaults to 2 decimal places', () => {
		expect(formatNumber(1.5)).toBe('1,50');
	});

	it('preserves sign for negative values', () => {
		const result = formatNumber(-5, 2);
		expect(result).toMatch(/[-−]5,00/);
	});
});

describe('calculateChange', () => {
	it('reports an upward change', () => {
		expect(calculateChange(150, 100)).toEqual({
			absolute: 50,
			percent: 50,
			direction: 'up'
		});
	});

	it('reports a downward change', () => {
		const change = calculateChange(80, 100);
		expect(change.absolute).toBe(-20);
		expect(change.direction).toBe('down');
	});

	it('reports a flat change and guards against division by zero', () => {
		expect(calculateChange(0, 0)).toEqual({
			absolute: 0,
			percent: 0,
			direction: 'flat'
		});
	});
});
