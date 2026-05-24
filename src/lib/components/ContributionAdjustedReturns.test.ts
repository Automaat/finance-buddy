import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte';

vi.mock('$env/dynamic/public', () => ({
	env: {
		PUBLIC_API_URL_BROWSER: 'http://localhost:8000',
		PUBLIC_API_URL: 'http://localhost:8000'
	}
}));

import ContributionAdjustedReturns from './ContributionAdjustedReturns.svelte';

const successPayload = {
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

describe('ContributionAdjustedReturns', () => {
	let fetchSpy: ReturnType<typeof vi.spyOn>;

	beforeEach(() => {
		fetchSpy = vi
			.spyOn(globalThis, 'fetch')
			.mockImplementation(async () => new Response(JSON.stringify(successPayload)));
	});

	afterEach(() => {
		fetchSpy.mockRestore();
	});

	it('renders metric cards after fetch', async () => {
		render(ContributionAdjustedReturns, { props: { scope: { type: 'all' } } });
		await waitFor(() => {
			expect(screen.getByText('Wpłacono netto')).toBeTruthy();
			expect(screen.getByText('Wartość obecna')).toBeTruthy();
			expect(screen.getByText('Zwrot (XIRR)')).toBeTruthy();
		});
	});

	it('renders snapshot-missing copy when has_snapshot=false', async () => {
		fetchSpy.mockImplementation(
			async () => new Response(JSON.stringify({ ...successPayload, has_snapshot: false }))
		);
		render(ContributionAdjustedReturns, { props: { scope: { type: 'all' } } });
		await waitFor(() => {
			expect(screen.getByText(/Brak snapshotów/)).toBeTruthy();
		});
	});

	it('re-fetches when period changes', async () => {
		render(ContributionAdjustedReturns, { props: { scope: { type: 'all' } } });
		await waitFor(() => expect(fetchSpy).toHaveBeenCalled());
		fetchSpy.mockClear();
		await fireEvent.click(screen.getByRole('tab', { name: '1M' }));
		await waitFor(() => expect(fetchSpy).toHaveBeenCalled());
		const lastCall = fetchSpy.mock.calls[fetchSpy.mock.calls.length - 1][0] as string;
		expect(lastCall).toContain('period=1m');
	});

	it('shows convergence-failed copy when API reports it', async () => {
		fetchSpy.mockImplementation(
			async () =>
				new Response(
					JSON.stringify({
						...successPayload,
						money_weighted_pct: null,
						convergence_failed: true
					})
				)
		);
		render(ContributionAdjustedReturns, { props: { scope: { type: 'all' } } });
		await waitFor(() => {
			expect(screen.getByText('Nie udało się obliczyć')).toBeTruthy();
		});
	});
});
