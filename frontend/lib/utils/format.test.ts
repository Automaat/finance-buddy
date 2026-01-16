import { describe, it, expect } from 'vitest';
import { formatPLN, formatDate, formatPercent } from './format';

describe('formatPLN', () => {
	it('formats positive numbers', () => {
		expect(formatPLN(1234.56)).toBe('1234,56\u00a0zł');
	});

	it('formats negative numbers', () => {
		expect(formatPLN(-1234.56)).toBe('-1234,56\u00a0zł');
	});

	it('formats zero', () => {
		expect(formatPLN(0)).toBe('0,00\u00a0zł');
	});

	it('formats large numbers', () => {
		expect(formatPLN(1234567.89)).toBe('1\u00a0234\u00a0567,89\u00a0zł');
	});

	it('formats string input', () => {
		expect(formatPLN('1234.56')).toBe('1234,56\u00a0zł');
	});

	it('handles decimal precision', () => {
		expect(formatPLN(100.1)).toBe('100,10\u00a0zł');
		expect(formatPLN(100.123)).toBe('100,12\u00a0zł');
	});
});

describe('formatDate', () => {
	it('formats Date object', () => {
		const date = new Date('2024-03-15T12:00:00Z');
		expect(formatDate(date)).toMatch(/15\.03\.2024/);
	});

	it('formats ISO string', () => {
		expect(formatDate('2024-03-15T12:00:00Z')).toMatch(/15\.03\.2024/);
	});

	it('handles edge dates', () => {
		expect(formatDate('2024-01-01T12:00:00Z')).toMatch(/1\.01\.2024/);
		expect(formatDate('2024-12-31T12:00:00Z')).toMatch(/31\.12\.2024/);
	});

	it('formats date without time', () => {
		expect(formatDate('2024-06-15')).toMatch(/15\.06\.2024/);
	});
});

describe('formatPercent', () => {
	it('formats positive percentages', () => {
		expect(formatPercent(50)).toBe('50,00%');
	});

	it('formats negative percentages', () => {
		expect(formatPercent(-25)).toBe('-25,00%');
	});

	it('formats zero', () => {
		expect(formatPercent(0)).toBe('0,00%');
	});

	it('formats decimal percentages', () => {
		expect(formatPercent(12.34)).toBe('12,34%');
		expect(formatPercent(0.56)).toBe('0,56%');
	});

	it('formats large percentages', () => {
		expect(formatPercent(150)).toBe('150,00%');
	});

	it('handles precision', () => {
		expect(formatPercent(33.333)).toBe('33,33%');
		expect(formatPercent(66.666)).toBe('66,67%');
	});
});
