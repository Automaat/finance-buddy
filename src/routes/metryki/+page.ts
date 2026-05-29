import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import { resolveRangeParams } from '$lib/utils/dateRange';
import type { CpiSeries } from '$lib/types/cpi';
import type { RealYieldAccount } from '$lib/components/RealYieldsTable.svelte';

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

		const dashboardRes = await fetch(dashboardURL);

		if (!dashboardRes.ok) {
			throw error(dashboardRes.status, 'Failed to load dashboard data');
		}

		const dashboard = await dashboardRes.json();

		// Fetch PPK stats
		const ppkStatsRes = await fetch(`${apiUrl}/api/retirement/ppk-stats`);
		let ppkStats = [];
		if (ppkStatsRes.ok) {
			ppkStats = await ppkStatsRes.json();
		}

		// Fetch stock stats
		const stockStatsRes = await fetch(`${apiUrl}/api/investment/stock-stats`);
		let stockStats = null;
		if (stockStatsRes.ok) {
			stockStats = await stockStatsRes.json();
		}

		// Fetch bond stats
		const bondStatsRes = await fetch(`${apiUrl}/api/investment/bond-stats`);
		let bondStats = null;
		if (bondStatsRes.ok) {
			bondStats = await bondStatsRes.json();
		}

		// Fetch owners for owner_user_id resolution
		const ownersRes = await fetch(`${apiUrl}/api/users`);
		const owners = ownersRes.ok ? await ownersRes.json() : [];

		// Accounts carry per-account real_yield_pct (post-Belka, post-CPI) and
		// the CPI series feeds the cumulative-inflation chart — both power the
		// real-return section. These are best-effort: a failure (network or
		// malformed JSON) degrades only that section to empty, never the whole
		// metrics page, so they get their own try/catch.
		let realYieldAccounts: RealYieldAccount[] = [];
		let cpiSeries: CpiSeries = { points: [], base_year: null, latest_year: null, source: '' };
		try {
			const accountsRes = await fetch(`${apiUrl}/api/accounts`);
			if (accountsRes.ok) {
				const accountsData = await accountsRes.json();
				realYieldAccounts = (accountsData.assets ?? []).filter(
					(a: RealYieldAccount) => a.interest_rate_pct != null
				);
			}
		} catch {
			realYieldAccounts = [];
		}
		try {
			const cpiSeriesRes = await fetch(`${apiUrl}/api/cpi/series`);
			if (cpiSeriesRes.ok) {
				cpiSeries = await cpiSeriesRes.json();
			}
		} catch {
			// keep the empty default
		}

		return {
			metricCards: dashboard.metric_cards,
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
			range,
			dateFrom,
			dateTo
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load metrics data');
	}
}
