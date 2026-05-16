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
	bondStats: null
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
});
