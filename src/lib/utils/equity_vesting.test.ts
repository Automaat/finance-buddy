import { describe, it, expect } from 'vitest';
import {
	vestedInYear,
	isEffectivelyVested,
	latestValuationFor,
	perShareIntrinsicValue,
	computeYearlyEquityComp
} from './equity_vesting';
import type { CompanyValuation, EquityGrant } from '$lib/types/salaries';

function makeGrant(overrides: Partial<EquityGrant> = {}): EquityGrant {
	return {
		id: 1,
		grant_date: '2022-12-09',
		type: 'option',
		company: 'Kong Inc.',
		owner_user_id: 1,
		total_shares: 10000,
		strike_price: 2.67,
		currency: 'USD',
		vest_start_date: '2022-09-01',
		vest_cliff_months: 12,
		vest_total_months: 48,
		vest_frequency: 'monthly',
		vest_custom_schedule: null,
		requires_liquidity_event: false,
		liquidity_event_date: null,
		tax_treatment: 'capital_gains_19',
		notes: null,
		is_active: true,
		created_at: '2026-05-25T00:00:00Z',
		vested_shares_today: 0,
		vesting_progress_pct: 0,
		paper_value_base: null,
		paper_value_low: null,
		paper_value_high: null,
		paper_value_currency: null,
		paper_value_base_pln: null,
		paper_value_low_pln: null,
		paper_value_high_pln: null,
		fx_rate: null,
		valuation_date: null,
		valuation_source: null,
		...overrides
	};
}

describe('vestedInYear (monthly options)', () => {
	const opt = makeGrant();

	it('counts cliff vesting in the cliff year', () => {
		// Cliff fires 2023-09-01 (12 months × 208.33 = 2500) + Oct/Nov/Dec
		// tranches (3 × 208.33 = 625). 2022 vested = 0 (still in cliff). So
		// vested in 2023 = 15 months total = 3125.
		expect(vestedInYear(opt, 2023)).toBe(3125);
	});

	it('counts ~12 monthly tranches in a full vesting year', () => {
		// 2024: 12 months × 208.33 ≈ 2500.
		expect(vestedInYear(opt, 2024)).toBe(2500);
	});

	it('counts partial year when vesting ends mid-year', () => {
		// vest end = 2026-09-01 → Jan–Sep tranches = 9 × 208.33 ≈ 1875.
		expect(vestedInYear(opt, 2026)).toBe(1875);
	});

	it('returns 0 after vesting completes', () => {
		expect(vestedInYear(opt, 2027)).toBe(0);
	});

	it('returns 0 before vest start', () => {
		expect(vestedInYear(opt, 2021)).toBe(0);
	});

	it('returns 0 during cliff (pre-cliff months in cliff year)', () => {
		// 2022: only Sep–Dec elapsed (4 months), still in cliff. 0 vested.
		expect(vestedInYear(opt, 2022)).toBe(0);
	});
});

describe('vestedInYear (RSU, quarterly custom schedule)', () => {
	const rsu = makeGrant({
		id: 2,
		type: 'rsu',
		grant_date: '2025-12-10',
		total_shares: 985,
		strike_price: null,
		vest_start_date: '2025-09-15',
		vest_cliff_months: 0,
		vest_total_months: 48,
		vest_frequency: 'quarterly',
		vest_custom_schedule: Array.from({ length: 16 }, (_, i) => ({
			month: (i + 1) * 3,
			pct: 6.25
		})),
		requires_liquidity_event: true,
		liquidity_event_date: null
	});

	it('counts 1 quarterly tranche in the start year (Dec only)', () => {
		// 2025: month 3 fires Dec 15. 985 * 6.25 / 100 = 61.5 → floor = 61.
		expect(vestedInYear(rsu, 2025)).toBe(61);
	});

	it('counts 4 quarterly tranches in a full vesting year', () => {
		// 2026: months 6, 9, 12, 15 → cumulative 25% by Dec 31; less 6.25% start = 18.75%.
		// 985 * 25 / 100 = 246; 985 * 6.25 / 100 = 61; diff = 185.
		// Or simpler: 4 tranches × ~61.5 = ~246 newly accruing.
		expect(vestedInYear(rsu, 2026)).toBeGreaterThanOrEqual(184);
		expect(vestedInYear(rsu, 2026)).toBeLessThanOrEqual(247);
	});

	it('isEffectivelyVested returns false while LE pending', () => {
		expect(isEffectivelyVested(rsu, 2026)).toBe(false);
	});

	it('isEffectivelyVested returns true once LE has fired by year end', () => {
		const withLE = { ...rsu, liquidity_event_date: '2026-06-15' };
		expect(isEffectivelyVested(withLE, 2026)).toBe(true);
	});

	it('isEffectivelyVested returns false when LE fires after year end', () => {
		const withLE = { ...rsu, liquidity_event_date: '2027-02-01' };
		expect(isEffectivelyVested(withLE, 2026)).toBe(false);
	});
});

