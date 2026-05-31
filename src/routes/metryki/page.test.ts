import { describe, expect, it, beforeAll, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import Page from './+page.svelte';
import { load } from './+page';

beforeAll(() => {
	Object.defineProperty(window, 'matchMedia', {
		writable: true,
		value: vi.fn().mockImplementation((query: string) => ({
			matches: false,
			media: query,
			onchange: null,
			addListener: vi.fn(),
			removeListener: vi.fn(),
			addEventListener: vi.fn(),
			removeEventListener: vi.fn(),
			dispatchEvent: vi.fn()
		}))
	});
});

vi.mock('$env/dynamic/public', () => ({
	env: {
		PUBLIC_API_URL_BROWSER: 'http://localhost:8000',
		PUBLIC_API_URL: 'http://localhost:8000'
	}
}));

vi.mock('$app/environment', () => ({ browser: false }));

vi.mock('echarts', () => ({
	init: vi.fn(() => ({ setOption: vi.fn(), resize: vi.fn(), dispose: vi.fn() }))
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn(),
	invalidateAll: vi.fn()
}));

vi.mock('$app/stores', async () => {
	const { readable } = await import('svelte/store');
	return {
		page: readable({ url: new URL('http://localhost/metryki') })
	};
});

const metricCards = {
	property_sqm: 0,
	emergency_fund_months: 6,
	retirement_income_monthly: 0,
	mortgage_remaining: 0,
	mortgage_months_left: 0,
	mortgage_years_left: 0,
	retirement_total: 0,
	investment_contributions: 0,
	investment_returns: 0,
	savings_rate: null,
	debt_to_income_ratio: null,
	hour_of_work_cost: null,
	hour_of_life_cost: null
};

const mockData = {
	user: null,
	metricCards,
	allocationAnalysis: {
		by_category: [],
		by_wrapper: [],
		rebalancing: [],
		total_investment_value: 0
	},
	investmentTimeSeries: [],
	wrapperTimeSeries: { ike: [], ikze: [], ppk: [] },
	categoryTimeSeries: { stock: [], bond: [] },
	ppkStats: [],
	stockStats: null,
	bondStats: null,
	owners: [],
	realYieldAccounts: [],
	cpiSeries: { points: [], base_year: null, latest_year: null, source: '' },
	range: 'all' as const,
	dateFrom: null,
	dateTo: null
};

describe('Metryki Page', () => {
	it('renders the main heading', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: 'Metryki', level: 1 })).toBeTruthy();
	});

	it('renders the financial overview section', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: 'Przegląd finansowy' })).toBeTruthy();
	});

	it('renders the investing guidance section', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: 'Jak inwestować nowe pieniądze' })).toBeTruthy();
	});

	it('renders the emergency fund metric card with its formatted value', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByText('Ile miesięcy bez pracy')).toBeTruthy();
		expect(screen.getByText('6,00')).toBeTruthy();
	});

	it('shows the no-rebalancing message when there are no suggestions', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByText(/Portfel jest zgodny z docelową alokacją/)).toBeTruthy();
	});

	it('hides optional metric cards when their values are null', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.queryByText('Ile oszczędzamy miesięcznie')).toBeNull();
		expect(screen.queryByText('Stosunek długu do dochodu')).toBeNull();
		expect(screen.queryByText('Koszt godziny pracy')).toBeNull();
		expect(screen.queryByText('Koszt godziny życia')).toBeNull();
	});

	it('omits PPK, stock and bond summary sections when no data', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.queryByRole('heading', { name: 'Podsumowanie PPK' })).toBeNull();
		expect(screen.queryByRole('heading', { name: 'Podsumowanie Akcji' })).toBeNull();
		expect(screen.queryByRole('heading', { name: 'Podsumowanie Obligacji' })).toBeNull();
	});
});

