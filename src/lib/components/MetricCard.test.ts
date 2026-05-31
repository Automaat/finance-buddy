import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import MetricCard from './MetricCard.svelte';

describe('MetricCard', () => {
	it('renders label and formatted value with suffix', () => {
		render(MetricCard, { props: { label: 'Wartość', value: 1234.5, decimals: 1, suffix: ' PLN' } });
		expect(screen.getByText('Wartość')).toBeTruthy();
		// pl-PL uses a comma decimal + a (non-breaking) space thousands sep;
		// match loosely on the meaningful parts rather than the exact spacing.
		expect(screen.getByText((text) => text.includes('234,5') && text.includes('PLN'))).toBeTruthy();
	});

	it('shows an em-dash for null/NaN', () => {
		render(MetricCard, { props: { label: 'Brak', value: null } });
		expect(screen.getByText('—')).toBeTruthy();
	});

	it('applies the colour class for the chosen colour', () => {
		render(MetricCard, { props: { label: 'Zysk', value: 10, color: 'green' } });
		const valueEl = screen.getByText('10');
		expect(valueEl.className).toContain('text-success-600-400');
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
});
