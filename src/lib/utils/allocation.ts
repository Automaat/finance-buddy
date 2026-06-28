// sumsToHundred reports whether a set of allocation percentages adds up to
// 100 within tolerance. Shared by the config page (market allocation, exact)
// and the allocation-targets page (per-category targets, float-tolerant) so
// the "must total 100%" rule lives in one place.
//
// tolerance defaults to 0.01 (a hundredth of a percent) to absorb float drift
// from summing decimals; pass 0 for an exact integer match.
export function sumsToHundred(total: number, tolerance = 0.01): boolean {
	return Math.abs(total - 100) <= tolerance;
}

export interface AllocationBucket {
	category: string;
	owner_user_id: number | null;
	value: number;
	isOther?: boolean;
}

export function topNWithOther<T extends AllocationBucket>(
	items: T[],
	n: number
): AllocationBucket[] {
	const ranked = items
		.map((item, index) => ({ item, index }))
		.filter(({ item }) => Number.isFinite(item.value) && item.value > 0);

	const limit = Math.max(0, n);
	if (ranked.length <= limit) return ranked.map(({ item }) => item);

	ranked.sort((a, b) => b.item.value - a.item.value || a.index - b.index);

	const head = ranked.slice(0, limit).map(({ item }) => item);
	const tailSum = ranked.slice(limit).reduce((sum, { item }) => sum + item.value, 0);

	const other = { category: 'Inne', owner_user_id: null, value: tailSum, isOther: true };
	return [...head, other];
}
