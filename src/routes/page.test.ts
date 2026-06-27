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
	invalidateAll: vi.fn(),
	goto: vi.fn()
}));

vi.mock('$app/stores', async () => {
	const { readable } = await import('svelte/store');
	return {
		page: readable({ url: new URL('http://localhost/') })
	};
});

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
		currentYear: overrides.currentYear ?? 2024,
		range: 'all' as const,
		dateFrom: null,
		dateTo: null
	};
}

describe('Dashboard Page', () => {
	it('renders the main heading', () => {
		render(Page, { props: { data: buildData() } });
		expect(screen.getByRole('heading', { name: /Pulpit/i })).toBeTruthy();
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
		const live = screen.getByRole('status', { name: /Ładowanie pulpitu/i });
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

		await waitFor(() => {
			const putCalls = fetchMock.mock.calls.filter((c) => c[1]?.method === 'PUT');
			expect(putCalls).toHaveLength(2);
		});
		const putCalls = fetchMock.mock.calls.filter((c) => c[1]?.method === 'PUT');
		const urls = putCalls.map((c) => c[0] as string);
		expect(urls).toEqual(
			expect.arrayContaining([
				'http://localhost:8000/api/retirement/limits/2024/IKE/1',
				'http://localhost:8000/api/retirement/limits/2024/IKZE/1'
			])
		);
		// Guard against accidental non-GET/PUT side-effects (e.g. a stray
		// POST/DELETE) sneaking past the relaxed PUT-only filter.
		for (const call of fetchMock.mock.calls) {
			const method = call[1]?.method ?? 'GET';
			expect(['GET', 'PUT']).toContain(method);
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

describe('Dashboard — FIRE / runway / bonds cards', () => {
	it('renders FIRE+runway card when metric_cards has values', async () => {
		render(Page, {
			props: {
				data: buildData({
					dashboardData: {
						net_worth_history: [{ date: '2024-01-01', value: 5000 }],
						current_net_worth: 6000,
						change_vs_last_month: 1000,
						total_assets: 10000,
						total_liabilities: 4000,
						allocation: [],
						retirementStats: [],
						tile_deltas: baseTileDeltas,
						metric_cards: {
							fire_number: 1000000,
							runway_months: 18.5,
							fi_progress: 25,
							annual_expenses: 40000,
							withdrawal_rate: 0.04
						}
					}
				})
			}
		});
		await waitFor(() => expect(screen.getByText(/FIRE i runway/)).toBeTruthy());
		expect(screen.getByText('FI progress')).toBeTruthy();
		expect(screen.getByText('25.0%')).toBeTruthy();
		expect(screen.getByText('18.5 mies.')).toBeTruthy();
		expect(screen.getByText(/SWR 4\.0%/)).toBeTruthy();
	});

	it('renders the treasury bonds card when count > 0', async () => {
		render(Page, {
			props: {
				data: buildData({
					dashboardData: {
						net_worth_history: [{ date: '2024-01-01', value: 5000 }],
						current_net_worth: 6000,
						change_vs_last_month: 1000,
						total_assets: 10000,
						total_liabilities: 4000,
						allocation: [],
						retirementStats: [],
						tile_deltas: baseTileDeltas,
						treasuryBondsCount: 3,
						treasuryBondsValue: 30000
					}
				})
			}
		});
		await waitFor(() => expect(screen.getByText('Obligacje skarbowe')).toBeTruthy());
		expect(screen.getByText('3 obligacji (auto-wycena wg CPI)')).toBeTruthy();
	});

	it('shows the FIRE-net-worth notice when an exclusion is configured', async () => {
		render(Page, {
			props: {
				data: buildData({
					dashboardData: {
						net_worth_history: [{ date: '2024-01-01', value: 5000 }],
						current_net_worth: 1_000_000,
						change_vs_last_month: 0,
						total_assets: 1_000_000,
						total_liabilities: 0,
						allocation: [],
						retirementStats: [],
						tile_deltas: baseTileDeltas,
						metric_cards: {
							fire_number: 1_500_000,
							runway_months: 12,
							fi_progress: 26.67,
							annual_expenses: 60000,
							withdrawal_rate: 0.04,
							fire_net_worth: 400_000,
							fire_excluded_value: 600_000
						}
					}
				})
			}
		});
		await waitFor(() => expect(screen.getByText(/FIRE i runway/)).toBeTruthy());
		expect(screen.getByText(/FIRE liczone z/)).toBeTruthy();
		expect(screen.getByText(/aktywów oznaczonych „poza FIRE”/)).toBeTruthy();
	});

	it('singularises bonds count to "obligacja" when count is 1', async () => {
		render(Page, {
			props: {
				data: buildData({
					dashboardData: {
						net_worth_history: [{ date: '2024-01-01', value: 5000 }],
						current_net_worth: 6000,
						change_vs_last_month: 1000,
						total_assets: 10000,
						total_liabilities: 4000,
						allocation: [],
						retirementStats: [],
						tile_deltas: baseTileDeltas,
						treasuryBondsCount: 1,
						treasuryBondsValue: 10000
					}
				})
			}
		});
		await waitFor(() => expect(screen.getByText('1 obligacja (auto-wycena wg CPI)')).toBeTruthy());
	});
});

describe('Dashboard — IKE/IKZE year-end countdown', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	function statAt(percentage_used: number) {
		return {
			account_wrapper: 'IKE',
			owner_user_id: 1,
			total_contributed: 1000,
			limit_amount: 28260,
			remaining: 27260,
			percentage_used
		};
	}

	function dataAt(percentage_used: number) {
		return buildData({
			dashboardData: {
				net_worth_history: [{ date: '2024-01-01', value: 5000 }],
				current_net_worth: 6000,
				change_vs_last_month: 0,
				total_assets: 10000,
				total_liabilities: 4000,
				allocation: [],
				retirementStats: [statAt(percentage_used)],
				tile_deltas: baseTileDeltas
			},
			owners: [{ id: 1, name: 'Marcin' }]
		});
	}

	it('shows the days-to-Dec-31 countdown in the header', async () => {
		vi.setSystemTime(new Date(2026, 5, 1, 12, 0, 0));
		render(Page, { props: { data: dataAt(20) } });
		await waitFor(() =>
			expect(screen.getByRole('heading', { name: /Limity Emerytalne/ })).toBeTruthy()
		);
		expect(screen.getByText(/214 dni do 31 grudnia/)).toBeTruthy();
	});

	it('renders maxed badge when limit reached (any month)', async () => {
		vi.setSystemTime(new Date(2026, 0, 15, 12, 0, 0));
		render(Page, { props: { data: dataAt(100) } });
		await waitFor(() => expect(screen.getByText('Limit osiągnięty')).toBeTruthy());
	});

	it('renders no urgency badge in Q1-Q2 when not maxed', async () => {
		vi.setSystemTime(new Date(2026, 3, 15, 12, 0, 0));
		render(Page, { props: { data: dataAt(10) } });
		await waitFor(() =>
			expect(screen.getByRole('heading', { name: /Limity Emerytalne/ })).toBeTruthy()
		);
		expect(screen.queryByText(/Końcówka roku/)).toBeNull();
		expect(screen.queryByText(/Coraz mniej czasu/)).toBeNull();
		expect(screen.queryByText('Limit osiągnięty')).toBeNull();
	});

	it('renders warn badge in Q3 when not maxed', async () => {
		vi.setSystemTime(new Date(2026, 7, 15, 12, 0, 0));
		render(Page, { props: { data: dataAt(40) } });
		await waitFor(() =>
			expect(screen.getByText(/Coraz mniej czasu na wykorzystanie limitu/)).toBeTruthy()
		);
	});

	it('renders urgent badge in Q4 when not maxed', async () => {
		vi.setSystemTime(new Date(2026, 10, 15, 12, 0, 0));
		render(Page, { props: { data: dataAt(70) } });
		await waitFor(() =>
			expect(screen.getByText(/Końcówka roku — limit przepadnie 31\.12/)).toBeTruthy()
		);
	});
});
