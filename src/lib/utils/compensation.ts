export enum ContractType {
	UOP = 'uop',
	B2B_MONTHLY = 'b2b_monthly',
	B2B_HOURLY = 'b2b_hourly'
}

export enum B2BTaxForm {
	LINIOWY = 'liniowy',
	RYCZALT = 'ryczalt',
	SKALA = 'skala'
}

export enum ZUSTier {
	ULGA = 'ulga',
	PREFERENCYJNY = 'preferencyjny',
	PELNY = 'pelny'
}

export interface OfferInput {
	name: string;
	contractType: ContractType;
	grossMonthly?: number;
	ppkEnabled?: boolean;
	netInvoice?: number;
	hourlyRate?: number;
	hoursPerMonth?: number;
	taxForm?: B2BTaxForm;
	zusTier?: ZUSTier;
	accountingCost?: number;
	rsuAnnual?: number;
	isCurrentJob?: boolean;
}

export interface OfferBreakdown {
	name: string;
	contractType: ContractType;
	grossMonthly: number;
	zusEmployee: number;
	healthInsurance: number;
	pit: number;
	ppkEmployee: number;
	accountingCost: number;
	netMonthly: number;
	netAnnual: number;
	rsuAfterTax: number;
	totalAnnual: number;
	employerCost: number;
	effectiveTaxRate: number;
	vacationEquivalent: number;
	isCurrentJob: boolean;
}

// 2026 tax constants
const ZUS_EMERYTALNE = 0.0976;
const ZUS_RENTOWE = 0.015;
const ZUS_CHOROBOWE = 0.0245;
const ZUS_EMPLOYER = 0.2048;
const ZUS_CAP = 282_600;
const HEALTH_RATE = 0.09;
const PIT_12 = 0.12;
const PIT_32 = 0.32;
const PIT_THRESHOLD = 120_000;
const KWOTA_WOLNA = 30_000;
const KUP_MONTHLY = 250;
const PPK_EE = 0.02;
const PPK_ER = 0.015;

const B2B_ZUS_MONTHLY: Record<ZUSTier, number> = {
	[ZUSTier.ULGA]: 0,
	[ZUSTier.PREFERENCYJNY]: 456,
	[ZUSTier.PELNY]: 1927
};

const B2B_HEALTH_LINIOWY = 0.049;
const B2B_HEALTH_LINIOWY_MIN = 498;
const B2B_HEALTH_RYCZALT_TIERS = [
	{ max: 60_000, health: 498 },
	{ max: 300_000, health: 831 },
	{ max: Infinity, health: 1495 }
];
const B2B_LINIOWY = 0.19;
const B2B_RYCZALT_IT = 0.12;
const RSU_TAX = 0.19;
const WORKING_DAYS = 21;
const VACATION_DAYS = 26;

function progressivePIT(taxBase: number): number {
	if (taxBase <= 0) return 0;
	const reduction = PIT_12 * KWOTA_WOLNA;
	const pit =
		taxBase <= PIT_THRESHOLD
			? PIT_12 * taxBase - reduction
			: PIT_12 * PIT_THRESHOLD + PIT_32 * (taxBase - PIT_THRESHOLD) - reduction;
	return Math.max(0, pit);
}

export function calculateUoP(grossMonthly: number, ppkEnabled = false): OfferBreakdown {
	const annual = grossMonthly * 12;
	const capped = Math.min(annual, ZUS_CAP);

	const zusAnnual = ZUS_EMERYTALNE * capped + ZUS_RENTOWE * capped + ZUS_CHOROBOWE * annual;
	const zusMonthly = zusAnnual / 12;

	const healthMonthly = (HEALTH_RATE * (annual - zusAnnual)) / 12;

	const taxBase = annual - zusAnnual - KUP_MONTHLY * 12;
	const pitMonthly = progressivePIT(taxBase) / 12;

	const ppkEe = ppkEnabled ? PPK_EE * grossMonthly : 0;
	const ppkEr = ppkEnabled ? PPK_ER * grossMonthly : 0;

	const netMonthly = grossMonthly - zusMonthly - healthMonthly - pitMonthly - ppkEe;
	const deductions = zusMonthly + healthMonthly + pitMonthly + ppkEe;

	return {
		name: '',
		contractType: ContractType.UOP,
		grossMonthly,
		zusEmployee: zusMonthly,
		healthInsurance: healthMonthly,
		pit: pitMonthly,
		ppkEmployee: ppkEe,
		accountingCost: 0,
		netMonthly,
		netAnnual: netMonthly * 12,
		rsuAfterTax: 0,
		totalAnnual: netMonthly * 12,
		employerCost: grossMonthly * (1 + ZUS_EMPLOYER) + ppkEr,
		effectiveTaxRate: grossMonthly > 0 ? (deductions / grossMonthly) * 100 : 0,
		vacationEquivalent: 0,
		isCurrentJob: false
	};
}