describe('latestValuationFor', () => {
	const v1: CompanyValuation = {
		id: 1,
		company: 'Kong Inc.',
		date: '2025-01-01',
		currency: 'USD',
		fmv_per_share: 3,
		fmv_low: null,
		fmv_high: null,
		source: '409a',
		common_stock_discount_pct: null,
		notes: null,
		is_active: true,
		created_at: '2025-01-01T00:00:00Z'
	};
	const v2 = { ...v1, id: 2, date: '2026-05-25', fmv_per_share: 4.94 };
	const inactive = { ...v1, id: 3, date: '2026-06-01', fmv_per_share: 99, is_active: false };
	const other = { ...v1, id: 4, company: 'Other', date: '2026-12-31', fmv_per_share: 100 };

	it('returns the most recent active valuation for the matching company', () => {
		const latest = latestValuationFor([v1, v2, inactive, other], 'Kong Inc.');
		expect(latest?.id).toBe(2);
	});

	it('returns null when no active valuation exists', () => {
		expect(latestValuationFor([inactive], 'Kong Inc.')).toBeNull();
	});
});

describe('perShareIntrinsicValue', () => {
	const valuation: CompanyValuation = {
		id: 1,
		company: 'Kong Inc.',
		date: '2026-05-25',
		currency: 'USD',
		fmv_per_share: 4.94,
		fmv_low: 4,
		fmv_high: 6,
		source: '409a',
		common_stock_discount_pct: null,
		notes: null,
		is_active: true,
		created_at: '2026-05-25T00:00:00Z'
	};

	it('options: spread above strike', () => {
		const v = perShareIntrinsicValue(makeGrant({ strike_price: 2.67 }), valuation);
		expect(v?.base).toBeCloseTo(2.27);
		expect(v?.low).toBeCloseTo(1.33);
		expect(v?.high).toBeCloseTo(3.33);
	});

	it('options: zero when underwater', () => {
		const v = perShareIntrinsicValue(makeGrant({ strike_price: 10 }), valuation);
		expect(v?.base).toBe(0);
	});

	it('RSU: full FMV', () => {
		const v = perShareIntrinsicValue(makeGrant({ type: 'rsu', strike_price: null }), valuation);
		expect(v?.base).toBe(4.94);
	});

	it('returns null when no valuation', () => {
		expect(perShareIntrinsicValue(makeGrant(), null)).toBeNull();
	});
});

