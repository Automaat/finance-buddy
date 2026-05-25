import { PL_RULES } from './pl_rules.generated';

export type IKZELimitKind = 'employee' | 'b2b';

export interface IKZEOptimizerInput {
	year: number;
	limitKind: IKZELimitKind;
	alreadyContributed: number;
	marginalTaxRate: number;
	now?: Date;
	limitOverride?: number;
}

export interface IKZEOptimizerResult {
	annualTarget: number;
	remaining: number;
	monthlyTarget: number;
	monthsLeft: number;
	refundEstimate: number;
	limitSource: 'rule' | 'override';
}

export function limitFromRules(year: number, kind: IKZELimitKind): number | null {
	const key = kind === 'b2b' ? `ikze_limit_b2b_${year}` : `ikze_limit_${year}`;
	const rule = (PL_RULES as Record<string, { value: number }>)[key];
	return rule ? rule.value : null;
}

export function monthsLeftIn(year: number, now: Date): number {
	if (now.getFullYear() < year) return 12;
	if (now.getFullYear() > year) return 0;
	return 12 - now.getMonth();
}

export function optimizeIKZE(input: IKZEOptimizerInput): IKZEOptimizerResult {
	const now = input.now ?? new Date();
	const ruleLimit = limitFromRules(input.year, input.limitKind);
	const limit = input.limitOverride ?? ruleLimit ?? 0;
	const annualTarget = Math.max(0, limit);
	const contributed = Math.max(0, input.alreadyContributed);
	const remaining = Math.max(0, annualTarget - contributed);
	const monthsLeft = monthsLeftIn(input.year, now);
	const monthlyTarget = monthsLeft > 0 ? remaining / monthsLeft : 0;
	const refundEstimate = annualTarget * Math.max(0, input.marginalTaxRate);
	return {
		annualTarget,
		remaining,
		monthlyTarget,
		monthsLeft,
		refundEstimate,
		limitSource: input.limitOverride != null ? 'override' : 'rule'
	};
}
