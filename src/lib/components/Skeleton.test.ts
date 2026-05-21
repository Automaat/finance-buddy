import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import Skeleton from './Skeleton.svelte';

describe('Skeleton', () => {
	it('renders with role="status" and aria-busy="true"', () => {
		render(Skeleton);
		const el = screen.getByRole('status');
		expect(el.getAttribute('aria-busy')).toBe('true');
		expect(el.getAttribute('aria-live')).toBe('polite');
	});

	it('uses the default Polish aria-label when none is provided', () => {
		render(Skeleton);
		const el = screen.getByRole('status');
		expect(el.getAttribute('aria-label')).toBe('Ładowanie');
	});

	it('forwards a custom aria-label', () => {
		render(Skeleton, { props: { 'aria-label': 'Ładowanie wykresu' } });
		const el = screen.getByRole('status', { name: 'Ładowanie wykresu' });
		expect(el).toBeTruthy();
	});

	it('applies width and height inline styles', () => {
		render(Skeleton, { props: { width: '50%', height: '2rem' } });
		const el = screen.getByRole('status');
		expect(el.style.width).toBe('50%');
		expect(el.style.height).toBe('2rem');
	});

	it('applies the rounded variant class', () => {
		render(Skeleton, { props: { rounded: 'full' } });
		const el = screen.getByRole('status');
		expect(el.className).toContain('rounded-full');
	});

	it('merges custom class names', () => {
		render(Skeleton, { props: { class: 'mt-2 custom-thing' } });
		const el = screen.getByRole('status');
		expect(el.className).toContain('mt-2');
		expect(el.className).toContain('custom-thing');
	});
});
