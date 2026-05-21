import { describe, it, expect } from 'vitest';
import {
	constantsFor,
	grossToNet,
	netB2bLiniowy,
	netB2bRyczalt,
	netUod,
	netUop,
	netUz
} from './pl_tax';

const APPROX = 0.01;

describe('constantsFor', () => {
	it('returns known year', () => {
		expect(constantsFor(2024).year).toBe(2024);
	});

	it('falls back for unknown year', () => {
		const c = constantsFor(2030);
		expect([2024, 2025, 2026]).toContain(c.year);
	});
});

describe('netUop', () => {
	it('matches expected breakdown at 10k/mo', () => {
		const r = netUop(10000, 2024);
		const gross = 120000;
		const zus = gross * 0.1371;
		const afterZus = gross - zus;
		const health = afterZus * 0.09;
		const taxable = afterZus - 3000;
		const pit = (taxable - 30000) * 0.12;
		expect(r.grossAnnual).toBeCloseTo(gross, APPROX);
		expect(r.zusAnnual).toBeCloseTo(zus, APPROX);
		expect(r.healthAnnual).toBeCloseTo(health, APPROX);
		expect(r.pitAnnual).toBeCloseTo(pit, APPROX);
		expect(r.netAnnual).toBeCloseTo(gross - zus - health - pit, APPROX);
	});

	it('applies ZUS cap above 30x threshold', () => {
		const r = netUop(50000, 2024);
		const cap = 282600;
		const gross = 600000;
		const expectedZus = cap * (0.0976 + 0.015) + gross * 0.0245;
		expect(r.zusAnnual).toBeCloseTo(expectedZus, APPROX);
	});

	it('triggers 32% bracket above 120k', () => {
		const r = netUop(30000, 2024);
		const gross = 360000;
		expect(r.netAnnual).toBeLessThan(gross);
		const r2 = netUop(8000, 2024);
		const effLow = r2.pitAnnual / r2.grossAnnual;
		const effHigh = r.pitAnnual / r.grossAnnual;
		expect(effHigh).toBeGreaterThan(effLow);
	});

	it('produces no negative PIT for tiny salary', () => {
		const r = netUop(1500, 2024);
		expect(r.pitAnnual).toBeGreaterThanOrEqual(0);
		expect(r.netAnnual).toBeGreaterThan(0);
	});
});

describe('netB2bLiniowy', () => {
	it('matches expected at 15k/mo', () => {
		const r = netB2bLiniowy(15000, 2024);
		const revenue = 180000;
		const health = revenue * 0.049;
		const pitBase = revenue - health;
		expect(r.healthAnnual).toBeCloseTo(health, APPROX);
		expect(r.pitAnnual).toBeCloseTo(pitBase * 0.19, APPROX);
		expect(r.zusAnnual).toBe(0);
	});

	it('caps health deduction', () => {
		const r = netB2bLiniowy(41667, 2024);
		const revenue = 41667 * 12;
		const pitBase = revenue - 11600;
		expect(r.pitAnnual).toBeCloseTo(pitBase * 0.19, APPROX);
	});
});

describe('netB2bRyczalt', () => {
	it('low tier under 60k', () => {
		const r = netB2bRyczalt(4000, 2024);
		expect(r.healthAnnual).toBeCloseTo(419.46 * 12, APPROX);
	});

	it('mid tier 60–300k', () => {
		const r = netB2bRyczalt(15000, 2024);
		expect(r.healthAnnual).toBeCloseTo(699.11 * 12, APPROX);
	});

	it('high tier above 300k', () => {
		const r = netB2bRyczalt(30000, 2024);
		expect(r.healthAnnual).toBeCloseTo(1258.39 * 12, APPROX);
	});

	it('honors custom rate', () => {
		const r = netB2bRyczalt(10000, 2024, 0.085);
		expect(r.pitAnnual).toBeCloseTo(120000 * 0.085, APPROX);
	});
});

describe('netUz and netUod', () => {
	it('UZ has ZUS and health', () => {
		const r = netUz(8000, 2024);
		expect(r.zusAnnual).toBeGreaterThan(0);
		expect(r.healthAnnual).toBeGreaterThan(0);
		expect(r.netAnnual).toBeLessThan(r.grossAnnual);
	});

	it('UoD has no ZUS or health', () => {
		const r = netUod(8000, 2024);
		expect(r.zusAnnual).toBe(0);
		expect(r.healthAnnual).toBe(0);
		expect(r.pitAnnual).toBeGreaterThan(0);
	});
});

describe('grossToNet dispatcher', () => {
	it('dispatches UOP correctly', () => {
		const r = grossToNet(10000, 'UOP', 2024);
		expect(r.zusAnnual).toBeGreaterThan(0);
	});

	it('dispatches B2B as liniowy default', () => {
		const r = grossToNet(10000, 'B2B', 2024);
		expect(r.zusAnnual).toBe(0);
		expect(r.pitAnnual).toBeGreaterThan(0);
	});

	it('dispatches UoD with no ZUS', () => {
		const r = grossToNet(10000, 'UoD', 2024);
		expect(r.zusAnnual).toBe(0);
		expect(r.healthAnnual).toBe(0);
	});
});
