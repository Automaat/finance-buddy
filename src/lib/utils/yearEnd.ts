// Year-end contribution countdown helpers for IKE/IKZE caps.
// Caps don't roll over — Dec 31 is the deadline. The dashboard nudge
// escalates in Q3 (amber) and Q4 (red) when the limit is still not maxed.

export type CountdownTier = 'maxed' | 'safe' | 'warn' | 'urgent';

const MS_PER_DAY = 1000 * 60 * 60 * 24;

// daysUntilYearEnd returns the number of calendar days the user still has
// to contribute, inclusive of Dec 31. On Dec 31 it returns 1 (today is the
// last chance); on Jan 1 it returns 365 (or 366 in a leap year).
export function daysUntilYearEnd(now: Date): number {
	const year = now.getFullYear();
	const today = new Date(year, now.getMonth(), now.getDate());
	const yearEnd = new Date(year, 11, 31);
	const diffDays = Math.round((yearEnd.getTime() - today.getTime()) / MS_PER_DAY);
	return diffDays < 0 ? 0 : diffDays + 1;
}

// countdownTier picks the urgency level for the IKE/IKZE limit card.
// Maxed wraps any quarter; otherwise the month bucket drives the color.
export function countdownTier(now: Date, percentageUsed: number): CountdownTier {
	if (percentageUsed >= 100) return 'maxed';
	const month = now.getMonth() + 1;
	if (month <= 6) return 'safe';
	if (month <= 9) return 'warn';
	return 'urgent';
}

// daysLabel returns the Polish plural for "day" (dzień/dni).
// Polish: 1 → dzień, otherwise → dni.
export function daysLabel(days: number): string {
	return days === 1 ? 'dzień' : 'dni';
}
