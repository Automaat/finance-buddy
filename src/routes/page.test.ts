import { render, screen } from '@testing-library/svelte';
import Page from './+page.svelte';

describe('Home Page', () => {
	it('renders the main heading', () => {
		render(Page);
		expect(screen.getByRole('heading', { name: /Finansowa Forteca/i })).toBeTruthy();
	});

	it('renders the description', () => {
		render(Page);
		expect(screen.getByText('Your personal finance tracker')).toBeTruthy();
	});
});
