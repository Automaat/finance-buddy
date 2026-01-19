import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';

export async function load({ fetch }) {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}

		const dashboardRes = await fetch(`${apiUrl}/api/dashboard`);

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

		return {
			metricCards: dashboard.metric_cards,
			allocationAnalysis: dashboard.allocation_analysis,
			investmentTimeSeries: dashboard.investment_time_series,
			wrapperTimeSeries: dashboard.wrapper_time_series,
			categoryTimeSeries: dashboard.category_time_series,
			ppkStats,
			stockStats,
			bondStats
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load metrics data');
	}
}
