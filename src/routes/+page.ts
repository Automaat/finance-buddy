import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';

export async function load({ fetch }) {
	const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
	if (!apiUrl) {
		throw error(500, 'API URL is not configured');
	}

	const currentYear = new Date().getFullYear();

	const personas = (async () => {
		const res = await fetch(`${apiUrl}/api/personas`);
		return res.ok ? await res.json() : [];
	})();

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

	return {
		dashboardData,
		personas,
		currentYear
	};
}
