import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
	calculateNetWorth,
	calculateChange,
	calculateGoalProgress,
	calculateMonthsRemaining
} from './calculations';

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

describe('calculateChange', () => {
	it('calculates positive change (increase)', () => {
		const result = calculateChange(150, 100);
		expect(result.value).toBe(50);
		expect(result.percent).toBe(50);
	});

	it('calculates negative change (decrease)', () => {
		const result = calculateChange(75, 100);
		expect(result.value).toBe(-25);
		expect(result.percent).toBe(-25);
	});

	it('handles zero previous value', () => {
		const result = calculateChange(100, 0);
		expect(result.value).toBe(100);
		expect(result.percent).toBe(0);
	});

	it('handles same values (no change)', () => {
		const result = calculateChange(100, 100);
		expect(result.value).toBe(0);
		expect(result.percent).toBe(0);
	});

	it('handles negative to positive transition', () => {
		const result = calculateChange(50, -50);
		expect(result.value).toBe(100);
		expect(result.percent).toBe(-200);
	});

	it('calculates decimal percentages correctly', () => {
		const result = calculateChange(110, 100);
		expect(result.value).toBe(10);
		expect(result.percent).toBe(10);
	});
});

describe('calculateGoalProgress', () => {
	it('calculates partial progress', () => {
		expect(calculateGoalProgress(5000, 10000)).toBe(50);
	});

	it('calculates complete progress', () => {
		expect(calculateGoalProgress(10000, 10000)).toBe(100);
	});

	it('calculates over-target progress', () => {
		expect(calculateGoalProgress(15000, 10000)).toBe(150);
	});

	it('handles zero target', () => {
		expect(calculateGoalProgress(5000, 0)).toBe(0);
	});

	it('handles zero current', () => {
		expect(calculateGoalProgress(0, 10000)).toBe(0);
	});

	it('handles both zero', () => {
		expect(calculateGoalProgress(0, 0)).toBe(0);
	});

	it('calculates decimal progress', () => {
		expect(calculateGoalProgress(3333, 10000)).toBe(33.33);
	});
});

describe('calculateMonthsRemaining', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('calculates future months (same day)', () => {
		vi.setSystemTime(new Date('2024-01-15'));
		const target = new Date('2024-06-15');
		expect(calculateMonthsRemaining(target)).toBe(5);
	});

	it('handles day-level precision (target day passed)', () => {
		vi.setSystemTime(new Date('2024-01-31'));
		const target = new Date('2024-02-01');
		expect(calculateMonthsRemaining(target)).toBe(0);
	});

	it('handles past dates', () => {
		vi.setSystemTime(new Date('2024-06-15'));
		const target = new Date('2024-01-15');
		expect(calculateMonthsRemaining(target)).toBe(0);
	});

	it('handles same month (future day)', () => {
		vi.setSystemTime(new Date('2024-01-15'));
		const target = new Date('2024-01-20');
		expect(calculateMonthsRemaining(target)).toBe(0);
	});

	it('handles same month (past day)', () => {
		vi.setSystemTime(new Date('2024-01-20'));
		const target = new Date('2024-01-15');
		expect(calculateMonthsRemaining(target)).toBe(0);
	});

	it('calculates year boundaries', () => {
		vi.setSystemTime(new Date('2023-12-15'));
		const target = new Date('2024-01-15');
		expect(calculateMonthsRemaining(target)).toBe(1);
	});

	it('handles multiple years ahead', () => {
		vi.setSystemTime(new Date('2024-01-15'));
		const target = new Date('2026-01-15');
		expect(calculateMonthsRemaining(target)).toBe(24);
	});
});
