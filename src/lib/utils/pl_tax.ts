/**
 * Polish gross-to-net salary calculator.
 *
 * Mirrors backend/app/services/pl_tax.py — keep both in sync when constants
 * or formulas change. Constants are stable since the 2022 PIT reform.
 *
 * Models the most common employment contract types:
 * - UOP: ZUS pracownika (capped) + składka zdrowotna 9% + PIT 12/32 + 30k kwota wolna
 * - B2B liniowy 19%: flat PIT, składka zdrowotna 4.9% partially deductible
 * - B2B ryczałt 12% (IT default): flat % on revenue, tiered składka zdrowotna
 * - UZ: ZUS + health + progressive PIT with 20% KUP
 * - UoD: progressive PIT only with 20% KUP
 */

export type PlContractType = 'UOP' | 'B2B' | 'UZ' | 'UoD';

export interface TaxConstants {
	year: number;
	zusEmerytalne: number;
	zusRentowe: number;
	zusChorobowe: number;
	zusCapAnnual: number;
	healthRate: number;
	pitLowRate: number;
	pitHighRate: number;
	pitThresholdAnnual: number;
	freeAmountAnnual: number;
	kupMonthly: number;
	b2bLiniowyRate: number;
	b2bLiniowyHealthRate: number;
	b2bLiniowyHealthDeductionCapAnnual: number;
	ryczaltItRate: number;
	ryczaltHealthLowMonthly: number;
	ryczaltHealthMidMonthly: number;
	ryczaltHealthHighMonthly: number;
	ryczaltLowThresholdAnnual: number;
	ryczaltHighThresholdAnnual: number;
}

const CONSTANTS_2024: TaxConstants = {
	year: 2024,
	zusEmerytalne: 0.0976,
	zusRentowe: 0.015,
	zusChorobowe: 0.0245,
	zusCapAnnual: 282_600,
	healthRate: 0.09,
	pitLowRate: 0.12,
	pitHighRate: 0.32,
	pitThresholdAnnual: 120_000,
	freeAmountAnnual: 30_000,
	kupMonthly: 250,
	b2bLiniowyRate: 0.19,
	b2bLiniowyHealthRate: 0.049,
	b2bLiniowyHealthDeductionCapAnnual: 11_600,
	ryczaltItRate: 0.12,
	ryczaltHealthLowMonthly: 419.46,
	ryczaltHealthMidMonthly: 699.11,
	ryczaltHealthHighMonthly: 1258.39,
	ryczaltLowThresholdAnnual: 60_000,
	ryczaltHighThresholdAnnual: 300_000
};

const BY_YEAR: Record<number, TaxConstants> = {
	2024: CONSTANTS_2024,
	2025: CONSTANTS_2024,
	2026: CONSTANTS_2024
};

export function constantsFor(year: number): TaxConstants {
	if (year in BY_YEAR) return BY_YEAR[year];
	const latest = Math.max(...Object.keys(BY_YEAR).map(Number));
	return BY_YEAR[latest];
}

export interface NetBreakdown {
	grossAnnual: number;
	zusAnnual: number;
	healthAnnual: number;
	pitAnnual: number;
	netAnnual: number;
}

function progressivePit(taxable: number, c: TaxConstants): number {
	if (taxable <= c.freeAmountAnnual) return 0;
	const afterFree = taxable - c.freeAmountAnnual;
	const lowBandCap = c.pitThresholdAnnual - c.freeAmountAnnual;
	if (afterFree <= lowBandCap) return afterFree * c.pitLowRate;
	const highBand = afterFree - lowBandCap;
	return lowBandCap * c.pitLowRate + highBand * c.pitHighRate;
}

function zusEmployeeAnnual(grossAnnual: number, c: TaxConstants): number {
	const capped = Math.min(grossAnnual, c.zusCapAnnual);
	return capped * c.zusEmerytalne + capped * c.zusRentowe + grossAnnual * c.zusChorobowe;
}

