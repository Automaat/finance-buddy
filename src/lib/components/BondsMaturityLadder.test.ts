import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import BondsMaturityLadder from './BondsMaturityLadder.svelte';
import type { MaturityLadderEvent, NextMaturityWarning } from '../../routes/bonds/+page';

const emptyProps = {
	events: [] as MaturityLadderEvent[],
	nextMaturity: null as NextMaturityWarning | null,
	taxRatePct: 19
};

describe('BondsMaturityLadder', () => {
	it('renders empty state when no events', () => {
		render(BondsMaturityLadder, { props: emptyProps });
		expect(screen.getByText(/Brak nadchodzących przepływów/)).toBeTruthy();
	});

	it('shows tax-rate disclosure in the header', () => {
		render(BondsMaturityLadder, { props: emptyProps });
		expect(screen.getByText(/Wartości netto po podatku Belki \(19%\)/)).toBeTruthy();
	});

	function makeWarning(daysUntil: number): NextMaturityWarning {
		return {
			date: '2026-06-01',
			type: 'DOS',
			bond_ids: [1],
			count: 1,
			principal: 1000,
			interest_gross: 130,
			tax: 24.7,
			net_cashflow: 1105.3,
			days_until: daysUntil
		};
	}

	it('renders next-maturity warning with urgent tier within 30 days', () => {
		const { container } = render(BondsMaturityLadder, {
			props: { ...emptyProps, nextMaturity: makeWarning(6) }
		});
		expect(screen.getByText(/Najbliższy wykup/)).toBeTruthy();
		expect(screen.getByText(/6 dni/)).toBeTruthy();
		expect(container.querySelector('.preset-filled-error-500')).toBeTruthy();
	});

	it('uses warn tier between 31 and 90 days', () => {
		const { container } = render(BondsMaturityLadder, {
			props: { ...emptyProps, nextMaturity: makeWarning(50) }
		});
		expect(container.querySelector('.preset-filled-warning-500')).toBeTruthy();
		expect(container.querySelector('.preset-filled-error-500')).toBeNull();
	});

	it('uses info tier beyond 90 days', () => {
		const { container } = render(BondsMaturityLadder, {
			props: { ...emptyProps, nextMaturity: makeWarning(180) }
		});
		expect(container.querySelector('.preset-tonal-primary')).toBeTruthy();
		expect(container.querySelector('.preset-filled-warning-500')).toBeNull();
		expect(container.querySelector('.preset-filled-error-500')).toBeNull();
	});

	it('groups events by month and shows monthly net total', () => {
		const events: MaturityLadderEvent[] = [
			{
				month: '2026-09-01',
				type: 'COI',
				kind: 'coupon',
				bond_ids: [1],
				count: 1,
				principal: 0,
				interest_gross: 78,
				tax: 14.82,
				net_cashflow: 63.18
			},
			{
				month: '2026-09-01',
				type: 'EDO',
				kind: 'redemption',
				bond_ids: [2],
				count: 1,
				principal: 1000,
				interest_gross: 200,
				tax: 38,
				net_cashflow: 1162
			},
			{
				month: '2027-01-01',
				type: 'COI',
				kind: 'coupon',
				bond_ids: [1],
				count: 1,
				principal: 0,
				interest_gross: 52.5,
				tax: 9.98,
				net_cashflow: 42.52
			}
		];

		render(BondsMaturityLadder, { props: { ...emptyProps, events } });

		const monthHeaders = screen.getAllByRole('heading', { level: 4 });
		expect(monthHeaders).toHaveLength(2);
		expect(screen.getAllByText('COI').length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText('EDO').length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText(/Kupon/).length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText(/Wykup/).length).toBeGreaterThanOrEqual(1);
	});
});
