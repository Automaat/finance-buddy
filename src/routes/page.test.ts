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
		PUBLIC_API_URL: 'http://localhost:8000'
	}
}));

// Mock echarts to avoid canvas issues in tests
vi.mock('echarts', () => ({
	default: {
		init: vi.fn(() => ({
			setOption: vi.fn(),
			resize: vi.fn(),
			dispose: vi.fn()
		})),
		graphic: {
			LinearGradient: vi.fn()
		}
	},
	init: vi.fn(() => ({
		setOption: vi.fn(),
		resize: vi.fn(),
		dispose: vi.fn()
	})),
	graphic: {
		LinearGradient: vi.fn()
	}
}));

vi.mock('$app/navigation', () => ({
	invalidateAll: vi.fn()
}));

const baseTileDeltas = {
	net_worth: {
		mom: { absolute: 1000, percentage: 20 },
		yoy: { absolute: 5000, percentage: 100 }
	},
	assets: {
		mom: { absolute: 2000, percentage: 25 },
		yoy: null
	},
	liabilities: {
		mom: { absolute: -500, percentage: -10 },
		yoy: { absolute: -1500, percentage: -30 }
	}
};

function buildData(
	overrides: {
		dashboardData?: Promise<Record<string, unknown>> | Record<string, unknown>;
		owners?: { id: number; name: string }[];
		currentYear?: number;
	} = {}
) {
	const baseDashboard = {
		net_worth_history: [
			{ date: '2024-01-01', value: 5000 },
			{ date: '2024-02-01', value: 6000 }
		],
		current_net_worth: 6000,
		change_vs_last_month: 1000,
		total_assets: 10000,
		total_liabilities: 4000,
		allocation: [
			{ category: 'banking', owner_user_id: 1, value: 5000 },
			{ category: 'investments', owner_user_id: 1, value: 5000 }
		],
		retirementStats: [] as unknown[],
		tile_deltas: baseTileDeltas
	};

	const dashboardData =
		overrides.dashboardData instanceof Promise
			? overrides.dashboardData
			: Promise.resolve({
					...baseDashboard,
					...(overrides.dashboardData ?? {})
				});

	return {
		user: null,
		dashboardData,
		owners: Promise.resolve(overrides.owners ?? []),
		currentYear: overrides.currentYear ?? 2024
	};
}

describe('Dashboard Page', () => {
	it('renders the main heading', () => {
		render(Page, { props: { data: buildData() } });
		expect(screen.getByRole('heading', { name: /Dashboard/i })).toBeTruthy();
	});

	it('renders the description', () => {
		render(Page, { props: { data: buildData() } });
		expect(screen.getByText('Twoja sytuacja finansowa w jednym miejscu')).toBeTruthy();
	});

	it('shows skeleton placeholders while dashboard data is loading', () => {
		const pending = new Promise<Record<string, unknown>>(() => {});
		const { container } = render(Page, {
			props: { data: buildData({ dashboardData: pending }) }
		});
		const live = screen.getByRole('status', { name: /Ładowanie dashboardu/i });
		expect(live).toBeTruthy();
		expect(container.querySelectorAll('.skeleton').length).toBeGreaterThan(0);
	});

	it('renders dashboard metrics after data resolves', async () => {
		render(Page, { props: { data: buildData() } });
		await waitFor(() => expect(screen.getByText('Wartość Netto')).toBeTruthy());
	});

	it('renders six delta badges across three tiles', async () => {
		render(Page, { props: { data: buildData() } });
		await waitFor(() => expect(screen.getAllByLabelText(/^Δ MoM$/)).toHaveLength(3));
		expect(screen.getAllByLabelText(/^Δ YoY$/)).toHaveLength(3);
	});

	it('colors positive net_worth MoM badge green', async () => {
		render(Page, { props: { data: buildData() } });
		await waitFor(() => screen.getAllByLabelText(/^Δ MoM$/));
		const momBadges = screen.getAllByLabelText(/^Δ MoM$/);
		expect(momBadges[0].className).toContain('text-success-600-400');
	});

	it('colors negative liabilities MoM badge red', async () => {
		render(Page, { props: { data: buildData() } });
		await waitFor(() => screen.getAllByLabelText(/^Δ MoM$/));
		const momBadges = screen.getAllByLabelText(/^Δ MoM$/);
		expect(momBadges[2].className).toContain('text-error-600-400');
	});

	it('renders em-dash for null YoY (assets)', async () => {
		render(Page, { props: { data: buildData() } });
		await waitFor(() => screen.getAllByLabelText(/^Δ YoY$/));
		const yoyBadges = screen.getAllByLabelText(/^Δ YoY$/);
		expect(yoyBadges[1].textContent).toContain('—');
	});

	it('uses Polish liability tooltip wording for MoM', async () => {
		render(Page, { props: { data: buildData() } });
		await waitFor(() => screen.getAllByLabelText(/^Δ MoM$/));
		const momBadges = screen.getAllByLabelText(/^Δ MoM$/);
		expect(momBadges[2].getAttribute('title')).toContain('niższe zobowiązania');
	});
});

