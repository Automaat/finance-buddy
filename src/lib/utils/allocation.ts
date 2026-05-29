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
