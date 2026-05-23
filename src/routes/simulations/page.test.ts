import { describe, expect, it, beforeAll, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import Page from './+page.svelte';

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

vi.mock('$app/environment', () => ({ browser: true }));

vi.mock('echarts', () => ({
	init: vi.fn(() => ({ setOption: vi.fn(), resize: vi.fn(), dispose: vi.fn(), clear: vi.fn() }))
}));

const mockData = {
	current_age: 35,
	retirement_age: 65,
	owners: [
		{ id: 1, name: 'Marcin' },
		{ id: 2, name: 'Ewa' }
	],
	balances: { ike_1: 1000, ikze_1: 2000, ike_2: 500, ikze_2: 800 },
	ppk_balances: { '1': 5000, '2': 3000 },
	monthly_salaries: { '1': 15000, '2': 12000 },
	ppk_rates: { '1': { employee: 2.5, employer: 1.5 } }
};

describe('Simulations Page — render', () => {
	it('renders the main heading', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: 'Symulacje Emerytalne' })).toBeTruthy();
	});

	it('renders the simulation parameters section', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: 'Parametry symulacji' })).toBeTruthy();
	});

	it('renders the accounts-to-simulate section', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: 'Konta do symulacji' })).toBeTruthy();
	});

	it('does not render results before a simulation is run', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.queryByRole('heading', { name: 'Wyniki symulacji' })).toBeNull();
	});

	it('renders one IKE + IKZE checkbox per owner', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByText('IKE (Marcin)')).toBeTruthy();
		expect(screen.getByText('IKZE (Marcin)')).toBeTruthy();
		expect(screen.getByText('IKE (Ewa)')).toBeTruthy();
		expect(screen.getByText('IKZE (Ewa)')).toBeTruthy();
	});
});

describe('Simulations Page — run simulation', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('POSTs request body and renders the results section on success', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			statusText: 'OK',
			json: async () => ({
				simulations: [
					{
						account_name: 'IKE Marcin',
						starting_balance: 1000,
						total_contributions: 5000,
						total_returns: 700,
						total_tax_savings: 0,
						final_balance: 6700,
						yearly_projections: [
							{
								year: 2026,
								age: 35,
								annual_contribution: 1000,
								balance_end_of_year: 2000,
								cumulative_contributions: 1000,
								cumulative_returns: 0,
								annual_limit: 5000,
								limit_utilized_pct: 20,
								tax_savings: 0
							}
						]
					}
				],
				summary: {
					total_final_balance: 6700,
					total_contributions: 5000,
					total_returns: 700,
					total_tax_savings: 0,
					estimated_monthly_income: 200,
					estimated_monthly_income_today: 150,
					years_until_retirement: 30
				}
			})
		});
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: mockData } });
		await fireEvent.click(screen.getByRole('button', { name: 'Uruchom symulację' }));

		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://localhost:8000/api/simulations/retirement');
		expect((init as RequestInit).method).toBe('POST');
		const body = JSON.parse((init as RequestInit).body as string);
		expect(body.current_age).toBe(35);
		expect(body.retirement_age).toBe(65);
		expect(Array.isArray(body.ike_ikze_accounts)).toBe(true);

		await waitFor(() =>
			expect(screen.getByRole('heading', { name: 'Wyniki symulacji' })).toBeTruthy()
		);
	});

	it('surfaces non-2xx response as an error', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: false,
			statusText: 'Internal Server Error',
			json: async () => ({})
		});
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: mockData } });
		await fireEvent.click(screen.getByRole('button', { name: 'Uruchom symulację' }));

		await waitFor(() => expect(screen.getByText(/Simulation failed/)).toBeTruthy());
		expect(screen.queryByRole('heading', { name: 'Wyniki symulacji' })).toBeNull();
	});
});
