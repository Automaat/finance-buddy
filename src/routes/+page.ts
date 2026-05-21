import { error } from '@sveltejs/kit';
import { API_URL_NOT_CONFIGURED_MESSAGE, resolveApiUrl } from '$lib/utils/api';

export async function load({ fetch }) {
	const apiUrl = resolveApiUrl();
	if (!apiUrl) {
		throw error(500, API_URL_NOT_CONFIGURED_MESSAGE);
	}

	const currentYear = new Date().getFullYear();

	const personas = (async () => {
		try {
			const res = await fetch(`${apiUrl}/api/personas`);
			return res.ok ? await res.json() : [];
		} catch {
			return [];
		}
	})();
	personas.catch(() => {});

	const dashboardData = (async () => {
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
	})();
	dashboardData.catch(() => {});

	return {
		dashboardData,
		personas,
		currentYear
	};
}