describe('Dashboard retirement limits save flow', () => {
	const retirementStat = {
		account_wrapper: 'IKE',
		owner_user_id: 1,
		total_contributed: 5000,
		limit_amount: 23000,
		remaining: 18000,
		percentage_used: 21
	};

	function dataWithStats() {
		return buildData({
			dashboardData: {
				net_worth_history: [{ date: '2024-01-01', value: 5000 }],
				current_net_worth: 6000,
				change_vs_last_month: 1000,
				total_assets: 10000,
				total_liabilities: 4000,
				allocation: [{ category: 'banking', owner_user_id: 1, value: 5000 }],
				retirementStats: [retirementStat]
			},
			owners: [{ id: 1, name: 'Marcin' }]
		});
	}

	beforeEach(() => {
		vi.clearAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('renders the retirement limits card when stats exist', async () => {
		render(Page, { props: { data: dataWithStats() } });
		await waitFor(() =>
			expect(screen.getByRole('heading', { name: /Limity Emerytalne 2024/ })).toBeTruthy()
		);
	});

	it('opens the limits modal and PUTs each owner/wrapper limit on save', async () => {
		const fetchMock = vi.fn().mockResolvedValue({ ok: true });
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: dataWithStats() } });

		await waitFor(() => screen.getByLabelText('Konfiguruj limity'));
		await fireEvent.click(screen.getByLabelText('Konfiguruj limity'));
		expect(screen.getByText('Konfiguracja Limitów Emerytalnych')).toBeTruthy();

		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
		const urls = fetchMock.mock.calls.map((c) => c[0] as string);
		expect(urls).toEqual(
			expect.arrayContaining([
				'http://localhost:8000/api/retirement/limits/2024/IKE/1',
				'http://localhost:8000/api/retirement/limits/2024/IKZE/1'
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

		render(Page, { props: { data: dataWithStats() } });
		await waitFor(() => screen.getByLabelText('Konfiguruj limity'));
		await fireEvent.click(screen.getByLabelText('Konfiguruj limity'));

		const ikeInput = screen.getByLabelText('IKE Marcin (PLN)') as HTMLInputElement;
		expect(ikeInput.value).toBe('23000');
	});

	it('shows an error toast when a limit save request fails', async () => {
		const fetchMock = vi.fn().mockResolvedValue({ ok: false });
		vi.stubGlobal('fetch', fetchMock);
		const errorSpy = vi.spyOn(toast, 'error');

		render(Page, { props: { data: dataWithStats() } });
		await waitFor(() => screen.getByLabelText('Konfiguruj limity'));
		await fireEvent.click(screen.getByLabelText('Konfiguruj limity'));
		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() => expect(errorSpy).toHaveBeenCalled());
		expect(errorSpy.mock.calls[0][0]).toContain('Nie udało się zapisać limitów');
		expect(invalidateAll).not.toHaveBeenCalled();
		errorSpy.mockRestore();
	});
});
