import { error, isHttpError } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';

export async function load({ fetch }) {
	const apiUrl = resolveApiUrl();

	const currentYear = new Date().getFullYear();

	const owners = (async () => {
		const res = await fetch(`${apiUrl}/api/users`);
		return res.ok ? await res.json() : [];
	})();

	const dashboardData = (async () => {
		try {
			const [dashboardRes, retirementRes] = await Promise.all([
				fetch(`${apiUrl}/api/dashboard`),
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
		currentYear
	};
}
