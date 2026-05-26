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
			const [dashboardRes, retirementRes, bondsRes, driftRes, ladderRes] = await Promise.all([
				fetch(dashboardURL),
				fetch(`${apiUrl}/api/retirement/stats?year=${currentYear}`),
				fetch(`${apiUrl}/api/bonds`),
				fetch(`${apiUrl}/api/allocation/drift`),
				fetch(`${apiUrl}/api/bonds/maturity-ladder`)
			]);

			if (!dashboardRes.ok) {
				throw error(dashboardRes.status, 'Failed to load dashboard data');
			}

			const dashboard = await dashboardRes.json();
			const retirementStats = retirementRes.ok ? await retirementRes.json() : [];
			const bondsBody = bondsRes.ok ? await bondsRes.json() : { total_value: 0, total_count: 0 };
			const allocationDrift = driftRes.ok ? await driftRes.json() : { scopes: [] };
			const bondsLadder = ladderRes.ok
				? await ladderRes.json()
				: { events: [], next_maturity: null, tax_rate_pct: 19 };

			return {
				...dashboard,
				retirementStats,
				treasuryBondsValue: bondsBody.total_value ?? 0,
				treasuryBondsCount: bondsBody.total_count ?? 0,
				allocationDrift,
				bondsNextMaturity: bondsLadder.next_maturity
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
