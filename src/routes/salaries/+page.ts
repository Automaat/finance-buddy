import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';
import type {
	BonusEventsData,
	CompanyValuationsData,
	EquityGrantsData,
	SalariesData
} from '$lib/types/salaries';
import type { CpiSeries } from '$lib/types/cpi';
import type { OwnerOption } from '$lib/types/owners';

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
		const apiUrl = resolveApiUrl();

		const ownerUserId = url.searchParams.get('owner_user_id');
		const dateFrom = url.searchParams.get('date_from');
		const dateTo = url.searchParams.get('date_to');
		const company = url.searchParams.get('company');

		const params = new URLSearchParams();
		if (ownerUserId) params.set('owner_user_id', ownerUserId);
		if (dateFrom) params.set('date_from', dateFrom);
		if (dateTo) params.set('date_to', dateTo);
		if (company) params.set('company', company);

		// Bonuses, equity and valuations are fetched unfiltered: the total comp
		// summary has its own owner selector independent of the URL salary filter,
		// so it needs the full dataset to switch personas without re-loading.
		const [
			salariesResponse,
			ownersResponse,
			cpiResponse,
			bonusesResponse,
			equityResponse,
			valuationsResponse
		] = await Promise.all([
			fetch(`${apiUrl}/api/salaries?${params.toString()}`),
			fetch(`${apiUrl}/api/users`),
			fetch(`${apiUrl}/api/cpi/series`),
			fetch(`${apiUrl}/api/bonuses`),
			fetch(`${apiUrl}/api/equity-grants`),
			fetch(`${apiUrl}/api/company-valuations`)
		]);

		if (!salariesResponse.ok) {
			throw error(salariesResponse.status, 'Failed to load salary records');
		}

		const data: SalariesData = await salariesResponse.json();
		const owners: OwnerOption[] = ownersResponse.ok ? await ownersResponse.json() : [];
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
				owner_user_id: ownerUserId,
				date_from: dateFrom,
				date_to: dateTo,
				company
			},
			owners,
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
