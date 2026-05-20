import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
import type { PageLoad } from './$types';
import type { SalariesData } from '$lib/types/salaries';
import type { CpiSeries } from '$lib/types/cpi';

const EMPTY_CPI: CpiSeries = { points: [], base_year: null, latest_year: null, source: '' };

export const load: PageLoad = async ({ fetch, url }) => {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}

		const owner = url.searchParams.get('owner');
		const dateFrom = url.searchParams.get('date_from');
		const dateTo = url.searchParams.get('date_to');
		const company = url.searchParams.get('company');

		const params = new URLSearchParams();
		if (owner) params.set('owner', owner);
		if (dateFrom) params.set('date_from', dateFrom);
		if (dateTo) params.set('date_to', dateTo);
		if (company) params.set('company', company);

		const [salariesResponse, personasResponse, cpiResponse] = await Promise.all([
			fetch(`${apiUrl}/api/salaries?${params.toString()}`),
			fetch(`${apiUrl}/api/personas`),
			fetch(`${apiUrl}/api/cpi/series`)
		]);

		if (!salariesResponse.ok) {
			throw error(salariesResponse.status, 'Failed to load salary records');
		}

		const data: SalariesData = await salariesResponse.json();
		const personas = personasResponse.ok ? await personasResponse.json() : [];
		const cpiSeries: CpiSeries = cpiResponse.ok ? await cpiResponse.json() : EMPTY_CPI;

		return {
			salaries: data,
			filters: {
				owner,
				date_from: dateFrom,
				date_to: dateTo,
				company
			},
			personas,
			cpiSeries
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load salary records');
	}
};
