import { describe, it, expect } from 'vitest';
import {
	formatPctSigned,
	formatPlnSigned,
	groupSalariesByCompany,
	isNonNegative,
	isoDateLocal
} from './salaries';

describe('isNonNegative', () => {
	it('treats null as zero', () => {
		expect(isNonNegative(null)).toBe(true);
	});
	it('matches zero', () => {
		expect(isNonNegative(0)).toBe(true);
	});
	it('matches positive', () => {
		expect(isNonNegative(0.01)).toBe(true);
	});
	it('rejects negative', () => {
		expect(isNonNegative(-0.01)).toBe(false);
	});
});

describe('formatPctSigned', () => {
	it('renders em-dash for null', () => {
		expect(formatPctSigned(null)).toBe('—');
	});
	it('renders em-dash for NaN', () => {
		expect(formatPctSigned(Number.NaN)).toBe('—');
	});
	it('prefixes positive with +', () => {
		expect(formatPctSigned(5)).toBe('+5.0%');
	});
	it('prefixes zero with +', () => {
		expect(formatPctSigned(0)).toBe('+0.0%');
	});
	it('keeps the native minus for negative — no explicit sign override', () => {
		expect(formatPctSigned(-3.25)).toBe('-3.3%');
	});
	it('rounds to one decimal', () => {
		expect(formatPctSigned(12.34)).toBe('+12.3%');
	});
});

describe('formatPlnSigned', () => {
	it('renders em-dash for null', () => {
		expect(formatPlnSigned(null)).toBe('—');
	});
	it('renders em-dash for NaN', () => {
		expect(formatPlnSigned(Number.NaN)).toBe('—');
	});
	it('prefixes positive with +', () => {
		expect(formatPlnSigned(1500).startsWith('+')).toBe(true);
	});
	it('prefixes zero with + (matches legacy delta-cell look)', () => {
		expect(formatPlnSigned(0).startsWith('+')).toBe(true);
	});
	it('prefixes negative with U+2212 and drops the inner minus', () => {
		const out = formatPlnSigned(-1500);
		expect(out.startsWith('−')).toBe(true);
		expect(out).not.toContain('-');
	});
});

describe('isoDateLocal', () => {
	it('formats a normal date as YYYY-MM-DD using local components', () => {
		// Constructed via local-time fields so the assertion holds regardless
		// of the test runner's timezone — including TZ > UTC where
		// toISOString() would shift the day.
		expect(isoDateLocal(new Date(2026, 11, 31, 0, 0, 0))).toBe('2026-12-31');
	});
	it('pads single-digit month and day', () => {
		expect(isoDateLocal(new Date(2025, 0, 5, 0, 0, 0))).toBe('2025-01-05');
	});
	it('survives end-of-month rollover', () => {
		expect(isoDateLocal(new Date(2025, 1, 28, 23, 59, 0))).toBe('2025-02-28');
	});
});

describe('groupSalariesByCompany', () => {
	const today = '2026-05-31';

	it('extends only the current company to today', () => {
		const map = groupSalariesByCompany(
			[
				{ company: 'Old Co', date: '2022-01-01', amount: 8000 },
				{ company: 'Old Co', date: '2023-01-01', amount: 9000 },
				{ company: 'Current Co', date: '2024-06-01', amount: 20000 },
				{ company: 'Current Co', date: '2025-06-01', amount: 25000 }
			],
			today
		);
		expect(map.get('Old Co')).toEqual([
			['2022-01-01', 8000],
			['2023-01-01', 9000]
		]);
		expect(map.get('Current Co')).toEqual([
			['2024-06-01', 20000],
			['2025-06-01', 25000],
			[today, 25000]
		]);
	});

	it('sorts each company ascending by date before carry-forward', () => {
		const map = groupSalariesByCompany(
			[
				{ company: 'Acme', date: '2025-06-01', amount: 12000 },
				{ company: 'Acme', date: '2024-06-01', amount: 10000 }
			],
			today
		);
		expect(map.get('Acme')).toEqual([
			['2024-06-01', 10000],
			['2025-06-01', 12000],
			[today, 12000]
		]);
	});

	it('labels blank company names', () => {
		const map = groupSalariesByCompany(
			[{ company: '  ', date: '2025-01-01', amount: 5000 }],
			today
		);
		expect([...map.keys()]).toEqual(['Nieokreślona firma']);
	});

	it('does not duplicate the today point when last record is already today', () => {
		const map = groupSalariesByCompany([{ company: 'Acme', date: today, amount: 5000 }], today);
		expect(map.get('Acme')).toEqual([[today, 5000]]);
	});
});
