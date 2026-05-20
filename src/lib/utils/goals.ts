export function goalProgress(current: number, target: number): number {
	if (target <= 0) return 0;
	return Math.min(100, (current / target) * 100);
}

export function goalRemaining(current: number, target: number): number {
	return Math.max(0, target - current);
}

export function projectGoalHitDate(
	current: number,
	target: number,
	monthlyContribution: number,
	today: Date = new Date()
): Date | null {
	if (current >= target) return today;
	if (monthlyContribution <= 0) return null;
	const remaining = target - current;
	const monthsNeeded = Math.ceil(remaining / monthlyContribution);
	const result = new Date(today);
	result.setMonth(result.getMonth() + monthsNeeded);
	return result;
}
