import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Skeleton from './Skeleton.svelte';

describe('Skeleton', () => {
	it('renders a presentational, aria-hidden block', () => {
		const { container } = render(Skeleton);
		const el = container.querySelector('.skeleton') as HTMLElement;
		expect(el).toBeTruthy();
		expect(el.getAttribute('aria-hidden')).toBe('true');
	});

	it('applies width and height inline styles', () => {
		const { container } = render(Skeleton, { props: { width: '50%', height: '2rem' } });
		const el = container.querySelector('.skeleton') as HTMLElement;
		expect(el.style.width).toBe('50%');
		expect(el.style.height).toBe('2rem');
	});

	it('applies the rounded variant class', () => {
		const { container } = render(Skeleton, { props: { rounded: 'full' } });
		const el = container.querySelector('.skeleton') as HTMLElement;
		expect(el.className).toContain('rounded-full');
	});

	it('defaults to rounded-md', () => {
		const { container } = render(Skeleton);
		const el = container.querySelector('.skeleton') as HTMLElement;
		expect(el.className).toContain('rounded-md');
	});

	it('merges custom class names', () => {
		const { container } = render(Skeleton, { props: { class: 'mt-2 custom-thing' } });
		const el = container.querySelector('.skeleton') as HTMLElement;
		expect(el.className).toContain('mt-2');
		expect(el.className).toContain('custom-thing');
	});
});
