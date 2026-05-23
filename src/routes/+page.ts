import { error, isHttpError } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import { resolveRangeParams } from '$lib/utils/dateRange';

export async function load({ fetch, url }) {
	const apiUrl = resolveApiUrl();

	const currentYear = new Date().getFullYear();

	const { range, dateFrom, dateTo } = resolveRangeParams(url.searchParams);
	const dashboardQS = new URLSearchParams();
	if (dateFrom) dashboardQS.set('date_from', dateFrom);
	if (dateTo) dashboardQS.set('date_to', dateTo);
	const dashboardURL = dashboardQS.toString()
		? `${apiUrl}/api/dashboard?${dashboardQS.toString()}`
		: `${apiUrl}/api/dashboard`;

	const owners = (async () => {
		const res = await fetch(`${apiUrl}/api/users`);
		return res.ok ? await res.json() : [];
	})();

	const dashboardData = (async () => {
		try {
			const [dashboardRes, retirementRes] = await Promise.all([
				fetch(dashboardURL),
				fetch(`${apiUrl}/api/retirement/stats?year=${currentYear}`)
			]);

			if (!dashboardRes.ok) {
				throw error(dashboardRes.status, 'Failed to load dashboard data');
			}

			const dashboard = await dashboardRes.json();
			const retirementStats = retirementRes.ok ? await retirementRes.json() : [];

			return {
				...dashboard,
				retirementStats
			};
		} catch (err) {
			if (isHttpError(err)) {
				throw err;
			}
			throw error(500, 'Failed to load dashboard data');
		}
	})();

	return {
		dashboardData,
		owners,
		currentYear,
		range,
		dateFrom,
		dateTo
	};
}
