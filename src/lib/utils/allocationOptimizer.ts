import { PL_RULES } from './pl_rules.generated';

export type OptionKey = 'ikze' | 'ike' | 'ppk' | 'mortgage' | 'bonds' | 'brokerage';

export interface OptionInputs {
	// Amount the user plans to contribute. Used to scale benefit factors
	// for options whose tax/APR upside applies only up to their remaining
	// limit — e.g. paying 5000 PLN into IKZE with 1000 PLN of remaining
	// limit converts only 20% of the deposit into a deductible amount.
	amountPLN: number;
	marginalPitRate: number;
	ikzeRemainingPLN: number;
	ikeRemainingPLN: number;
	ppkEmployerMatchRate: number;
	ppkMatched: boolean;
	mortgageAPRPct: number;
	mortgageRemainingPLN: number;
	bondsYieldPct: number;
	brokerageReturnPct: number;
	allocationDrift: Record<OptionKey, number>;
	liquidityNeedScore: number;
}

export interface ScoreFactor {
	label: string;
	pp: number;
}

export interface OptionScore {
	option: OptionKey;
	name: string;
	available: boolean;
	availabilityReason?: string;
	factors: ScoreFactor[];
	total: number;
}

export const OPTION_NAMES: Record<OptionKey, string> = {
	ikze: 'IKZE',
	ike: 'IKE',
	ppk: 'PPK',
	mortgage: 'Nadpłata hipoteki',
	bonds: 'Obligacje skarbowe',
	brokerage: 'Konto maklerskie'
};

const capitalGainsTax = PL_RULES['capital_gains_tax_2026']?.value ?? 0.19;

const LIQUIDITY_BASE: Record<OptionKey, number> = {
	brokerage: 0,
	bonds: -1,
	ike: -3,
	ikze: -5,
	ppk: -5,
	mortgage: -4
};

function liquidityFactor(option: OptionKey, liquidityNeed: number): ScoreFactor {
	const base = LIQUIDITY_BASE[option];
	const value = base * Math.max(0, Math.min(5, liquidityNeed));
	return { label: 'Płynność', pp: value };
}

function driftFactor(option: OptionKey, drift: Record<OptionKey, number>): ScoreFactor {
	return { label: 'Dryft alokacji', pp: drift[option] ?? 0 };
}

function pp(rate: number): number {
	return rate * 100;
}

// limitCoverage is the share of the user's planned deposit that fits
// inside a per-option contribution cap. Anything above the cap delivers
// none of the tax-side benefit and is silently re-routed to a taxable
// account, so the scorer multiplies cap-bound factors by this ratio
// instead of overstating the value.
function limitCoverage(amount: number, remaining: number): number {
	if (amount <= 0) return 0;
	return Math.max(0, Math.min(1, remaining / amount));
}

function partialLabel(coverage: number, base: string): string {
	if (coverage >= 1) return base;
	return `${base} (część ${Math.round(coverage * 100)}%)`;
}

function scoreIKZE(input: OptionInputs): OptionScore {
	const coverage = limitCoverage(input.amountPLN, input.ikzeRemainingPLN);
	const factors: ScoreFactor[] = [
		{
			label: partialLabel(coverage, 'Ulga PIT (zwrot)'),
			pp: pp(input.marginalPitRate) * coverage
		},
		{
			label: partialLabel(coverage, 'Belka nie pobierana'),
			pp: pp(capitalGainsTax * 0.3) * coverage
		},
		driftFactor('ikze', input.allocationDrift),
		liquidityFactor('ikze', input.liquidityNeedScore)
	];
	return {
		option: 'ikze',
		name: OPTION_NAMES.ikze,
		available: input.ikzeRemainingPLN > 0,
		availabilityReason: input.ikzeRemainingPLN > 0 ? undefined : 'Limit IKZE wyczerpany',
		factors,
		total: factors.reduce((sum, f) => sum + f.pp, 0)
	};
}