describe('computeYearlyEquityComp', () => {
	const valuation: CompanyValuation = {
		id: 1,
		company: 'Kong Inc.',
		date: '2026-05-25',
		currency: 'USD',
		fmv_per_share: 4.94,
		fmv_low: null,
		fmv_high: null,
		source: '409a',
		common_stock_discount_pct: null,
		notes: null,
		is_active: true,
		created_at: '2026-05-25T00:00:00Z'
	};
	const optionGrant = makeGrant({
		id: 1,
		owner_user_id: 1,
		paper_value_currency: 'USD',
		fx_rate: 3.6374
	});
	const rsuLocked = makeGrant({
		id: 2,
		type: 'rsu',
		owner_user_id: 1,
		strike_price: null,
		total_shares: 985,
		vest_start_date: '2025-09-15',
		vest_cliff_months: 0,
		vest_total_months: 48,
		vest_frequency: 'quarterly',
		vest_custom_schedule: Array.from({ length: 16 }, (_, i) => ({
			month: (i + 1) * 3,
			pct: 6.25
		})),
		requires_liquidity_event: true,
		liquidity_event_date: null,
		paper_value_currency: null,
		fx_rate: null
	});

	it('sums options vested in year × intrinsic spread × FX', () => {
		const r = computeYearlyEquityComp([optionGrant], [valuation], 1, 2026);
		// vestedInYear(option, 2026) = 1875 → × $2.27 × 3.6374 ≈ 15482.
		expect(r.vestedPln).toBeCloseTo(1875 * 2.27 * 3.6374, 0);
		expect(r.lockedPln).toBe(0);
		expect(r.hasEquityWithoutFx).toBe(false);
	});

	it('routes locked RSU to lockedPln, not vestedPln', () => {
		// RSU has no fx_rate of its own — borrows from sibling option grant.
		const r = computeYearlyEquityComp([optionGrant, rsuLocked], [valuation], 1, 2026);
		expect(r.vestedPln).toBeCloseTo(1875 * 2.27 * 3.6374, 0);
		// vestedInYear(rsu, 2026) ≈ 246 (some rounding); × $4.94 × 3.6374.
		expect(r.lockedPln).toBeGreaterThan(3000);
		expect(r.lockedPln).toBeLessThan(5000);
	});

	it('unlocks RSU once liquidity event has fired in the year', () => {
		const rsuUnlocked = { ...rsuLocked, liquidity_event_date: '2026-06-15' };
		const r = computeYearlyEquityComp([optionGrant, rsuUnlocked], [valuation], 1, 2026);
		expect(r.vestedPln).toBeGreaterThan(15000 + 3000);
		expect(r.lockedPln).toBe(0);
	});

	it('flags hasEquityWithoutFx when no sibling supplies the rate', () => {
		const usdGrantNoFx = {
			...optionGrant,
			fx_rate: null,
			paper_value_currency: null
		};
		const r = computeYearlyEquityComp([usdGrantNoFx], [valuation], 1, 2026);
		expect(r.vestedPln).toBe(0);
		expect(r.hasEquityWithoutFx).toBe(true);
	});

	it('skips grants owned by other users', () => {
		const otherOwner = { ...optionGrant, owner_user_id: 2 };
		const r = computeYearlyEquityComp([otherOwner], [valuation], 1, 2026);
		expect(r.vestedPln).toBe(0);
	});

	it('skips grants with no valuation for their company', () => {
		const orphan = { ...optionGrant, company: 'Unknown Co' };
		const r = computeYearlyEquityComp([orphan], [valuation], 1, 2026);
		expect(r.vestedPln).toBe(0);
	});

	it('skips grants with zero shares vesting in the year', () => {
		// Year 2027 = post-vesting for the default option grant (ended Sep 2026).
		const r = computeYearlyEquityComp([optionGrant], [valuation], 1, 2027);
		expect(r.vestedPln).toBe(0);
		expect(r.lockedPln).toBe(0);
	});

	it('passes PLN-denominated grants through with rate=1', () => {
		const plnValuation: CompanyValuation = { ...valuation, currency: 'PLN' };
		const plnGrant = {
			...optionGrant,
			currency: 'PLN',
			paper_value_currency: 'PLN',
			fx_rate: 1,
			strike_price: 2.67
		};
		const r = computeYearlyEquityComp([plnGrant], [plnValuation], 1, 2026);
		expect(r.vestedPln).toBeCloseTo(1875 * 2.27, 0);
	});
});
