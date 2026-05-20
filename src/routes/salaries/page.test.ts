import { describe, expect, it, vi, beforeAll, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { invalidateAll } from '$app/navigation';
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
	init: vi.fn(() => ({ setOption: vi.fn(), resize: vi.fn(), dispose: vi.fn() }))
}));

const baseData = {
	salaries: {
		salary_records: [],
		total_count: 0,
		current_salaries: { Marcin: null }
	},
	filters: { owner: null, date_from: null, date_to: null },
	personas: [{ id: 1, name: 'Marcin', ppk_employee_rate: 2, ppk_employer_rate: 1.5 }]
};

async function openNewSalaryModalAndFill(opts: {
	date?: string;
	gross_amount?: string;
	company?: string;
}) {
	await fireEvent.click(screen.getByRole('button', { name: /Nowe Wynagrodzenie/i }));

	const dateInput = screen.getByLabelText(/Data zmiany/) as HTMLInputElement;
	const amountInput = screen.getByLabelText(/Pensja brutto/) as HTMLInputElement;
	const companyInput = screen.getByLabelText(/^Firma/) as HTMLInputElement;

	if (opts.date !== undefined) await fireEvent.input(dateInput, { target: { value: opts.date } });
	if (opts.gross_amount !== undefined)
		await fireEvent.input(amountInput, { target: { value: opts.gross_amount } });
	if (opts.company !== undefined)
		await fireEvent.input(companyInput, { target: { value: opts.company } });
}

describe('Salaries page — saveSalary validation & error display', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-05-20T12:00:00Z'));
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it('blocks save and shows error when date is in the future', async () => {
		const fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: baseData } });
		await openNewSalaryModalAndFill({
			date: '2026-06-20',
			gross_amount: '5000',
			company: 'ACME'
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() => expect(screen.getByText('Data nie może być z przyszłości')).toBeTruthy());
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('blocks save and shows error when date is empty', async () => {
		const fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: baseData } });
		await openNewSalaryModalAndFill({
			date: '',
			gross_amount: '5000',
			company: 'ACME'
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() => expect(screen.getByText('Data jest wymagana')).toBeTruthy());
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('renders joined messages from FastAPI 422 detail array', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: false,
			json: async () => ({
				detail: [
					{ loc: ['body', 'gross_amount'], msg: 'Gross amount must be greater than 0' },
					{ loc: ['body', 'company'], msg: 'Company cannot be empty' }
				]
			})
		});
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: baseData } });
		await openNewSalaryModalAndFill({
			date: '2026-05-20',
			gross_amount: '5000',
			company: 'ACME'
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() =>
			expect(
				screen.getByText('Gross amount must be greater than 0; Company cannot be empty')
			).toBeTruthy()
		);
		expect(invalidateAll).not.toHaveBeenCalled();
	});

	it('falls back to default message when 422 detail array has no usable msg fields', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: false,
			json: async () => ({ detail: [{ loc: ['body', 'date'] }, { loc: ['body', 'gross_amount'] }] })
		});
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: baseData } });
		await openNewSalaryModalAndFill({
			date: '2026-05-20',
			gross_amount: '5000',
			company: 'ACME'
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() => expect(screen.getByText('Failed to save salary record')).toBeTruthy());
	});

	it('rejects future date evaluated at submit time, not page load', async () => {
		const fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: baseData } });
		await openNewSalaryModalAndFill({
			date: '2026-05-21',
			gross_amount: '5000',
			company: 'ACME'
		});

		vi.setSystemTime(new Date('2026-05-20T23:59:00Z'));
		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() => expect(screen.getByText('Data nie może być z przyszłości')).toBeTruthy());
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('renders string detail unchanged on 409 conflict', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: false,
			json: async () => ({ detail: 'Salary record for Marcin on 2026-05-20 already exists' })
		});
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: baseData } });
		await openNewSalaryModalAndFill({
			date: '2026-05-20',
			gross_amount: '5000',
			company: 'ACME'
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() =>
			expect(screen.getByText('Salary record for Marcin on 2026-05-20 already exists')).toBeTruthy()
		);
	});

	it('POSTs to /api/salaries and invalidates on success', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			json: async () => ({})
		});
		vi.stubGlobal('fetch', fetchMock);

		render(Page, { props: { data: baseData } });
		await openNewSalaryModalAndFill({
			date: '2026-05-20',
			gross_amount: '5000',
			company: 'ACME'
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));

		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://localhost:8000/api/salaries');
		expect((init as RequestInit).method).toBe('POST');
		const body = JSON.parse((init as RequestInit).body as string);
		expect(body).toMatchObject({
			date: '2026-05-20',
			gross_amount: 5000,
			contract_type: 'UOP',
			company: 'ACME',
			owner: 'Marcin'
		});
		await waitFor(() => expect(invalidateAll).toHaveBeenCalled());
	});
});
