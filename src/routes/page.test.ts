import { describe, expect, it, beforeAll, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { vi } from 'vitest';
import { invalidateAll } from '$app/navigation';
import Page from './+page.svelte';
import { toast } from '$lib/stores/toast.svelte';

// Mock window.matchMedia for media query tests
beforeAll(() => {
	Object.defineProperty(window, 'matchMedia', {
		writable: true,
		value: vi.fn().mockImplementation((query) => ({
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

// Mock SvelteKit env module
vi.mock('$env/dynamic/public', () => ({
	env: {
		PUBLIC_API_URL_BROWSER: 'http://localhost:8000',
		PUBLIC_API_URL: 'http://localhost:8000',
		PUBLIC_DEFAULT_OWNER: 'Marcin'
	}
}));

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

vi.mock('$app/navigation', () => ({
	invalidateAll: vi.fn()
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
		],
		retirementStats: [],
		currentYear: 2024
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

describe('Dashboard retirement limits save flow', () => {
	const dataWithStats = {
		net_worth_history: [{ date: '2024-01-01', value: 5000 }],
		current_net_worth: 6000,
		change_vs_last_month: 1000,
		total_assets: 10000,
		total_liabilities: 4000,
		allocation: [{ category: 'banking', owner: 'Marcin', value: 5000 }],
		retirementStats: [
			{
				account_wrapper: 'IKE',
				owner: 'Marcin',
				total_contributed: 5000,
				limit_amount: 23000,
				remaining: 18000,
				percentage_used: 21
			}
		],
		personas: [{ name: 'Marcin' }],
		currentYear: 2024
	};

	beforeEach(() => {
		vi.clearAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('renders the retirement limits card when stats exist', () => {
		render(Page, { props: { data: dataWithStats } });
		expect(screen.getByRole('heading', { name: /Limity Emerytalne 2024/ })).toBeTruthy();
	});

	it('opens the limits modal and PUTs each persona/wrapper limit on save', async () => {
		const fetchMock = vi.fn().mockResolvedValue({ ok: true });
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: dataWithStats } });

		await fireEvent.click(screen.getByLabelText('Konfiguruj limity'));
		expect(screen.getByText('Konfiguracja Limitów Emerytalnych')).toBeTruthy();

		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
		const urls = fetchMock.mock.calls.map((c) => c[0] as string);
		expect(urls).toEqual(
			expect.arrayContaining([
				'http://localhost:8000/api/retirement/limits/2024/IKE/Marcin',
				'http://localhost:8000/api/retirement/limits/2024/IKZE/Marcin'
			])
		);
		for (const call of fetchMock.mock.calls) {
			expect(call[1].method).toBe('PUT');
		}
		await waitFor(() => expect(invalidateAll).toHaveBeenCalled());
	});

	it('prefills the IKE limit from existing retirement stats', async () => {
		const fetchMock = vi.fn().mockResolvedValue({ ok: true });
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: dataWithStats } });
		await fireEvent.click(screen.getByLabelText('Konfiguruj limity'));

		const ikeInput = screen.getByLabelText('IKE Marcin (PLN)') as HTMLInputElement;
		expect(ikeInput.value).toBe('23000');
	});

	it('shows an error toast when a limit save request fails', async () => {
		const fetchMock = vi.fn().mockResolvedValue({ ok: false });
		vi.stubGlobal('fetch', fetchMock);
		const errorSpy = vi.spyOn(toast, 'error');

		render(Page, { props: { data: dataWithStats } });
		await fireEvent.click(screen.getByLabelText('Konfiguruj limity'));
		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() => expect(errorSpy).toHaveBeenCalled());
		expect(errorSpy.mock.calls[0][0]).toContain('Nie udało się zapisać limitów');
		expect(invalidateAll).not.toHaveBeenCalled();
	});
});
