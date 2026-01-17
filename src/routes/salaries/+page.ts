import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
import type { PageLoad } from './$types';
import type { SalariesData } from '$lib/types/salaries';

export const load: PageLoad = async ({ fetch, url }) => {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}

		// Build query params from URL
		const owner = url.searchParams.get('owner');
		const dateFrom = url.searchParams.get('date_from');
		const dateTo = url.searchParams.get('date_to');

		const params = new URLSearchParams();
		if (owner) params.set('owner', owner);
		if (dateFrom) params.set('date_from', dateFrom);
		if (dateTo) params.set('date_to', dateTo);

		// Fetch salaries with filters
		const response = await fetch(`${apiUrl}/api/salaries?${params.toString()}`);

		if (!response.ok) {
			throw error(response.status, 'Failed to load salary records');
		}

		const data: SalariesData = await response.json();

		return {
			salaries: data,
			filters: {
				owner,
				date_from: dateFrom,
				date_to: dateTo
			}
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load salary records');
	}
};
