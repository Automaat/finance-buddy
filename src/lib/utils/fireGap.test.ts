import { describe, it, expect } from 'vitest';
import { projectFireGap, type FireGapInputs } from './fireGap';

function inputs(overrides: Partial<FireGapInputs> = {}): FireGapInputs {
	return {
		currentAge: 35,
		retirementAge: 65,
		lifeExpectancy: 85,
		currentPortfolioPLN: 100000,
		annualContributionPLN: 20000,
		expectedReturnPct: 6,
		inflationPct: 3,
		withdrawalRatePct: 4,
		monthlyPensionNetPLN: 3500,
		...overrides
	};
}

describe('projectFireGap', () => {
	it('returns one entry per year from currentAge to lifeExpectancy inclusive', () => {
		const rows = projectFireGap(inputs());
		expect(rows.length).toBe(85 - 35 + 1);
		expect(rows[0].age).toBe(35);
		expect(rows[rows.length - 1].age).toBe(85);
	});

	it('marks years before retirement as not after-retirement', () => {
		const rows = projectFireGap(inputs());
		expect(rows.find((r) => r.age === 60)?.afterRetirement).toBe(false);
		expect(rows.find((r) => r.age === 65)?.afterRetirement).toBe(true);
		expect(rows.find((r) => r.age === 70)?.afterRetirement).toBe(true);
	});

	it('reports zero ZUS and zero private income before retirement', () => {
		const rows = projectFireGap(inputs());
		const preRetire = rows.filter((r) => !r.afterRetirement);
		for (const row of preRetire) {
			expect(row.zusMonthlyIncomePLN).toBe(0);
			expect(row.privateMonthlyIncomePLN).toBe(0);
			expect(row.gapPLN).toBe(0);
		}
	});

	it('inflation-adjusts the ZUS pension after retirement', () => {
		const rows = projectFireGap(inputs({ inflationPct: 3 }));
		const atRetire = rows.find((r) => r.age === 65)!;
		const later = rows.find((r) => r.age === 75)!;
		// 10 years of 3% inflation × 1.34
		expect(later.zusMonthlyIncomePLN / atRetire.zusMonthlyIncomePLN).toBeCloseTo(
			Math.pow(1.03, 10),
			3
		);
	});

	it('computes positive gap when private withdrawal exceeds ZUS pension', () => {
		const rows = projectFireGap(
			inputs({ currentPortfolioPLN: 3_000_000, monthlyPensionNetPLN: 1000 })
		);
		const atRetire = rows.find((r) => r.age === 65)!;
		expect(atRetire.gapPLN).toBeGreaterThan(0);
		expect(atRetire.privateMonthlyIncomePLN).toBeGreaterThan(atRetire.zusMonthlyIncomePLN);
	});

	it('contributes during working years and stops at retirement', () => {
		const rows = projectFireGap(
			inputs({
				annualContributionPLN: 50000,
				expectedReturnPct: 0,
				inflationPct: 0,
				retirementAge: 40,
				lifeExpectancy: 42,
				currentAge: 38
			})
		);
		const atRetire = rows.find((r) => r.age === 40)!;
		// 2 contribution years of 50k + starting 100k = 200k
		expect(atRetire.portfolioPLN).toBeCloseTo(200000 - 200000 * 0.04, 0);
	});

	it('skips current→retirement inflation when basis is retirement-year PLN', () => {
		const rows = projectFireGap(inputs({ inflationPct: 3, pensionBasis: 'retirement' }));
		const atRetire = rows.find((r) => r.age === 65)!;
		// Pension was already in retirement-year PLN — first retirement
		// row keeps the input verbatim.
		expect(atRetire.zusMonthlyIncomePLN).toBeCloseTo(3500, 1);
		const later = rows.find((r) => r.age === 75)!;
		expect(later.zusMonthlyIncomePLN / atRetire.zusMonthlyIncomePLN).toBeCloseTo(
			Math.pow(1.03, 10),
			3
		);
	});

	it('uses the calendar-year mapper to label rows', () => {
		const rows = projectFireGap(inputs(), (age) => 2026 + (age - 35));
		expect(rows[0].year).toBe(2026);
		expect(rows[1].year).toBe(2027);
	});
});
