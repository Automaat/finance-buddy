export function calculateNetWorth(assets: number, liabilities: number): number {
	return assets - liabilities;
}

export function calculateChange(
	current: number,
	previous: number
): {
	value: number;
	percent: number;
} {
	const value = current - previous;
	const percent = previous !== 0 ? (value / previous) * 100 : 0;
	return { value, percent };
}

export function calculateGoalProgress(current: number, target: number): number {
	return target !== 0 ? (current / target) * 100 : 0;
}

export function calculateMonthsRemaining(targetDate: Date): number {
	const now = new Date();
	let months =
		(targetDate.getFullYear() - now.getFullYear()) * 12 + (targetDate.getMonth() - now.getMonth());

	// Adjust for day-level precision: if target day hasn't been reached in the month, subtract 1
	if (targetDate.getDate() < now.getDate()) {
		months--;
	}

	return Math.max(0, months);
}
