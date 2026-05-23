import { describe, expect, it, beforeAll, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
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
