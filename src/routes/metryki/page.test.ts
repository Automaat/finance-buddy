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
	ikzePitStats: [],
	snapshotDate: null,
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

	it('keeps optional metric cards visible with a config hint when null', () => {
		render(Page, { props: { data: mockData } });
		// Labels stay on the page instead of vanishing…
		expect(screen.getByText('Stosunek długu do dochodu')).toBeTruthy();
		expect(screen.getByText('Koszt godziny pracy')).toBeTruthy();
		expect(screen.getByText('Koszt godziny życia')).toBeTruthy();
		// …and point the user at where to fill the gap.
		const hints = screen.getAllByRole('link', { name: 'Uzupełnij konfigurację' });
		expect(hints.length).toBeGreaterThan(0);
		expect(hints[0].getAttribute('href')).toBe('/settings/config');
		// savings_rate has its own bespoke hint pointing at /salaries.
		const savingsHint = screen.getByRole('link', { name: 'Dodaj pensje i snapshoty, by policzyć' });
		expect(savingsHint.getAttribute('href')).toBe('/salaries');
	});

	it('omits the mortgage payoff line when there is nothing left to pay', () => {
		render(Page, { props: { data: mockData } });
		// mockData has mortgage_months_left: 0 → no "X mies. … do spłaty" line.
		// (The card's tooltip mentions "do spłaty"; the secondary line is the
		// one that also carries "mies.".)
		expect(screen.queryByText(/mies\..*do spłaty/)).toBeNull();
	});

	it('does not show a snapshot date when none is available', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.queryByText(/stan na/)).toBeNull();
	});

	it('omits PPK, stock and bond summary sections when no data', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.queryByRole('heading', { name: 'Podsumowanie PPK' })).toBeNull();
		expect(screen.queryByRole('heading', { name: 'Podsumowanie Akcji' })).toBeNull();
		expect(screen.queryByRole('heading', { name: 'Podsumowanie Obligacji' })).toBeNull();
	});

	it('omits the IKZE tax-benefit section when there are no PIT stats', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.queryByRole('heading', { name: /Korzyść podatkowa IKZE/ })).toBeNull();
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
		ikzePitStats: [
			{
				year: 2024,
				account_wrapper: 'IKZE',
				owner_user_id: 1,
				total_contributed: 9000,
				marginal_tax_rate: 0.32,
				pit_savings: 2880
			}
		],
		snapshotDate: '2026-05-24',
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
		// values present → no config hint links
		expect(screen.queryByRole('link', { name: 'Uzupełnij konfigurację' })).toBeNull();
	});

	it('shows the snapshot date and one consolidated mortgage card', () => {
		render(Page, { props: { data: populatedData } });
		expect(screen.getByText('stan na 2026-05-24')).toBeTruthy();
		// the three legacy mortgage cards collapse into one with a payoff line
		expect(screen.getByText('Hipoteka do spłaty')).toBeTruthy();
		expect(screen.queryByText('Ile miesięcy do spłaty hipoteki')).toBeNull();
		expect(screen.queryByText('Ile lat do spłaty hipoteki')).toBeNull();
		expect(screen.getByText(/240 mies\..*do spłaty/)).toBeTruthy();
	});

	it('renders the PPK, stock and bond summary sections', () => {
		render(Page, { props: { data: populatedData } });
		expect(screen.getByRole('heading', { name: 'Podsumowanie PPK' })).toBeTruthy();
		// "Marcin" heads both the PPK and IKZE owner sub-sections.
		expect(screen.getAllByRole('heading', { name: 'Marcin', level: 3 }).length).toBeGreaterThan(0);
		expect(screen.getByRole('heading', { name: 'Podsumowanie Akcji' })).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Podsumowanie Obligacji' })).toBeTruthy();
	});

	it('renders the IKZE tax-benefit section with its estimated PIT saving', () => {
		render(Page, { props: { data: populatedData } });
		expect(screen.getByRole('heading', { name: /Korzyść podatkowa IKZE/ })).toBeTruthy();
		expect(screen.getByText('IKZE - Szacowana ulga PIT')).toBeTruthy();
		// pit_savings 2880 → pl-PL formats with a narrow no-break space; match on
		// the digits regardless of which whitespace separates the thousands.
		expect(screen.getByText((text) => /2\s*880\s*PLN/.test(text))).toBeTruthy();
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
		}),
		'/api/retirement/stats': jsonResponse([
			{
				year: 2024,
				account_wrapper: 'IKE',
				owner_user_id: 1,
				total_contributed: 5000,
				marginal_tax_rate: null,
				pit_savings: null
			},
			{
				year: 2024,
				account_wrapper: 'IKZE',
				owner_user_id: 1,
				total_contributed: 9000,
				marginal_tax_rate: 0.32,
				pit_savings: 2880
			},
			{
				year: 2024,
				account_wrapper: 'IKZE',
				owner_user_id: 2,
				total_contributed: 0,
				marginal_tax_rate: null,
				pit_savings: null
			}
		]),
		'/api/snapshots': jsonResponse([
			{ id: 9, date: '2026-05-24' },
			{ id: 8, date: '2026-04-24' }
		])
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
		// only IKZE rows with a non-null pit_savings survive the filter
		expect(result.ikzePitStats).toEqual([
			{
				year: 2024,
				account_wrapper: 'IKZE',
				owner_user_id: 1,
				total_contributed: 9000,
				marginal_tax_rate: 0.32,
				pit_savings: 2880
			}
		]);
		// the most recent snapshot's date drives the "stan na" label
		expect(result.snapshotDate).toBe('2026-05-24');
	});

	it('dispatches all nine requests before any response resolves', async () => {
		// Gate every response behind a manually-released promise. A serial
		// loader would dispatch request N+1 only after response N resolved, so
		// it would have issued just one fetch while the gate is shut. The
		// concurrent loader fires all nine up front — assert that before
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
		expect(dispatched).toBe(9);

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
		routes['/api/retirement/stats'] = jsonResponse(null, 500);
		routes['/api/snapshots'] = new Error('network down');

		const { fetchFn } = routedFetch(routes);
		const result = await load(loadEvent(fetchFn as unknown as typeof fetch));

		expect(result.ppkStats).toEqual([]);
		expect(result.stockStats).toBeNull();
		expect(result.bondStats).toBeNull();
		expect(result.owners).toEqual([]);
		expect(result.realYieldAccounts).toEqual([]);
		expect(result.ikzePitStats).toEqual([]);
		expect(result.snapshotDate).toBeNull();
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
