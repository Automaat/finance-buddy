import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { Wallet } from 'lucide-svelte';
import MetricCard from './MetricCard.svelte';

describe('MetricCard', () => {
	it('renders label and formatted value with suffix', () => {
		render(MetricCard, { props: { label: 'Wartość', value: 1234.5, decimals: 1, suffix: ' PLN' } });
		expect(screen.getByText('Wartość')).toBeTruthy();
		// pl-PL uses a comma decimal + a (non-breaking) space thousands sep;
		// match loosely on the meaningful parts rather than the exact spacing.
		expect(screen.getByText((text) => text.includes('234,5') && text.includes('PLN'))).toBeTruthy();
	});

	it('can render the label as a card heading', () => {
		render(MetricCard, { props: { label: 'Wartość Netto', labelHeadingLevel: 3, value: 1234 } });
		expect(screen.getByRole('heading', { name: 'Wartość Netto', level: 3 })).toBeTruthy();
	});

	it('shows an em-dash for null/NaN', () => {
		render(MetricCard, { props: { label: 'Brak', value: null } });
		expect(screen.getByText('—')).toBeTruthy();
	});

	it('uses neutral ink for ordinary values', () => {
		render(MetricCard, { props: { label: 'Wartość', value: 10 } });
		const valueEl = screen.getByText('10');
		expect(valueEl.className).toContain('text-surface-950-50');
	});

	it('colors only explicitly signed values', () => {
		render(MetricCard, { props: { label: 'Zysk', value: 10, signed: true } });
		const valueEl = screen.getByText('+10');
		expect(valueEl.className).toContain('text-success-600-400');
	});

	it('treats near-zero signed values that round to zero as neutral', () => {
		render(MetricCard, {
			props: { label: 'Zmiana', value: -0.004, decimals: 2, suffix: '%', signed: true }
		});
		const valueEl = screen.getByText((text) => text.includes('0,00') && text.includes('%'));
		// Must NOT show a sign prefix or error/success color
		expect(valueEl.textContent).not.toMatch(/[+−]/);
		expect(valueEl.className).toContain('text-surface-950-50');
		expect(valueEl.className).not.toContain('text-error');
		expect(valueEl.className).not.toContain('text-success');
	});

	it('exposes the tooltip as a title on the label', () => {
		render(MetricCard, { props: { label: 'Koszt', value: 5, tooltip: 'Wyjaśnienie' } });
		expect(screen.getByText('Koszt').getAttribute('title')).toBe('Wyjaśnienie');
	});

	it('renders a secondary line beneath the value', () => {
		render(MetricCard, {
			props: { label: 'Hipoteka', value: 1000, secondary: '12 mies. do spłaty' }
		});
		expect(screen.getByText('12 mies. do spłaty')).toBeTruthy();
	});

	it('renders an empty-hint link instead of an em-dash when value is null', () => {
		render(MetricCard, {
			props: {
				label: 'Brak',
				value: null,
				emptyHint: 'Uzupełnij konfigurację',
				emptyHref: '/settings/config'
			}
		});
		expect(screen.queryByText('—')).toBeNull();
		const link = screen.getByRole('link', { name: 'Uzupełnij konfigurację' });
		expect(link.getAttribute('href')).toBe('/settings/config');
	});

	it('falls back to the em-dash when value is null and no hint is given', () => {
		render(MetricCard, { props: { label: 'Brak', value: null } });
		expect(screen.getByText('—')).toBeTruthy();
	});

	it('renders valueText verbatim and ignores numeric formatting props', () => {
		render(MetricCard, {
			props: {
				label: 'Gotowe',
				value: 1234.5,
				valueText: '1 234,50 zł',
				decimals: 0,
				suffix: ' PLN'
			}
		});
		expect(screen.getByText('1 234,50 zł')).toBeTruthy();
		expect(screen.queryByText((text) => text.includes('PLN'))).toBeNull();
	});

	it('renders a valueText-only card without a numeric value', () => {
		render(MetricCard, { props: { label: 'Data', valueText: '2026-06-27' } });
		expect(screen.getByText('2026-06-27')).toBeTruthy();
		expect(screen.queryByText('—')).toBeNull();
	});

	it('renders an icon in the label row', () => {
		const { container } = render(MetricCard, {
			props: { label: 'Portfel', value: 100, icon: Wallet }
		});
		expect(screen.getByText('Portfel')).toBeTruthy();
		expect(container.querySelector('svg')).toBeTruthy();
	});

	it('renders children below the value', () => {
		render(MetricCard, {
			props: {
				label: 'Z dodatkiem',
				value: 10,
				children: createRawSnippet(() => ({
					render: () => '<span>Treść dodatkowa</span>'
				}))
			}
		});
		expect(screen.getByText('Treść dodatkowa')).toBeTruthy();
	});
});
