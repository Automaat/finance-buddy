import { describe, it, expect } from 'vitest';
import { round2, daysSince, staleQuotes, type HoldingQuote } from './quoteFreshness';

describe('round2', () => {
	it('rounds long decimals to two places', () => {
		expect(round2(53970.9459578916)).toBe(53970.95);
	});

	it('leaves clean values unchanged', () => {
		expect(round2(100)).toBe(100);
		expect(round2(12.34)).toBe(12.34);
	});

	it('handles zero and negatives', () => {
		expect(round2(0)).toBe(0);
		expect(round2(-1.005)).toBe(-1); // banker's edge: -1.005 → -1
	});
});

describe('daysSince', () => {
	const now = Date.UTC(2026, 5, 14); // 2026-06-14

	it('returns Infinity for a missing date', () => {
		expect(daysSince(null, now)).toBe(Number.POSITIVE_INFINITY);
	});

	it('returns 0 for today', () => {
		expect(daysSince('2026-06-14', now)).toBe(0);
	});

	it('counts whole days elapsed', () => {
		expect(daysSince('2026-06-12', now)).toBe(2);
		expect(daysSince('2026-06-01', now)).toBe(13);
	});
});

describe('staleQuotes', () => {
	const now = Date.UTC(2026, 5, 14);
	const holding = (
		name: string,
		quantity: string,
		latest_quote_date: string | null
	): HoldingQuote => ({ security: { name }, quantity, latest_quote_date });

	it('flags quotes older than the threshold', () => {
		const result = staleQuotes([holding('VWCE', '10', '2026-06-10')], now, 2);
		expect(result).toEqual([{ name: 'VWCE', date: '2026-06-10', daysOld: 4 }]);
	});

	it('flags positions with no quote', () => {
		const result = staleQuotes([holding('ISAC', '5', null)], now, 2);
		expect(result[0]).toMatchObject({ name: 'ISAC', date: null });
		expect(result[0].daysOld).toBe(Number.POSITIVE_INFINITY);
	});

	it('excludes fresh quotes within the threshold', () => {
		expect(staleQuotes([holding('VWCE', '10', '2026-06-13')], now, 2)).toEqual([]);
	});

	it('ignores positions with zero or negative quantity', () => {
		expect(staleQuotes([holding('SOLD', '0', null)], now, 2)).toEqual([]);
	});
});
