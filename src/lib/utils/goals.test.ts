import { describe, it, expect } from 'vitest';
import { goalProgress, goalRemaining, projectGoalHitDate } from './goals';

describe('goalProgress', () => {
	it('calculates percent when current < target', () => {
		expect(goalProgress(1500, 5000)).toBe(30);
	});

	it('caps at 100 when current >= target', () => {
		expect(goalProgress(6000, 5000)).toBe(100);
	});

	it('returns 0 when target is 0', () => {
		expect(goalProgress(100, 0)).toBe(0);
	});

	it('returns 0 when both are 0', () => {
		expect(goalProgress(0, 0)).toBe(0);
	});

	it('handles current = 0', () => {
		expect(goalProgress(0, 1000)).toBe(0);
	});
});

describe('goalRemaining', () => {
	it('returns positive remaining', () => {
		expect(goalRemaining(2000, 5000)).toBe(3000);
	});

	it('returns 0 when target reached', () => {
		expect(goalRemaining(5000, 5000)).toBe(0);
	});

	it('returns 0 when current exceeds target', () => {
		expect(goalRemaining(6000, 5000)).toBe(0);
	});
});

describe('projectGoalHitDate', () => {
	const today = new Date('2026-01-15');

	it('returns today when target already reached', () => {
		const result = projectGoalHitDate(5000, 5000, 100, today);
		expect(result).toEqual(today);
	});

	it('returns null when monthly contribution is 0', () => {
		expect(projectGoalHitDate(1000, 5000, 0, today)).toBe(null);
	});

	it('returns null when monthly contribution is negative', () => {
		expect(projectGoalHitDate(1000, 5000, -100, today)).toBe(null);
	});

	it('projects future date for partial progress', () => {
		// 4000 remaining / 1000 per month = 4 months
		const result = projectGoalHitDate(1000, 5000, 1000, today);
		expect(result).not.toBe(null);
		expect(result!.getMonth()).toBe(4); // May (0-indexed)
		expect(result!.getFullYear()).toBe(2026);
	});

	it('rounds up months for partial last month', () => {
		// 1500 remaining / 1000 per month = 1.5 → 2 months
		const result = projectGoalHitDate(3500, 5000, 1000, today);
		expect(result).not.toBe(null);
		expect(result!.getMonth()).toBe(2); // March
	});
});
