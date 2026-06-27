import { error, isHttpError } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import { resolveRangeParams } from '$lib/utils/dateRange';
import type { CpiSeries } from '$lib/types/cpi';
import type { OwnerOption } from '$lib/types/owners';
import type { RealYieldAccount } from '$lib/components/RealYieldsTable.svelte';

// PpkStat / CategoryStat mirror GET /api/retirement/ppk-stats and
// /api/investment/{stock,bond}-stats — the subset the page renders. Declared
// here (not reused from $lib/types/retirement, whose PPKStats keys on a legacy
// `owner` string) so PageData keeps the owner_user_id shape +page.svelte reads.
interface PpkStat {
	owner_user_id: number | null;
	total_value: number;
	employee_contributed: number;
	employer_contributed: number;
	government_contributed: number;
	total_contributed: number;
	returns: number;
	roi_percentage: number;
}

interface CategoryStat {
	total_value: number;
	total_contributed: number;
	returns: number;
	roi_percentage: number;
}

// YearlyStat mirrors a GET /api/retirement/stats row. Only IKZE rows carry the
// PIT fields; the page renders those to surface the tax benefit of IKZE.
export interface YearlyStat {
	year: number;
	account_wrapper: string;
	owner_user_id: number | null;
	total_contributed: number;
	marginal_tax_rate: number | null;
	pit_savings: number | null;
}

// fetchJson GETs a URL and returns its parsed body, or `fallback` on any
// failure (non-OK status, network error, malformed JSON). Lets the page run
// every best-effort request inside one Promise.all without a rejection from
// one source tearing down the batch. The explicit return type keeps the
// fallback from narrowing T to `never[]`/`null` at each call site.
async function fetchJson<T>(fetchFn: typeof fetch, url: string, fallback: T): Promise<T> {
	try {
		const res = await fetchFn(url);
		if (!res.ok) return fallback;
		return (await res.json()) as T;
	} catch {
		return fallback;
	}
}

export async function load({ fetch, url }) {
	try {
		const apiUrl = resolveApiUrl();

		const { range, dateFrom, dateTo } = resolveRangeParams(url.searchParams);
		const dashboardQS = new URLSearchParams();
		if (dateFrom) dashboardQS.set('date_from', dateFrom);
		if (dateTo) dashboardQS.set('date_to', dateTo);
		const dashboardURL = dashboardQS.toString()
			? `${apiUrl}/api/dashboard?${dashboardQS.toString()}`
			: `${apiUrl}/api/dashboard`;

		// Fire every request concurrently — only the dashboard fetch gates the
		// page, so there's no reason to serialize the rest. Wall-clock load
		// collapses from the sum of 7 round-trips to the slowest single one.
		// Each fetch is wrapped so a network rejection can't reject the batch:
		// the dashboard's HTTP-status check is enforced explicitly below, and
		// every other source keeps its current degrade-to-default semantics.
		const cpiDefault: CpiSeries = {
			points: [],
			base_year: null,
			latest_year: null,
			source: ''
		};

		const [
			dashboardRes,
			ppkStats,
			stockStats,
			bondStats,
			owners,
			realYieldAccounts,
			cpiSeries,
			retirementStats
		] = await Promise.all([
			fetch(dashboardURL),
			fetchJson<PpkStat[]>(fetch, `${apiUrl}/api/retirement/ppk-stats`, []),
			fetchJson<CategoryStat | null>(fetch, `${apiUrl}/api/investment/stock-stats`, null),
			fetchJson<CategoryStat | null>(fetch, `${apiUrl}/api/investment/bond-stats`, null),
			fetchJson<OwnerOption[]>(fetch, `${apiUrl}/api/users`, []),
			// Accounts carry per-account real_yield_pct (post-Belka, post-CPI);
			// the CPI series feeds the cumulative-inflation chart — both power
			// the real-return section and degrade to empty on failure without
			// taking down the page.
			fetchJson<{ assets?: RealYieldAccount[] }>(fetch, `${apiUrl}/api/accounts`, {}).then((data) =>
				(data.assets ?? []).filter((a: RealYieldAccount) => a.interest_rate_pct != null)
			),
			fetchJson<CpiSeries>(fetch, `${apiUrl}/api/cpi/series`, cpiDefault),
			// Yearly retirement stats (current year, all owners) — only the IKZE
			// rows with a computed PIT benefit are surfaced below.
			fetchJson<YearlyStat[]>(fetch, `${apiUrl}/api/retirement/stats`, [])
		]);

		// IKZE is the only Polish wrapper with an up-front PIT deduction; the
		// backend fills pit_savings on those rows when the owner has a salary on
		// record. Surface just those so the tax benefit is visible.
		const ikzePitStats = retirementStats.filter(
			(s) => s.account_wrapper === 'IKZE' && s.pit_savings != null
		);

		if (!dashboardRes.ok) {
			throw error(dashboardRes.status, 'Failed to load dashboard data');
		}

		const dashboard = await dashboardRes.json();
		// The always-latest tiles/metric cards reflect this snapshot regardless
		// of the date-range filter; the dashboard exposes it so we don't pay for
		// a full /api/snapshots list just to read one date.
		const snapshotDate: string | null = dashboard.latest_snapshot_date ?? null;

		return {
			allocationAnalysis: dashboard.allocation_analysis,
			investmentTimeSeries: dashboard.investment_time_series,
			wrapperTimeSeries: dashboard.wrapper_time_series,
			categoryTimeSeries: dashboard.category_time_series,
			ppkStats,
			stockStats,
			bondStats,
			owners,
			realYieldAccounts,
			cpiSeries,
			ikzePitStats,
			snapshotDate,
			range,
			dateFrom,
			dateTo
		};
	} catch (err) {
		if (isHttpError(err)) {
			throw err;
		}
		throw error(500, 'Failed to load metrics data');
	}
}
