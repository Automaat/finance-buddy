import { describe, expect, it, beforeAll, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import Page from './+page.svelte';
import type { CurrencyExposureReport } from '$lib/types/exposure';

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

vi.mock('echarts', () => ({
	init: vi.fn(() => ({ setOption: vi.fn(), resize: vi.fn(), dispose: vi.fn() }))
}));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

vi.mock('$app/stores', async () => {
	const { readable } = await import('svelte/store');
	return { page: readable({ url: new URL('http://localhost/ekspozycja') }) };
});

const report = (over: Partial<CurrencyExposureReport> = {}): CurrencyExposureReport => ({
	currencies: [
		{ currency: 'PLN', value_pln: 60000, percent: 60 },
		{ currency: 'USD', value_pln: 40000, percent: 40 }
	],
	total_pln: 100000,
	pln_pct: 60,
	foreign_pct: 40,
	snapshot_date: '2026-04-30',
	...over
});

describe('Ekspozycja page', () => {
	it('renders the heading and per-currency rows', () => {
		render(Page, {
			props: { data: { user: null, report: report(), targetPLNPct: null, tolerance: 5 } }
		});
		expect(screen.getByRole('heading', { name: 'Ekspozycja walutowa', level: 1 })).toBeTruthy();
		expect(screen.getByText('PLN')).toBeTruthy();
		expect(screen.getByText('USD')).toBeTruthy();
	});

	it('shows the empty-state when there are no currencies', () => {
		render(Page, {
			props: {
				data: { user: null, report: report({ currencies: [] }), targetPLNPct: null, tolerance: 5 }
			}
		});
		expect(screen.getByText('Brak danych o ekspozycji walutowej')).toBeTruthy();
		expect(screen.getByText(/Dodaj pierwszy snapshot lub aktywne konto/)).toBeTruthy();
	});

	it('renders the drift band when a target is set', () => {
		render(Page, {
			props: {
				data: {
					user: null,
					report: report({
						drift: {
							target_pln_pct: 50,
							actual_pln_pct: 60,
							drift_pln_pct: 10,
							within_tolerance: false,
							tolerance_pct: 5
						}
					}),
					targetPLNPct: 50,
					tolerance: 5
				}
			}
		});
		expect(screen.getByText(/Dryft względem celu 50% PLN/)).toBeTruthy();
		expect(screen.getByText('+10.0 pp')).toBeTruthy();
		expect(screen.getByText(/poza pasmem ±5 pp/)).toBeTruthy();
	});

	it('omits the drift band when no target is set', () => {
		render(Page, {
			props: { data: { user: null, report: report(), targetPLNPct: null, tolerance: 5 } }
		});
		expect(screen.queryByText(/Dryft względem celu/)).toBeNull();
	});
});
