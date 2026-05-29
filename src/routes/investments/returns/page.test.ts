import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('$env/dynamic/public', () => ({
	env: {
		PUBLIC_API_URL_BROWSER: 'http://localhost:8000',
		PUBLIC_API_URL: 'http://localhost:8000'
	}
}));

import Page from './+page.svelte';
import type { ReturnsAccountOption } from './+page.ts';

const returnsPayload = {
	scope: { type: 'all' },
	period: 'all',
	since: null,
	as_of: '2026-05-24',
	deposits: 10000,
	withdrawals: 0,
	net_contributed: 10000,
	current_value: 12000,
	valuation_change: 2000,
	simple_roi_pct: 20,
	money_weighted_pct: 15.5,
	has_snapshot: true,
	convergence_failed: false
};

const accounts: ReturnsAccountOption[] = [
	{ id: 1, name: 'XTB IKE', category: 'stock', account_wrapper: 'IKE', is_active: true },
	{ id: 2, name: 'mBank ETF', category: 'etf', account_wrapper: null, is_active: true }
];

describe('Returns page', () => {
	let fetchSpy: ReturnType<typeof vi.spyOn>;

	beforeEach(() => {
		fetchSpy = vi
			.spyOn(globalThis, 'fetch')
			.mockImplementation(async () => new Response(JSON.stringify(returnsPayload)));
	});

	afterEach(() => {
		fetchSpy.mockRestore();
	});

	it('renders the scope preset buttons and the returns card', async () => {
		render(Page, { props: { data: { user: null, accounts } } });
		expect(screen.getByRole('button', { name: 'Całość' })).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Akcje' })).toBeTruthy();
		expect(screen.getByRole('button', { name: 'IKE' })).toBeTruthy();
		await waitFor(() => expect(screen.getByText('Zwrot (XIRR)')).toBeTruthy());
	});

	it('marks the selected preset as pressed and switches scope', async () => {
		render(Page, { props: { data: { user: null, accounts } } });
		const akcje = screen.getByRole('button', { name: 'Akcje' });
		await fireEvent.click(akcje);
		expect(akcje.getAttribute('aria-pressed')).toBe('true');
		// fetch should be called with scope=category&value=stock
		await waitFor(() => {
			const calls = fetchSpy.mock.calls.map((c: unknown[]) => String(c[0]));
			expect(
				calls.some((u: string) => u.includes('scope=category') && u.includes('value=stock'))
			).toBe(true);
		});
	});

	it('lists investment accounts in the per-account selector', () => {
		render(Page, { props: { data: { user: null, accounts } } });
		expect(screen.getByRole('option', { name: 'XTB IKE' })).toBeTruthy();
		expect(screen.getByRole('option', { name: 'mBank ETF' })).toBeTruthy();
	});

	it('hides the account selector when there are no investment accounts', () => {
		render(Page, { props: { data: { user: null, accounts: [] } } });
		expect(screen.queryByText('Pojedyncze konto…')).toBeNull();
	});
});
