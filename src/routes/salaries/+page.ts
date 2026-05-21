import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
import type { PageLoad } from './$types';
import type {
	BonusEventsData,
	CompanyValuationsData,
	EquityGrantsData,
	SalariesData
} from '$lib/types/salaries';
import type { CpiSeries } from '$lib/types/cpi';

const EMPTY_CPI: CpiSeries = { points: [], base_year: null, latest_year: null, source: '' };
const EMPTY_BONUSES: BonusEventsData = {
	bonus_events: [],
	total_count: 0,
	available_companies: []
};
const EMPTY_EQUITY: EquityGrantsData = {
	equity_grants: [],
	total_count: 0,
	available_companies: []
};
const EMPTY_VALUATIONS: CompanyValuationsData = {
	company_valuations: [],
	total_count: 0,
	available_companies: []
};

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

		const equityParams = new URLSearchParams();
		if (owner) equityParams.set('owner', owner);
		if (company) equityParams.set('company', company);

		const valuationParams = new URLSearchParams();
		if (company) valuationParams.set('company', company);

		const [
			salariesResponse,
			personasResponse,
			cpiResponse,
			bonusesResponse,
			equityResponse,
			valuationsResponse
		] = await Promise.all([
			fetch(`${apiUrl}/api/salaries?${params.toString()}`),
			fetch(`${apiUrl}/api/personas`),
			fetch(`${apiUrl}/api/cpi/series`),
			fetch(`${apiUrl}/api/bonuses?${params.toString()}`),
			fetch(`${apiUrl}/api/equity-grants?${equityParams.toString()}`),
			fetch(`${apiUrl}/api/company-valuations?${valuationParams.toString()}`)
		]);

		if (!salariesResponse.ok) {
			throw error(salariesResponse.status, 'Failed to load salary records');
		}

		const data: SalariesData = await salariesResponse.json();
		const personas = personasResponse.ok ? await personasResponse.json() : [];
		const cpiSeries: CpiSeries = cpiResponse.ok ? await cpiResponse.json() : EMPTY_CPI;
		const bonuses: BonusEventsData = bonusesResponse.ok
			? await bonusesResponse.json()
			: EMPTY_BONUSES;
		const equity: EquityGrantsData = equityResponse.ok ? await equityResponse.json() : EMPTY_EQUITY;
		const valuations: CompanyValuationsData = valuationsResponse.ok
			? await valuationsResponse.json()
			: EMPTY_VALUATIONS;

		return {
			salaries: data,
			filters: {
				owner,
				date_from: dateFrom,
				date_to: dateTo,
				company
			},
			personas,
			cpiSeries,
			bonuses,
			equity,
			valuations
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load salary records');
	}
};
