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
});