describe('Metryki Page with populated data', () => {
	const populatedData = {
		user: null,
		metricCards: {
			property_sqm: 42,
			emergency_fund_months: 8,
			retirement_income_monthly: 1200,
			mortgage_remaining: 300000,
			mortgage_months_left: 240,
			mortgage_years_left: 20,
			retirement_total: 90000,
			investment_contributions: 50000,
			investment_returns: 12000,
			savings_rate: 22.5,
			debt_to_income_ratio: 28,
			hour_of_work_cost: 75,
			hour_of_life_cost: 30
		},
		allocationAnalysis: {
			by_category: [{ category: 'stock', current_percentage: 60, target_percentage: 70 }],
			by_wrapper: [{ wrapper: 'IKE', value: 30000 }],
			rebalancing: [{ category: 'stock', amount: 5000 }],
			total_investment_value: 80000
		},
		investmentTimeSeries: [{ date: '2024-01', value: 50000, contributions: 45000 }],
		wrapperTimeSeries: {
			ike: [{ date: '2024-01', value: 30000, contributions: 28000 }],
			ikze: [{ date: '2024-01', value: 10000, contributions: 9000 }],
			ppk: [{ date: '2024-01', value: 20000, contributions: 18000 }]
		},
		categoryTimeSeries: {
			stock: [{ date: '2024-01', value: 40000, contributions: 35000 }],
			bond: [{ date: '2024-01', value: 10000, contributions: 9500 }]
		},
		ppkStats: [
			{
				owner_user_id: 1,
				total_value: 20000,
				employee_contributed: 8000,
				employer_contributed: 6000,
				government_contributed: 1000,
				total_contributed: 15000,
				returns: 5000,
				roi_percentage: 33.33
			}
		],
		stockStats: {
			total_value: 40000,
			total_contributed: 35000,
			returns: 5000,
			roi_percentage: 14.29
		},
		bondStats: {
			total_value: 10000,
			total_contributed: 9500,
			returns: 500,
			roi_percentage: 5.26
		},
		owners: [{ id: 1, name: 'Marcin' }],
		realYieldAccounts: [
			{
				id: 1,
				name: 'Konto oszczędnościowe',
				category: 'saving_account',
				account_wrapper: null,
				interest_rate_pct: 5,
				cpi_yoy_pct: 3,
				real_yield_pct: 1.05
			}
		],
		cpiSeries: {
			points: [
				{ year: 2023, yoy_rate: 11, cumulative_index: 111 },
				{ year: 2024, yoy_rate: 3, cumulative_index: 114.3 }
			],
			base_year: 2022,
			latest_year: 2024,
			source: 'GUS'
		},
		range: 'all' as const,
		dateFrom: null,
		dateTo: null
	};

	it('renders the rebalancing suggestions when present', () => {
		render(Page, { props: { data: populatedData } });
		expect(screen.getByText('KUP')).toBeTruthy();
		expect(screen.getByText('stock')).toBeTruthy();
		expect(screen.queryByText(/Portfel jest zgodny z docelową alokacją/)).toBeNull();
	});

	it('renders the optional metric cards when values are present', () => {
		render(Page, { props: { data: populatedData } });
		expect(screen.getByText('Ile oszczędzamy miesięcznie')).toBeTruthy();
		expect(screen.getByText('Stosunek długu do dochodu')).toBeTruthy();
		expect(screen.getByText('Koszt godziny pracy')).toBeTruthy();
		expect(screen.getByText('Koszt godziny życia')).toBeTruthy();
	});

	it('renders the PPK, stock and bond summary sections', () => {
		render(Page, { props: { data: populatedData } });
		expect(screen.getByRole('heading', { name: 'Podsumowanie PPK' })).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Marcin', level: 3 })).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Podsumowanie Akcji' })).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Podsumowanie Obligacji' })).toBeTruthy();
	});
});

// jsonResponse fakes the slice of the Fetch Response that the loader reads:
// `ok`, `status`, and `json()`. `ok` derives from status like the platform.
function jsonResponse(body: unknown, status = 200): Response {
	return {
		ok: status >= 200 && status < 300,
		status,
		json: async () => body
	} as Response;
}

// routedFetch returns a fetch stub that answers each endpoint from `routes`,
// recording call order so concurrency can be asserted. A route value may be a
// Response or a thrown error (to simulate a network failure).
function routedFetch(routes: Record<string, Response | Error>) {
	const calls: string[] = [];
	const fetchFn = vi.fn(async (input: string | URL | Request) => {
		const url = typeof input === 'string' ? input : input.toString();
		calls.push(url);
		const key = Object.keys(routes).find((k) => url.includes(k));
		const route = key ? routes[key] : undefined;
		if (route instanceof Error) throw route;
		if (route) return route;
		return jsonResponse({}, 404);
	});
	return { fetchFn, calls };
}

const loadEvent = (fetchFn: typeof fetch) =>
	({
		fetch: fetchFn,
		url: new URL('http://localhost/metryki')
	}) as unknown as Parameters<typeof load>[0];

