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

function scenarioFixture(id: number, name: string) {
	return {
		id,
		name,
		kind: 'retirement',
		created_at: '2026-01-01T00:00:00',
		updated_at: '2026-01-01T00:00:00',
		inputs_json: {
			current_age: 35,
			retirement_age: 65,
			ike_ikze_accounts: [],
			ppk_accounts: [],
			brokerage_accounts: [],
			annual_return_rate: 7.0,
			limit_growth_rate: 5.0,
			expected_salary_growth: 3.0,
			inflation_rate: 3.0
		}
	};
}

describe('Compare scenarios page', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});
	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('renders the empty-state message when no scenarios are saved', async () => {
		const fetchMock = vi.fn().mockImplementation((url: string) => {
			if (url.includes('/api/scenarios')) {
				return Promise.resolve({ ok: true, json: async () => ({ scenarios: [] }) });
			}
			return Promise.resolve({
				ok: true,
				json: async () => ({ monthly_expenses: 5000, withdrawal_rate: 0.04 })
			});
		});
		vi.stubGlobal('fetch', fetchMock);
		render(Page);
		await waitFor(() => expect(screen.getByText(/Brak zapisanych scenariuszy/)).toBeTruthy());
	});

	it('still renders the table when one scenario sim fails (partial-success)', async () => {
		let callCount = 0;
		const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
			if (url.includes('/api/scenarios')) {
				return Promise.resolve({
					ok: true,
					json: async () => ({
						scenarios: [
							scenarioFixture(1, 'Plan A'),
							scenarioFixture(2, 'Plan B'),
							scenarioFixture(3, 'Plan C')
						]
					})
				});
			}
			if (url.includes('/api/config')) {
				return Promise.resolve({
					ok: true,
					json: async () => ({ monthly_expenses: 5000, withdrawal_rate: 0.04 })
				});
			}
			if (url.includes('/api/simulations/retirement') && init?.method === 'POST') {
				callCount += 1;
				// Second sim POST fails; the other two succeed.
				if (callCount === 2) {
					return Promise.resolve({
						ok: false,
						status: 500,
						json: async () => ({ detail: 'boom' })
					});
				}
				return Promise.resolve({
					ok: true,
					json: async () => ({
						summary: {
							total_final_balance: 1_000_000,
							total_contributions: 500_000,
							total_returns: 500_000,
							total_tax_savings: 0,
							estimated_monthly_income: 3000,
							estimated_monthly_income_today: 1500,
							years_until_retirement: 30
						}
					})
				});
			}
			return Promise.resolve({ ok: false, status: 404, json: async () => ({}) });
		});
		vi.stubGlobal('fetch', fetchMock);
		render(Page);

		await waitFor(() => expect(screen.getByText('Plan A')).toBeTruthy());
		const checkboxes = screen.getAllByRole('checkbox');
		await fireEvent.click(checkboxes[0]);
		await fireEvent.click(checkboxes[1]);
		await fireEvent.click(checkboxes[2]);

		const button = screen.getByRole('button', { name: /Porównaj zaznaczone/ });
		await fireEvent.click(button);

		// Results table still rendered (≥2 successful scenarios).
		await waitFor(() => expect(screen.getByText('Wyniki porównania')).toBeTruthy());
		// Error banner mentions the failed scenario name (Plan B was call 2).
		expect(screen.getByText(/Plan B: HTTP 500/)).toBeTruthy();
	});

	it('compares two selected scenarios and renders the results table', async () => {
		const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
			if (url.includes('/api/scenarios')) {
				return Promise.resolve({
					ok: true,
					json: async () => ({
						scenarios: [scenarioFixture(1, 'Plan A'), scenarioFixture(2, 'Plan B')]
					})
				});
			}
			if (url.includes('/api/config')) {
				return Promise.resolve({
					ok: true,
					json: async () => ({ monthly_expenses: 5000, withdrawal_rate: 0.04 })
				});
			}
			if (url.includes('/api/simulations/retirement') && init?.method === 'POST') {
				return Promise.resolve({
					ok: true,
					json: async () => ({
						summary: {
							total_final_balance: 1_000_000,
							total_contributions: 500_000,
							total_returns: 500_000,
							total_tax_savings: 0,
							estimated_monthly_income: 3000,
							estimated_monthly_income_today: 1500,
							years_until_retirement: 30
						}
					})
				});
			}
			return Promise.resolve({ ok: false, status: 404, json: async () => ({}) });
		});
		vi.stubGlobal('fetch', fetchMock);
		render(Page);

		// Wait for scenarios to load + checkboxes to render.
		await waitFor(() => expect(screen.getByText('Plan A')).toBeTruthy());

		const checkboxes = screen.getAllByRole('checkbox');
		await fireEvent.click(checkboxes[0]);
		await fireEvent.click(checkboxes[1]);

		const button = screen.getByRole('button', { name: /Porównaj zaznaczone/ });
		await fireEvent.click(button);

		await waitFor(() => expect(screen.getByText('Wyniki porównania')).toBeTruthy());
		// Two retirement POSTs fired.
		const simCalls = fetchMock.mock.calls.filter(([u]) =>
			String(u).includes('/api/simulations/retirement')
		);
		expect(simCalls).toHaveLength(2);
	});
});
