import { describe, it, expect } from 'vitest';
import {
	calculateUoP,
	calculateB2B,
	calculateRSUAfterTax,
	calculateOffer,
	findBreakEvenAmount,
	ContractType,
	B2BTaxForm,
	ZUSTier
} from './compensation';

describe('calculateUoP', () => {
	it('calculates 15k gross correctly', () => {
		const r = calculateUoP(15_000);
		expect(r.zusEmployee).toBeCloseTo(2056.5, 0);
		expect(r.healthInsurance).toBeCloseTo(1164.92, 0);
		expect(r.pit).toBeCloseTo(1761.92, 0);
		expect(r.netMonthly).toBeCloseTo(10016.67, 0);
		expect(r.employerCost).toBeCloseTo(15_000 * 1.2048, 0);
		expect(r.vacationEquivalent).toBe(0);
	});

	it('calculates 8k gross (below 32% threshold)', () => {
		const r = calculateUoP(8_000);
		// taxBase = 96000 - 13161.6 - 3000 = 79838.4 → all at 12%
		expect(r.pit).toBeCloseTo(498.38, 0);
		expect(r.netMonthly).toBeGreaterThan(5500);
		expect(r.netMonthly).toBeLessThan(6000);
	});

	it('calculates with PPK enabled', () => {
		const r = calculateUoP(15_000, true);
		expect(r.ppkEmployee).toBeCloseTo(300, 2);
		expect(r.netMonthly).toBeLessThan(calculateUoP(15_000).netMonthly);
		expect(r.employerCost).toBeCloseTo(15_000 * 1.2048 + 15_000 * 0.015, 0);
	});

	it('handles above 32% PIT threshold (25k gross)', () => {
		const r = calculateUoP(25_000);
		// annual gross 300k, taxBase ~255k → hits 32% bracket
		expect(r.pit).toBeGreaterThan(2000);
		expect(r.effectiveTaxRate).toBeGreaterThan(30);
	});

	it('handles high salary with ZUS cap (40k gross)', () => {
		const r = calculateUoP(40_000);
		// annual 480k > ZUS cap 282,600
		// emerytalne + rentowe capped, chorobowe not capped
		const cappedZus = 0.0976 * 282_600 + 0.015 * 282_600 + 0.0245 * 480_000;
		expect(r.zusEmployee).toBeCloseTo(cappedZus / 12, 0);
	});

	it('handles zero gross', () => {
		const r = calculateUoP(0);
		expect(r.netMonthly).toBe(0);
		expect(r.effectiveTaxRate).toBe(0);
	});
});

describe('calculateB2B', () => {
	it('calculates liniowy + pelny ZUS', () => {
		const r = calculateB2B(25_000, B2BTaxForm.LINIOWY, ZUSTier.PELNY, 500);
		expect(r.zusEmployee).toBe(1927);
		expect(r.accountingCost).toBe(500);
		// income = 25000*12 - 1927*12 - 500*12 = 270876
		// pit = 19% * 270876 / 12
		expect(r.pit).toBeCloseTo((0.19 * 270_876) / 12, 0);
		expect(r.netMonthly).toBeGreaterThan(15_000);
	});

	it('calculates ryczalt', () => {
		const r = calculateB2B(20_000, B2BTaxForm.RYCZALT, ZUSTier.PELNY);
		// annual 240k → health tier 60-300k = 831
		expect(r.healthInsurance).toBe(831);
		// pit = 12% * (240000 - 1927*12) / 12
		expect(r.pit).toBeCloseTo((0.12 * (240_000 - 1927 * 12)) / 12, 0);
	});

	it('calculates skala tax form', () => {
		const r = calculateB2B(15_000, B2BTaxForm.SKALA, ZUSTier.PELNY, 300);
		// Should use progressive brackets like UoP
		expect(r.healthInsurance).toBeGreaterThan(0);
		expect(r.pit).toBeGreaterThan(0);
	});

	it('calculates ulga na start (ZUS=0)', () => {
		const r = calculateB2B(20_000, B2BTaxForm.LINIOWY, ZUSTier.ULGA);
		expect(r.zusEmployee).toBe(0);
		expect(r.netMonthly).toBeGreaterThan(
			calculateB2B(20_000, B2BTaxForm.LINIOWY, ZUSTier.PELNY).netMonthly
		);
	});

	it('calculates preferencyjny ZUS', () => {
		const r = calculateB2B(20_000, B2BTaxForm.LINIOWY, ZUSTier.PREFERENCYJNY);
		expect(r.zusEmployee).toBe(456);
	});

	it('calculates vacation equivalent > 0', () => {
		const r = calculateB2B(20_000, B2BTaxForm.LINIOWY, ZUSTier.PELNY);
		expect(r.vacationEquivalent).toBeGreaterThan(0);
		// ~26 days * dailyNet / 12
		const expectedDaily = r.netMonthly / 21;
		expect(r.vacationEquivalent).toBeCloseTo((26 * expectedDaily) / 12, 2);
	});

	it('handles ryczalt low revenue tier', () => {
		const r = calculateB2B(4_000, B2BTaxForm.RYCZALT, ZUSTier.ULGA);
		// annual 48k ≤ 60k → health = 498
		expect(r.healthInsurance).toBe(498);
	});

	it('handles ryczalt high revenue tier', () => {
		const r = calculateB2B(30_000, B2BTaxForm.RYCZALT, ZUSTier.PELNY);
		// annual 360k > 300k → health = 1495
		expect(r.healthInsurance).toBe(1495);
	});
});

