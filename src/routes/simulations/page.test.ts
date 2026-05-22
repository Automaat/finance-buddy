import { describe, expect, it, beforeAll, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import Page from './+page.svelte';

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

vi.mock('$env/dynamic/public', () => ({
	env: {
		PUBLIC_API_URL_BROWSER: 'http://localhost:8000',
		PUBLIC_API_URL: 'http://localhost:8000'
	}
}));

vi.mock('$app/environment', () => ({ browser: false }));

vi.mock('echarts', () => ({
	init: vi.fn(() => ({ setOption: vi.fn(), resize: vi.fn(), dispose: vi.fn() }))
}));

const mockData = {
	current_age: 35,
	retirement_age: 65,
	owners: [
		{ id: 1, name: 'Marcin' },
		{ id: 2, name: 'Ewa' }
	],
	balances: {},
	ppk_balances: {},
	monthly_salaries: {},
	ppk_rates: {}
};

describe('Simulations Page', () => {
	it('renders the main heading', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: 'Symulacje Emerytalne' })).toBeTruthy();
	});

	it('renders the simulation parameters section', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: 'Parametry symulacji' })).toBeTruthy();
	});

	it('renders the accounts-to-simulate section', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: 'Konta do symulacji' })).toBeTruthy();
	});

	it('does not render results before a simulation is run', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.queryByRole('heading', { name: 'Wyniki symulacji' })).toBeNull();
	});
});
