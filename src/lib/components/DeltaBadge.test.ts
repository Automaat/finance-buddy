import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import DeltaBadge from './DeltaBadge.svelte';

describe('DeltaBadge', () => {
	it('applies success color for positive absolute value', () => {
		render(DeltaBadge, {
			props: {
				label: 'MoM',
				absolute: 1000,
				percentage: 20,
				formulaTitle: 'test formula'
			}
		});
		const badge = screen.getByLabelText('Δ MoM');
		expect(badge.className).toContain('text-success-600-400');
	});

	it('shows signed PLN and percent for positive value', () => {
		render(DeltaBadge, {
			props: {
				label: 'MoM',
				absolute: 1000,
				percentage: 20,
				formulaTitle: 'test formula'
			}
		});
		const badge = screen.getByLabelText('Δ MoM');
		expect(badge.textContent).toMatch(/\+/);
	});

	it('applies error color for negative absolute value', () => {
		render(DeltaBadge, {
			props: {
				label: 'YoY',
				absolute: -500,
				percentage: -10,
				formulaTitle: 'test formula'
			}
		});
		const badge = screen.getByLabelText('Δ YoY');
		expect(badge.className).toContain('text-error-600-400');
	});

	it('shows − prefix for negative value', () => {
		render(DeltaBadge, {
			props: {
				label: 'YoY',
				absolute: -500,
				percentage: -10,
				formulaTitle: 'test formula'
			}
		});
		const badge = screen.getByLabelText('Δ YoY');
		expect(badge.textContent).toContain('−');
	});

	it('applies neutral color for zero value', () => {
		render(DeltaBadge, {
			props: {
				label: 'MoM',
				absolute: 0,
				percentage: 0,
				formulaTitle: 'test formula'
			}
		});
		const badge = screen.getByLabelText('Δ MoM');
		expect(badge.className).toContain('text-surface-700-300');
	});

	it('renders em-dash and no trend icon when absolute is null', () => {
		render(DeltaBadge, {
			props: {
				label: 'YoY',
				absolute: null,
				percentage: null,
				formulaTitle: 'test formula'
			}
		});
		const badge = screen.getByLabelText('Δ YoY');
		expect(badge.textContent).toContain('—');
		expect(badge.className).toContain('text-surface-700-300');
	});

	it('sets title attribute from formulaTitle prop', () => {
		render(DeltaBadge, {
			props: {
				label: 'MoM',
				absolute: 500,
				percentage: 10,
				formulaTitle: 'custom formula text'
			}
		});
		const badge = screen.getByLabelText('Δ MoM');
		expect(badge.getAttribute('title')).toBe('custom formula text');
	});
});