describe('calculateRSUAfterTax', () => {
	it('applies 19% capital gains tax', () => {
		expect(calculateRSUAfterTax(100_000)).toBe(81_000);
	});

	it('handles zero', () => {
		expect(calculateRSUAfterTax(0)).toBe(0);
	});
});

describe('calculateOffer', () => {
	it('dispatches UoP correctly', () => {
		const r = calculateOffer({
			name: 'Test UoP',
			contractType: ContractType.UOP,
			grossMonthly: 15_000
		});
		expect(r.name).toBe('Test UoP');
		expect(r.contractType).toBe(ContractType.UOP);
		expect(r.netMonthly).toBeCloseTo(calculateUoP(15_000).netMonthly, 2);
	});

	it('dispatches B2B monthly correctly', () => {
		const r = calculateOffer({
			name: 'Test B2B',
			contractType: ContractType.B2B_MONTHLY,
			netInvoice: 25_000,
			taxForm: B2BTaxForm.LINIOWY,
			zusTier: ZUSTier.PELNY
		});
		expect(r.contractType).toBe(ContractType.B2B_MONTHLY);
		expect(r.grossMonthly).toBe(25_000);
	});

	it('dispatches B2B hourly with conversion', () => {
		const r = calculateOffer({
			name: 'Hourly',
			contractType: ContractType.B2B_HOURLY,
			hourlyRate: 150,
			hoursPerMonth: 160,
			taxForm: B2BTaxForm.RYCZALT,
			zusTier: ZUSTier.PREFERENCYJNY
		});
		expect(r.contractType).toBe(ContractType.B2B_HOURLY);
		expect(r.grossMonthly).toBe(150 * 160);
	});

	it('includes RSU after tax', () => {
		const r = calculateOffer({
			name: 'With RSU',
			contractType: ContractType.UOP,
			grossMonthly: 20_000,
			rsuAnnual: 50_000
		});
		expect(r.rsuAfterTax).toBe(40_500);
		expect(r.totalAnnual).toBe(r.netAnnual + 40_500);
	});

	it('sets isCurrentJob flag', () => {
		const r = calculateOffer({
			name: 'Current',
			contractType: ContractType.UOP,
			grossMonthly: 10_000,
			isCurrentJob: true
		});
		expect(r.isCurrentJob).toBe(true);
	});
});

describe('findBreakEvenAmount', () => {
	it('finds UoP gross to match target net', () => {
		const target = calculateUoP(15_000).netMonthly;
		const breakEven = findBreakEvenAmount({ name: '', contractType: ContractType.UOP }, target);
		const result = calculateUoP(breakEven);
		expect(result.netMonthly).toBeGreaterThanOrEqual(target);
		expect(breakEven).toBeCloseTo(15_000, -1);
	});

	it('finds B2B invoice to match UoP net', () => {
		const target = calculateUoP(15_000).netMonthly;
		const breakEven = findBreakEvenAmount(
			{
				name: '',
				contractType: ContractType.B2B_MONTHLY,
				taxForm: B2BTaxForm.LINIOWY,
				zusTier: ZUSTier.PELNY
			},
			target
		);
		const result = calculateOffer({
			name: '',
			contractType: ContractType.B2B_MONTHLY,
			netInvoice: breakEven,
			taxForm: B2BTaxForm.LINIOWY,
			zusTier: ZUSTier.PELNY
		});
		expect(result.netMonthly).toBeGreaterThanOrEqual(target);
	});

	it('finds B2B hourly rate to match target net', () => {
		const target = calculateUoP(10_000).netMonthly;
		const breakEven = findBreakEvenAmount(
			{
				name: '',
				contractType: ContractType.B2B_HOURLY,
				hoursPerMonth: 160,
				taxForm: B2BTaxForm.RYCZALT,
				zusTier: ZUSTier.PREFERENCYJNY
			},
			target
		);
		expect(breakEven).toBeGreaterThan(0);
	});
});