function scoreIKE(input: OptionInputs): OptionScore {
	const coverage = limitCoverage(input.amountPLN, input.ikeRemainingPLN);
	const factors: ScoreFactor[] = [
		{
			label: partialLabel(coverage, 'Belka uniknięta'),
			pp: pp(capitalGainsTax * 0.5) * coverage
		},
		driftFactor('ike', input.allocationDrift),
		liquidityFactor('ike', input.liquidityNeedScore)
	];
	return {
		option: 'ike',
		name: OPTION_NAMES.ike,
		available: input.ikeRemainingPLN > 0,
		availabilityReason: input.ikeRemainingPLN > 0 ? undefined : 'Limit IKE wyczerpany',
		factors,
		total: factors.reduce((sum, f) => sum + f.pp, 0)
	};
}

function scorePPK(input: OptionInputs): OptionScore {
	const factors: ScoreFactor[] = [
		{ label: 'Dopłata pracodawcy', pp: pp(input.ppkEmployerMatchRate) },
		driftFactor('ppk', input.allocationDrift),
		liquidityFactor('ppk', input.liquidityNeedScore)
	];
	return {
		option: 'ppk',
		name: OPTION_NAMES.ppk,
		available: input.ppkMatched,
		availabilityReason: input.ppkMatched
			? undefined
			: 'Brak aktywnego PPK lub limit pracodawcy wykorzystany',
		factors,
		total: factors.reduce((sum, f) => sum + f.pp, 0)
	};
}

function scoreMortgage(input: OptionInputs): OptionScore {
	const coverage = limitCoverage(input.amountPLN, input.mortgageRemainingPLN);
	const factors: ScoreFactor[] = [
		{
			label: partialLabel(coverage, 'Odsetki gwarantowane oszczędzone'),
			pp: input.mortgageAPRPct * coverage
		},
		driftFactor('mortgage', input.allocationDrift),
		liquidityFactor('mortgage', input.liquidityNeedScore)
	];
	return {
		option: 'mortgage',
		name: OPTION_NAMES.mortgage,
		available: input.mortgageRemainingPLN > 0,
		availabilityReason:
			input.mortgageRemainingPLN > 0 ? undefined : 'Brak otwartego kredytu hipotecznego',
		factors,
		total: factors.reduce((sum, f) => sum + f.pp, 0)
	};
}

function scoreBonds(input: OptionInputs): OptionScore {
	const netYieldPct = input.bondsYieldPct * (1 - capitalGainsTax);
	const factors: ScoreFactor[] = [
		{ label: 'Oczekiwany zysk netto', pp: netYieldPct },
		driftFactor('bonds', input.allocationDrift),
		liquidityFactor('bonds', input.liquidityNeedScore)
	];
	return {
		option: 'bonds',
		name: OPTION_NAMES.bonds,
		available: true,
		factors,
		total: factors.reduce((sum, f) => sum + f.pp, 0)
	};
}

function scoreBrokerage(input: OptionInputs): OptionScore {
	const netReturnPct = input.brokerageReturnPct * (1 - capitalGainsTax);
	const factors: ScoreFactor[] = [
		{ label: 'Oczekiwany zysk po Belce', pp: netReturnPct },
		driftFactor('brokerage', input.allocationDrift),
		liquidityFactor('brokerage', input.liquidityNeedScore)
	];
	return {
		option: 'brokerage',
		name: OPTION_NAMES.brokerage,
		available: true,
		factors,
		total: factors.reduce((sum, f) => sum + f.pp, 0)
	};
}

export function rankOptions(input: OptionInputs): OptionScore[] {
	const all = [
		scoreIKZE(input),
		scoreIKE(input),
		scorePPK(input),
		scoreMortgage(input),
		scoreBonds(input),
		scoreBrokerage(input)
	];
	return all.sort((a, b) => {
		if (a.available !== b.available) return a.available ? -1 : 1;
		return b.total - a.total;
	});
}
