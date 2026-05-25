// Small pure helpers used by the salaries route. Extracted from
// src/routes/salaries/+page.svelte as step 1 of the broader refactor in
// #614 — the file is still 1957 lines and a full split is a multi-day job.
// These five primitives are the cleanest first slice because they are
// stateless and were already pure functions inside the .svelte file.

import { formatPLN } from './format';

export function isNonNegative(value: number | null): boolean {
	return (value ?? 0) >= 0;
}

// Differs from format.ts's formatSignedPercent: this variant relies on the
// number's own minus glyph (regular hyphen) for negatives and only prepends
// a `+` for non-negatives, matching the legacy salaries UI exactly.
export function formatPctSigned(value: number | null): string {
	if (value == null || Number.isNaN(value)) return '—';
	const sign = value >= 0 ? '+' : '';
	return `${sign}${value.toFixed(1)}%`;
}

// Differs from format.ts's formatSignedPLN by always prefixing a sign
// (including `+` for 0) — preserves the salaries-row "delta" cell look.
export function formatPlnSigned(value: number | null): string {
	if (value == null || Number.isNaN(value)) return '—';
	const sign = value >= 0 ? '+' : '−';
	return `${sign}${formatPLN(Math.abs(value))}`;
}

function pad2(n: number): string {
	return n.toString().padStart(2, '0');
}

// toISOString() converts to UTC and shifts the day for TZ > UTC (e.g. PL in
// winter: 2026-12-31 00:00 local → 2026-12-30T23:00Z). Build YYYY-MM-DD from
// the local components to keep comparisons consistent with date-only values
// returned by the API.
export function isoDateLocal(d: Date): string {
	return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}
