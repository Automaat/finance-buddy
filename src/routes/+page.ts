import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';

export async function load({ fetch }) {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}

		const currentYear = new Date().getFullYear();

		const [dashboardRes, retirementRes, personasRes] = await Promise.all([
			fetch(`${apiUrl}/api/dashboard`),
			fetch(`${apiUrl}/api/retirement/stats?year=${currentYear}`),
			fetch(`${apiUrl}/api/personas`)
		]);

		if (!dashboardRes.ok) {
			throw error(dashboardRes.status, 'Failed to load dashboard data');
		}

		const dashboard = await dashboardRes.json();
		const retirement = retirementRes.ok ? await retirementRes.json() : [];
		const personas = personasRes.ok ? await personasRes.json() : [];

		return {
			...dashboard,
			retirementStats: retirement,
			personas,
			currentYear
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load dashboard data');
	}
}
