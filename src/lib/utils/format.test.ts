import { describe, it, expect } from 'vitest';
import { formatSignedPLN, formatSignedPercent } from './format';

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
});
