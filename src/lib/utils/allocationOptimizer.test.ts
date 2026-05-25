import { describe, it, expect } from 'vitest';
import { rankOptions, type OptionInputs, type OptionKey } from './allocationOptimizer';

function baseInputs(overrides: Partial<OptionInputs> = {}): OptionInputs {
	const drift: Record<OptionKey, number> = {
		ikze: 0,
		ike: 0,
		ppk: 0,
		mortgage: 0,
		bonds: 0,
		brokerage: 0
	};
	return {
		amountPLN: 1000,
		marginalPitRate: 0.32,
		ikzeRemainingPLN: 5000,
		ikeRemainingPLN: 10000,
		ppkEmployerMatchRate: 0.015,
		ppkMatched: true,
		mortgageAPRPct: 7,
		mortgageRemainingPLN: 200000,
		bondsYieldPct: 6,
		brokerageReturnPct: 7,
		allocationDrift: drift,
		liquidityNeedScore: 0,
		...overrides
	};
}

describe('rankOptions', () => {
	it('puts IKZE at top when marginal rate is high and liquidity not needed', () => {
		const ranked = rankOptions(baseInputs());
		expect(ranked[0].option).toBe('ikze');
	});

	it('drops IKZE to the bottom of available list when limit exhausted', () => {
		const ranked = rankOptions(baseInputs({ ikzeRemainingPLN: 0 }));
		const ikze = ranked.find((r) => r.option === 'ikze');
		expect(ikze?.available).toBe(false);
		expect(ikze?.availabilityReason).toMatch(/Limit IKZE/);
		expect(ranked[ranked.length - 1].option).toBe('ikze');
	});

	it('marks PPK unavailable when no active match', () => {
		const ranked = rankOptions(baseInputs({ ppkMatched: false }));
		const ppk = ranked.find((r) => r.option === 'ppk');
		expect(ppk?.available).toBe(false);
	});

	it('marks mortgage unavailable when no remaining principal', () => {
		const ranked = rankOptions(baseInputs({ mortgageRemainingPLN: 0 }));
		const m = ranked.find((r) => r.option === 'mortgage');
		expect(m?.available).toBe(false);
	});

	it('penalizes illiquid options when liquidity need is high', () => {
		const liquid = rankOptions(baseInputs({ liquidityNeedScore: 0 }));
		const illiquid = rankOptions(baseInputs({ liquidityNeedScore: 5 }));
		const ikzeLiquid = liquid.find((r) => r.option === 'ikze')!;
		const ikzeIlliquid = illiquid.find((r) => r.option === 'ikze')!;
		expect(ikzeIlliquid.total).toBeLessThan(ikzeLiquid.total);
		const brokerageLiquid = liquid.find((r) => r.option === 'brokerage')!;
		const brokerageIlliquid = illiquid.find((r) => r.option === 'brokerage')!;
		// Brokerage is fully liquid → no penalty.
		expect(brokerageIlliquid.total).toBe(brokerageLiquid.total);
	});

	it('boosts the option whose category is most under target (positive drift)', () => {
		const skew: Record<OptionKey, number> = {
			ikze: 0,
			ike: 0,
			ppk: 0,
			mortgage: 0,
			bonds: 10,
			brokerage: 0
		};
		// Strip the IKZE/PPK incentives so bonds wins on drift alone.
		const ranked = rankOptions(
			baseInputs({
				marginalPitRate: 0,
				ppkMatched: false,
				mortgageRemainingPLN: 0,
				allocationDrift: skew
			})
		);
		expect(ranked[0].option).toBe('bonds');
	});

	it('reports score factors for every option', () => {
		const ranked = rankOptions(baseInputs());
		for (const option of ranked) {
			expect(option.factors.length).toBeGreaterThan(0);
			expect(option.name).toBeTruthy();
		}
	});

	it('scales IKZE benefit when the planned amount exceeds the remaining limit', () => {
		// Identical inputs; only amount changes. With remaining = 1000 and
		// amount = 1000 → full coverage; amount = 5000 → 20% coverage.
		const full = rankOptions(
			baseInputs({ ikzeRemainingPLN: 1000, amountPLN: 1000, liquidityNeedScore: 0 })
		).find((r) => r.option === 'ikze')!;
		const partial = rankOptions(
			baseInputs({ ikzeRemainingPLN: 1000, amountPLN: 5000, liquidityNeedScore: 0 })
		).find((r) => r.option === 'ikze')!;
		// liquidity + drift factors don't depend on coverage, so the
		// difference is exactly the cap-bound factors × (1 − coverage).
		expect(partial.total).toBeLessThan(full.total);
		expect(partial.factors[0].label).toMatch(/część 20%/);
	});

	it('renames the mortgage factor without the gwarantowanie typo', () => {
		const ranked = rankOptions(baseInputs({ mortgageRemainingPLN: 100000, amountPLN: 1000 }));
		const m = ranked.find((r) => r.option === 'mortgage')!;
		expect(m.factors.some((f) => /gwarantowane/.test(f.label))).toBe(true);
		expect(m.factors.some((f) => /gwarantowanie/.test(f.label))).toBe(false);
	});

	it('applies capital gains tax to brokerage expected return', () => {
		const ranked = rankOptions(
			baseInputs({
				marginalPitRate: 0,
				ppkMatched: false,
				mortgageRemainingPLN: 0,
				bondsYieldPct: 0,
				brokerageReturnPct: 10,
				liquidityNeedScore: 0
			})
		);
		const brokerage = ranked.find((r) => r.option === 'brokerage');
		// 10% × (1 − 0.19) = 8.1pp from yield; allocation drift adds 0; liquidity adds 0.
		expect(brokerage!.total).toBeCloseTo(8.1, 1);
	});
});
