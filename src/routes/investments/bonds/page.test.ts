import { describe, expect, it, vi, beforeAll, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
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

vi.mock('$app/navigation', () => ({
	goto: vi.fn(),
	invalidateAll: vi.fn()
}));

vi.mock('echarts', () => ({
	init: vi.fn(() => ({ setOption: vi.fn(), resize: vi.fn(), dispose: vi.fn() })),
	graphic: { LinearGradient: vi.fn() }
}));

const emptyLadder = { events: [], next_maturity: null, tax_rate_pct: 19 };

const emptyData = {
	user: null,
	bonds: [],
	total_value: 0,
	total_count: 0,
	owners: [{ id: 1, name: 'Marcin' }],
	accounts: [],
	ladder: emptyLadder
};

const sampleBond = {
	id: 1,
	type: 'EDO' as const,
	series: 'EDO0535',
	face_value: 1000,
	purchase_date: '2024-01-01',
	maturity_date: '2034-01-01',
	owner_user_id: 1,
	account_id: null,
	first_year_rate: 6.8,
	margin: 2.0,
	capitalize: true,
	current_value: 1132.08,
	current_yield: 6.0,
	created_at: '2024-01-01T00:00:00'
};

describe('Bonds page', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('renders empty state when no bonds exist', () => {
		render(Page, { props: { data: emptyData } });
		expect(screen.getByText('Brak obligacji')).toBeTruthy();
		expect(
			screen.getByText('Dodaj obligację aby zobaczyć projekcję wartości do wykupu.')
		).toBeTruthy();
	});

	it('lists existing bonds with computed value and owner', () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			json: async () => ({ bond_id: 1, points: [] })
		});
		vi.stubGlobal('fetch', fetchMock);

		render(Page, {
			props: {
				data: { ...emptyData, bonds: [sampleBond], total_value: 1132.08, total_count: 1 }
			}
		});
		expect(screen.getByText('EDO0535')).toBeTruthy();
		expect(screen.getByText('Marcin')).toBeTruthy();
		expect(screen.getByText('6.00%')).toBeTruthy();
	});

	it('shows create form when "Dodaj obligację" is clicked', async () => {
		render(Page, { props: { data: emptyData } });
		await fireEvent.click(screen.getByRole('button', { name: /Dodaj obligację/ }));
		await waitFor(() => expect(screen.getByText('Nowa obligacja')).toBeTruthy());
		expect(screen.getByLabelText(/Seria/)).toBeTruthy();
		expect(screen.getByLabelText(/Wartość nominalna/)).toBeTruthy();
	});

	it('posts to /api/bonds on submit with correct payload', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			json: async () => sampleBond
		});
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: emptyData } });
		await fireEvent.click(screen.getByRole('button', { name: /Dodaj obligację/ }));

		const series = screen.getByLabelText(/Seria/) as HTMLInputElement;
		await fireEvent.input(series, { target: { value: 'EDO0535' } });

		const buttons = screen.getAllByRole('button', { name: /Dodaj obligację/i });
		// Submit button (in-form) is the last one; the header trigger comes first.
		await fireEvent.click(buttons[buttons.length - 1]);

		await waitFor(() => {
			expect(fetchMock).toHaveBeenCalled();
			const [url, init] = fetchMock.mock.calls[0];
			expect(url).toContain('/api/bonds');
			expect(init.method).toBe('POST');
			const body = JSON.parse(init.body);
			expect(body.type).toBe('EDO');
			expect(body.series).toBe('EDO0535');
		});
	});
});
