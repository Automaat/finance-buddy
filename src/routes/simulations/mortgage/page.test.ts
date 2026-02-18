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

vi.mock('$app/environment', () => ({ browser: false }));

const mockChartInstance = {
	setOption: vi.fn(),
	resize: vi.fn(),
	dispose: vi.fn()
};

vi.mock('echarts', () => ({
	init: vi.fn(() => mockChartInstance)
}));

const mockResults = {
	yearly_projections: [
		{
			year: 1,
			annual_rate: 6.5,
			scenario_a_mortgage_balance: 290000,
			scenario_a_real_mortgage_balance: 281553,
			scenario_a_cumulative_interest: 19000,
			scenario_a_investment_balance: 0,
			scenario_a_after_tax_portfolio: 0,
			scenario_a_real_portfolio: 0,
			scenario_a_paid_off: false,
			scenario_b_mortgage_balance: 295000,
			scenario_b_real_mortgage_balance: 286408,
			scenario_b_investment_balance: 15000,
			scenario_b_after_tax_portfolio: 14500,
			scenario_b_real_portfolio: 14078,
			scenario_b_cumulative_interest: 19500,
			net_advantage_invest: -200
		}
	],
	summary: {
		regular_monthly_payment: 2237.5,
		total_interest_a: 236999,
		total_interest_b: 256000,
		interest_saved: 19001,
		final_investment_portfolio: 180000,
		belka_tax_a: 5000,
		belka_tax_b: 12000,
		final_portfolio_a_real: 60000,
		final_portfolio_b_real: 75000,
		months_saved: 24,
		winning_strategy: 'inwestycja',
		net_advantage: 15000
	}
};

describe('Mortgage vs Invest Page', () => {
	beforeEach(() => {
		vi.resetAllMocks();
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('renders the main heading', () => {
		render(Page);
		expect(screen.getByRole('heading', { name: /Hipoteka vs Inwestycja/i })).toBeTruthy();
	});

	it('renders the form with parameter inputs', () => {
		render(Page);
		expect(screen.getByText(/Kwota pozostała do spłaty/i)).toBeTruthy();
		expect(screen.getByText(/Oprocentowanie/i)).toBeTruthy();
		expect(screen.getByText(/Miesięczny budżet/i)).toBeTruthy();
	});

	it('renders the calculate button', () => {
		render(Page);
		expect(screen.getByRole('button', { name: /Oblicz/i })).toBeTruthy();
	});

	it('shows results after successful simulation', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: true,
				json: async () => mockResults
			})
		);

		render(Page);
		const button = screen.getByRole('button', { name: /Oblicz/i });
		await fireEvent.click(button);

		await waitFor(() => {
			expect(screen.getByText(/Wyniki/i)).toBeTruthy();
		});
		expect(screen.getByText(/Rata bazowa/i)).toBeTruthy();
		expect(screen.getByText(/Inwestycja wygrywa/i)).toBeTruthy();
	});

	it('shows error message on API failure', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: false,
				statusText: 'Bad Request',
				json: async () => ({ detail: 'Budget too low' })
			})
		);

		render(Page);
		const button = screen.getByRole('button', { name: /Oblicz/i });
		await fireEvent.click(button);

		await waitFor(() => {
			expect(screen.getByText('Budget too low')).toBeTruthy();
		});
	});

	it('shows nadplata wins label when strategy is nadplata', async () => {
		const nadplataResults = {
			...mockResults,
			summary: { ...mockResults.summary, winning_strategy: 'nadpłata' }
		};
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: true,
				json: async () => nadplataResults
			})
		);

		render(Page);
		await fireEvent.click(screen.getByRole('button', { name: /Oblicz/i }));

		await waitFor(() => {
			expect(screen.getByText(/Nadpłata wygrywa/i)).toBeTruthy();
		});
	});
});
