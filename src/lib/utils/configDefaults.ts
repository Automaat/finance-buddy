import { resolveApiUrl } from '$lib/api';

export interface ConfigDefaults {
	// expected_return_rate × 100 → e.g. 0.07 stored as 7.0
	annualReturnPct: number;
	// withdrawal_rate × 100 → e.g. 0.04 stored as 4.0
	withdrawalRatePct: number;
	// Floor(years since birth_date) when available
	currentAge: number | null;
	retirementAge: number | null;
	monthlyExpensesPLN: number | null;
	annualExpensesPLN: number | null;
	monthlyMortgagePLN: number | null;
}

const FALLBACK: ConfigDefaults = {
	annualReturnPct: 7,
	withdrawalRatePct: 4,
	currentAge: null,
	retirementAge: null,
	monthlyExpensesPLN: null,
	annualExpensesPLN: null,
	monthlyMortgagePLN: null
};

type FetchLike = typeof fetch;

export async function loadConfigDefaults(fetchFn: FetchLike = fetch): Promise<ConfigDefaults> {
	try {
		const r = await fetchFn(`${resolveApiUrl()}/api/config`);
		if (!r.ok) return { ...FALLBACK };
		const cfg = (await r.json()) as Record<string, unknown>;
		const expectedReturn = Number(cfg.expected_return_rate ?? NaN);
		const withdrawal = Number(cfg.withdrawal_rate ?? NaN);
		const birthDate = typeof cfg.birth_date === 'string' ? cfg.birth_date : null;
		const retirementAge = typeof cfg.retirement_age === 'number' ? cfg.retirement_age : null;
		const monthlyExpenses = Number(cfg.monthly_expenses ?? NaN);
		const monthlyMortgage = Number(cfg.monthly_mortgage_payment ?? NaN);
		return {
			annualReturnPct: Number.isFinite(expectedReturn)
				? expectedReturn * 100
				: FALLBACK.annualReturnPct,
			withdrawalRatePct: Number.isFinite(withdrawal)
				? withdrawal * 100
				: FALLBACK.withdrawalRatePct,
			currentAge: birthDate ? ageFromBirthDate(birthDate) : null,
			retirementAge,
			monthlyExpensesPLN: Number.isFinite(monthlyExpenses) ? monthlyExpenses : null,
			annualExpensesPLN: Number.isFinite(monthlyExpenses) ? monthlyExpenses * 12 : null,
			monthlyMortgagePLN: Number.isFinite(monthlyMortgage) ? monthlyMortgage : null
		};
	} catch {
		return { ...FALLBACK };
	}
}

function ageFromBirthDate(iso: string): number | null {
	const parsed = new Date(iso);
	if (Number.isNaN(parsed.getTime())) return null;
	const now = new Date();
	let age = now.getFullYear() - parsed.getFullYear();
	const monthDiff = now.getMonth() - parsed.getMonth();
	if (monthDiff < 0 || (monthDiff === 0 && now.getDate() < parsed.getDate())) age -= 1;
	return age >= 0 ? age : null;
}
