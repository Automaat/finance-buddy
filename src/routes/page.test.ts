import { render, screen } from '@testing-library/svelte';
import { vi } from 'vitest';
import Page from './+page.svelte';

// Mock echarts to avoid canvas issues in tests
vi.mock('echarts', () => ({
	default: {
		init: vi.fn(() => ({
			setOption: vi.fn(),
			resize: vi.fn()
		})),
		graphic: {
			LinearGradient: vi.fn()
		}
	},
	init: vi.fn(() => ({
		setOption: vi.fn(),
		resize: vi.fn()
	})),
	graphic: {
		LinearGradient: vi.fn()
	}
}));

describe('Dashboard Page', () => {
	const mockData = {
		net_worth_history: [
			{ date: '2024-01-01', value: 5000 },
			{ date: '2024-02-01', value: 6000 }
		],
		current_net_worth: 6000,
		change_vs_last_month: 1000,
		total_assets: 10000,
		total_liabilities: 4000,
		allocation: [
			{ category: 'banking', owner: 'John', value: 5000 },
			{ category: 'investments', owner: 'John', value: 5000 }
		]
	};

	it('renders the main heading', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByRole('heading', { name: /Dashboard/i })).toBeTruthy();
	});

	it('renders the description', () => {
		render(Page, { props: { data: mockData } });
		expect(screen.getByText('Twoja sytuacja finansowa w jednym miejscu')).toBeTruthy();
	});
});