describe('metryki load', () => {
	const dashboardBody = {
		metric_cards: metricCards,
		allocation_analysis: {
			by_category: [],
			by_wrapper: [],
			rebalancing: [],
			total_investment_value: 0
		},
		investment_time_series: [],
		wrapper_time_series: { ike: [], ikze: [], ppk: [] },
		category_time_series: { stock: [], bond: [] }
	};

	const happyRoutes = (): Record<string, Response | Error> => ({
		'/api/dashboard': jsonResponse(dashboardBody),
		'/api/retirement/ppk-stats': jsonResponse([{ owner_user_id: 1 }]),
		'/api/investment/stock-stats': jsonResponse({ total_value: 1 }),
		'/api/investment/bond-stats': jsonResponse({ total_value: 2 }),
		'/api/users': jsonResponse([{ id: 1, name: 'Marcin' }]),
		'/api/accounts': jsonResponse({
			assets: [
				{ id: 1, interest_rate_pct: 5 },
				{ id: 2, interest_rate_pct: null }
			]
		}),
		'/api/cpi/series': jsonResponse({
			points: [{ year: 2024 }],
			base_year: 2022,
			latest_year: 2024,
			source: 'GUS'
		})
	});

	it('maps every endpoint into the returned data shape', async () => {
		const { fetchFn } = routedFetch(happyRoutes());
		const result = await load(loadEvent(fetchFn as unknown as typeof fetch));

		expect(result.metricCards).toEqual(metricCards);
		expect(result.ppkStats).toEqual([{ owner_user_id: 1 }]);
		expect(result.stockStats).toEqual({ total_value: 1 });
		expect(result.bondStats).toEqual({ total_value: 2 });
		expect(result.owners).toEqual([{ id: 1, name: 'Marcin' }]);
		// accounts without interest_rate_pct are filtered out
		expect(result.realYieldAccounts).toEqual([{ id: 1, interest_rate_pct: 5 }]);
		expect(result.cpiSeries.source).toBe('GUS');
	});

	it('dispatches all seven requests before any response resolves', async () => {
		// Gate every response behind a manually-released promise. A serial
		// loader would dispatch request N+1 only after response N resolved, so
		// it would have issued just one fetch while the gate is shut. The
		// concurrent loader fires all seven up front — assert that before
		// releasing the gate.
		const { fetchFn } = routedFetch(happyRoutes());
		const respond = fetchFn.getMockImplementation();
		if (!respond) throw new Error('expected a mock implementation');
		let release!: () => void;
		const gate = new Promise<void>((resolve) => (release = resolve));
		// Count dispatches in the wrapper, BEFORE the gate, so the tally reflects
		// requests issued — not responses delivered (which the gate withholds).
		let dispatched = 0;
		fetchFn.mockImplementation((input: string | URL | Request) => {
			dispatched += 1;
			return gate.then(() => respond(input));
		});

		const pending = load(loadEvent(fetchFn as unknown as typeof fetch));
		// Flush the synchronous dispatch microtasks without resolving the gate.
		await Promise.resolve();
		await Promise.resolve();
		expect(dispatched).toBe(7);

		release();
		await pending;
	});

	it('degrades every best-effort source to its default without failing the page', async () => {
		const routes = happyRoutes();
		routes['/api/retirement/ppk-stats'] = jsonResponse(null, 500);
		routes['/api/investment/stock-stats'] = new Error('network down');
		routes['/api/investment/bond-stats'] = jsonResponse(null, 500);
		routes['/api/users'] = new Error('network down');
		routes['/api/accounts'] = new Error('network down');
		routes['/api/cpi/series'] = jsonResponse(null, 503);

		const { fetchFn } = routedFetch(routes);
		const result = await load(loadEvent(fetchFn as unknown as typeof fetch));

		expect(result.ppkStats).toEqual([]);
		expect(result.stockStats).toBeNull();
		expect(result.bondStats).toBeNull();
		expect(result.owners).toEqual([]);
		expect(result.realYieldAccounts).toEqual([]);
		expect(result.cpiSeries).toEqual({
			points: [],
			base_year: null,
			latest_year: null,
			source: ''
		});
		// the dashboard still loaded fine
		expect(result.metricCards).toEqual(metricCards);
	});

	it('throws the dashboard status when the dashboard request fails', async () => {
		const routes = happyRoutes();
		routes['/api/dashboard'] = jsonResponse(null, 502);

		const { fetchFn } = routedFetch(routes);
		await expect(load(loadEvent(fetchFn as unknown as typeof fetch))).rejects.toMatchObject({
			status: 502
		});
	});
});