export function netUop(grossMonthly: number, year: number): NetBreakdown {
	const c = constantsFor(year);
	const grossAnnual = grossMonthly * 12;
	const zusAnnual = zusEmployeeAnnual(grossAnnual, c);
	const afterZus = grossAnnual - zusAnnual;
	const healthAnnual = afterZus * c.healthRate;
	const kupAnnual = c.kupMonthly * 12;
	const taxable = Math.max(0, afterZus - kupAnnual);
	const pitAnnual = progressivePit(taxable, c);
	const netAnnual = grossAnnual - zusAnnual - healthAnnual - pitAnnual;
	return { grossAnnual, zusAnnual, healthAnnual, pitAnnual, netAnnual };
}

export function netUz(grossMonthly: number, year: number): NetBreakdown {
	const c = constantsFor(year);
	const grossAnnual = grossMonthly * 12;
	const zusAnnual = zusEmployeeAnnual(grossAnnual, c);
	const afterZus = grossAnnual - zusAnnual;
	const healthAnnual = afterZus * c.healthRate;
	const kupAnnual = grossAnnual * 0.2;
	const taxable = Math.max(0, afterZus - kupAnnual);
	const pitAnnual = progressivePit(taxable, c);
	const netAnnual = grossAnnual - zusAnnual - healthAnnual - pitAnnual;
	return { grossAnnual, zusAnnual, healthAnnual, pitAnnual, netAnnual };
}

export function netUod(grossMonthly: number, year: number): NetBreakdown {
	const c = constantsFor(year);
	const grossAnnual = grossMonthly * 12;
	const kupAnnual = grossAnnual * 0.2;
	const taxable = Math.max(0, grossAnnual - kupAnnual);
	const pitAnnual = progressivePit(taxable, c);
	const netAnnual = grossAnnual - pitAnnual;
	return { grossAnnual, zusAnnual: 0, healthAnnual: 0, pitAnnual, netAnnual };
}

export function netB2bLiniowy(revenueMonthly: number, year: number): NetBreakdown {
	const c = constantsFor(year);
	const revenueAnnual = revenueMonthly * 12;
	const healthAnnual = revenueAnnual * c.b2bLiniowyHealthRate;
	const deductible = Math.min(healthAnnual, c.b2bLiniowyHealthDeductionCapAnnual);
	const pitBase = Math.max(0, revenueAnnual - deductible);
	const pitAnnual = pitBase * c.b2bLiniowyRate;
	const netAnnual = revenueAnnual - healthAnnual - pitAnnual;
	return {
		grossAnnual: revenueAnnual,
		zusAnnual: 0,
		healthAnnual,
		pitAnnual,
		netAnnual
	};
}

export function netB2bRyczalt(revenueMonthly: number, year: number, rate?: number): NetBreakdown {
	const c = constantsFor(year);
	const revenueAnnual = revenueMonthly * 12;
	const effectiveRate = rate ?? c.ryczaltItRate;
	let healthMonthly: number;
	if (revenueAnnual <= c.ryczaltLowThresholdAnnual) {
		healthMonthly = c.ryczaltHealthLowMonthly;
	} else if (revenueAnnual <= c.ryczaltHighThresholdAnnual) {
		healthMonthly = c.ryczaltHealthMidMonthly;
	} else {
		healthMonthly = c.ryczaltHealthHighMonthly;
	}
	const healthAnnual = healthMonthly * 12;
	const pitAnnual = revenueAnnual * effectiveRate;
	const netAnnual = revenueAnnual - healthAnnual - pitAnnual;
	return {
		grossAnnual: revenueAnnual,
		zusAnnual: 0,
		healthAnnual,
		pitAnnual,
		netAnnual
	};
}

export function grossToNet(
	grossMonthly: number,
	contractType: PlContractType,
	year: number
): NetBreakdown {
	switch (contractType) {
		case 'UOP':
			return netUop(grossMonthly, year);
		case 'B2B':
			return netB2bLiniowy(grossMonthly, year);
		case 'UZ':
			return netUz(grossMonthly, year);
		case 'UoD':
			return netUod(grossMonthly, year);
	}
}
