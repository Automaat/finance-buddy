export interface FireGapInputs {
	currentAge: number;
	retirementAge: number;
	lifeExpectancy: number;
	currentPortfolioPLN: number;
	annualContributionPLN: number;
	expectedReturnPct: number;
	inflationPct: number;
	withdrawalRatePct: number;
	monthlyPensionNetPLN: number;
}

export interface FireGapYear {
	age: number;
	year: number;
	portfolioPLN: number;
	privateMonthlyIncomePLN: number;
	zusMonthlyIncomePLN: number;
	gapPLN: number;
	afterRetirement: boolean;
}

export function projectFireGap(
	input: FireGapInputs,
	calendarYearAt: (age: number) => number = (age) =>
		new Date().getFullYear() + (age - input.currentAge)
): FireGapYear[] {
	const out: FireGapYear[] = [];
	const r = Math.max(-0.99, input.expectedReturnPct / 100);
	const inflation = Math.max(0, input.inflationPct / 100);
	const wr = Math.max(0, input.withdrawalRatePct / 100);
	let portfolio = Math.max(0, input.currentPortfolioPLN);
	for (let age = input.currentAge; age <= input.lifeExpectancy; age++) {
		const afterRetirement = age >= input.retirementAge;
		if (!afterRetirement) {
			portfolio = portfolio * (1 + r) + Math.max(0, input.annualContributionPLN);
		}
		const annualWithdrawal = afterRetirement ? portfolio * wr : 0;
		if (afterRetirement) {
			portfolio = Math.max(0, portfolio - annualWithdrawal);
			portfolio *= 1 + r;
		}
		const yearsSinceNow = age - input.currentAge;
		const inflationFactor = Math.pow(1 + inflation, yearsSinceNow);
		const zusMonthly = afterRetirement
			? Math.max(0, input.monthlyPensionNetPLN) * inflationFactor
			: 0;
		const privateMonthly = annualWithdrawal / 12;
		out.push({
			age,
			year: calendarYearAt(age),
			portfolioPLN: portfolio,
			privateMonthlyIncomePLN: privateMonthly,
			zusMonthlyIncomePLN: zusMonthly,
			gapPLN: privateMonthly - zusMonthly,
			afterRetirement
		});
	}
	return out;
}