export function calculateB2B(
	monthlyRevenue: number,
	taxForm: B2BTaxForm,
	zusTier: ZUSTier,
	accountingCost = 0
): OfferBreakdown {
	const annual = monthlyRevenue * 12;
	const zus = B2B_ZUS_MONTHLY[zusTier];
	const zusAnnual = zus * 12;
	const costs = accountingCost * 12;
	const income = annual - zusAnnual - costs;

	let healthMonthly: number;
	let pitAnnual: number;

	switch (taxForm) {
		case B2BTaxForm.LINIOWY:
			healthMonthly = Math.max(B2B_HEALTH_LINIOWY_MIN, B2B_HEALTH_LINIOWY * (income / 12));
			pitAnnual = B2B_LINIOWY * Math.max(0, income);
			break;
		case B2BTaxForm.RYCZALT: {
			const tier = B2B_HEALTH_RYCZALT_TIERS.find((t) => annual <= t.max)!;
			healthMonthly = tier.health;
			pitAnnual = B2B_RYCZALT_IT * Math.max(0, annual - zusAnnual);
			break;
		}
		case B2BTaxForm.SKALA:
			healthMonthly = HEALTH_RATE * Math.max(0, income / 12);
			pitAnnual = progressivePIT(Math.max(0, income));
			break;
	}

	const pitMonthly = pitAnnual / 12;
	const netMonthly = monthlyRevenue - zus - healthMonthly - pitMonthly - accountingCost;
	const dailyNet = netMonthly / WORKING_DAYS;
	const vacationEquivalent = (VACATION_DAYS * dailyNet) / 12;
	const deductions = zus + healthMonthly + pitMonthly;

	return {
		name: '',
		contractType: ContractType.B2B_MONTHLY,
		grossMonthly: monthlyRevenue,
		zusEmployee: zus,
		healthInsurance: healthMonthly,
		pit: pitMonthly,
		ppkEmployee: 0,
		accountingCost,
		netMonthly,
		netAnnual: netMonthly * 12,
		rsuAfterTax: 0,
		totalAnnual: netMonthly * 12,
		employerCost: monthlyRevenue,
		effectiveTaxRate: monthlyRevenue > 0 ? (deductions / monthlyRevenue) * 100 : 0,
		vacationEquivalent,
		isCurrentJob: false
	};
}

export function calculateRSUAfterTax(annualValue: number): number {
	return annualValue * (1 - RSU_TAX);
}

export function calculateOffer(input: OfferInput): OfferBreakdown {
	let breakdown: OfferBreakdown;

	switch (input.contractType) {
		case ContractType.UOP:
			breakdown = calculateUoP(input.grossMonthly ?? 0, input.ppkEnabled ?? false);
			break;
		case ContractType.B2B_MONTHLY:
			breakdown = calculateB2B(
				input.netInvoice ?? 0,
				input.taxForm ?? B2BTaxForm.LINIOWY,
				input.zusTier ?? ZUSTier.PELNY,
				input.accountingCost ?? 0
			);
			break;
		case ContractType.B2B_HOURLY: {
			const revenue = (input.hourlyRate ?? 0) * (input.hoursPerMonth ?? 160);
			breakdown = calculateB2B(
				revenue,
				input.taxForm ?? B2BTaxForm.LINIOWY,
				input.zusTier ?? ZUSTier.PELNY,
				input.accountingCost ?? 0
			);
			breakdown.contractType = ContractType.B2B_HOURLY;
			break;
		}
	}

	breakdown.name = input.name;
	breakdown.isCurrentJob = input.isCurrentJob ?? false;

	if (input.rsuAnnual && input.rsuAnnual > 0) {
		breakdown.rsuAfterTax = calculateRSUAfterTax(input.rsuAnnual);
		breakdown.totalAnnual = breakdown.netAnnual + breakdown.rsuAfterTax;
	}

	return breakdown;
}

/** Search for the minimum gross/invoice/rate to match targetNetMonthly. */
export function findBreakEvenAmount(template: OfferInput, targetNetMonthly: number): number {
	// hoursPerMonth=0 means revenue is always 0 — no break-even possible
	if (template.contractType === ContractType.B2B_HOURLY && !((template.hoursPerMonth ?? 0) > 0)) {
		return 0;
	}

	const isRyczaltB2B =
		(template.contractType === ContractType.B2B_MONTHLY ||
			template.contractType === ContractType.B2B_HOURLY) &&
		template.taxForm === B2BTaxForm.RYCZALT;

	const hi = targetNetMonthly * 4;

	// For B2B ryczałt, healthMonthly has discrete tier jumps breaking monotonicity,
	// so binary search is unreliable — use linear search for correctness.
	if (isRyczaltB2B) {
		for (let amount = 0; amount <= hi; amount++) {
			const input = { ...template };
			switch (template.contractType) {
				case ContractType.B2B_MONTHLY:
					input.netInvoice = amount;
					break;
				case ContractType.B2B_HOURLY:
					input.hourlyRate = amount;
					break;
			}
			if (calculateOffer(input).netMonthly >= targetNetMonthly) {
				return amount;
			}
		}
		return Math.ceil(hi);
	}

	// For other contract types / tax forms, netMonthly is monotone — binary search is safe.
	let lo = 0;
	let hiVal = hi;

	for (let i = 0; i < 60; i++) {
		const mid = (lo + hiVal) / 2;
		const input = { ...template };
		switch (template.contractType) {
			case ContractType.UOP:
				input.grossMonthly = mid;
				break;
			case ContractType.B2B_MONTHLY:
				input.netInvoice = mid;
				break;
			case ContractType.B2B_HOURLY:
				input.hourlyRate = mid;
				break;
		}
		if (calculateOffer(input).netMonthly < targetNetMonthly) {
			lo = mid;
		} else {
			hiVal = mid;
		}
	}

	return Math.ceil(hiVal);
}
