import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import RealYieldsTable from './RealYieldsTable.svelte';
import type { RealYieldAccount } from './RealYieldsTable.svelte';

const account = (over: Partial<RealYieldAccount>): RealYieldAccount => ({
	id: 1,
	name: 'Konto',
	category: 'saving_account',
	account_wrapper: null,
	interest_rate_pct: 5,
	cpi_yoy_pct: 3,
	real_yield_pct: 1.05,
	...over
});

describe('RealYieldsTable', () => {
	it('renders a row per interest-bearing account', () => {
		render(RealYieldsTable, {
			props: { accounts: [account({ id: 1, name: 'Moje konto' })] }
		});
		expect(screen.getByText('Moje konto')).toBeTruthy();
		expect(screen.getByText('Moje konto').closest('tr')).toBeTruthy();
	});

	it('filters out accounts without a nominal rate', () => {
		render(RealYieldsTable, {
			props: {
				accounts: [
					account({ id: 1, name: 'Z oprocentowaniem', interest_rate_pct: 4 }),
					account({ id: 2, name: 'Bez oprocentowania', interest_rate_pct: null })
				]
			}
		});
		expect(screen.getByText('Z oprocentowaniem')).toBeTruthy();
		expect(screen.queryByText('Bez oprocentowania')).toBeNull();
	});

	it('shows the empty-state message when no rated accounts', () => {
		render(RealYieldsTable, {
			props: { accounts: [account({ interest_rate_pct: null })] }
		});
		expect(screen.getByText(/Brak kont z oprocentowaniem/)).toBeTruthy();
	});

	it('maps the account wrapper as a parenthetical', () => {
		render(RealYieldsTable, {
			props: { accounts: [account({ name: 'IKE konto', account_wrapper: 'IKE' })] }
		});
		expect(screen.getByText('(IKE)')).toBeTruthy();
	});

	it('shows the inflation drag as a single signed value (no double sign on deflation)', () => {
		render(RealYieldsTable, {
			props: { accounts: [account({ name: 'Deflacja', cpi_yoy_pct: -2 })] }
		});
		// −(−2%) = +2,0% drag (deflation boosts real return); never "−−".
		expect(screen.getByText('+2,0%')).toBeTruthy();
		expect(screen.queryByText(/−−/)).toBeNull();
	});

	it('sorts rows by real yield descending', () => {
		render(RealYieldsTable, {
			props: {
				accounts: [
					account({ id: 1, name: 'Niski', real_yield_pct: -2 }),
					account({ id: 2, name: 'Wysoki', real_yield_pct: 3 })
				]
			}
		});
		const names = screen
			.getAllByRole('row')
			.slice(1)
			.map((r) => r.querySelector('td')?.textContent?.trim());
		expect(names[0]).toContain('Wysoki');
		expect(names[1]).toContain('Niski');
	});
});
