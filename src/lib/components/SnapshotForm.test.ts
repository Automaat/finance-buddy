import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { goto, invalidateAll } from '$app/navigation';
import SnapshotForm from './SnapshotForm.svelte';
import type { Account, Asset, SnapshotResponse, SnapshotValueResponse } from '$lib/types';

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

function makeAccount(overrides: Partial<Account>): Account {
	return {
		id: 1,
		name: 'Account',
		type: 'asset',
		category: 'bank',
		owner_user_id: null,
		currency: 'PLN',
		account_wrapper: null,
		purpose: 'general',
		square_meters: null,
		is_active: true,
		receives_contributions: false,
		excluded_from_fire: false,
		created_at: '2024-01-01',
		current_value: 0,
		...overrides
	};
}

function makeAsset(overrides: Partial<Asset>): Asset {
	return {
		id: 1,
		name: 'Asset',
		is_active: true,
		created_at: '2024-01-01',
		current_value: 0,
		...overrides
	};
}

function makeSnapshotValue(overrides: Partial<SnapshotValueResponse>): SnapshotValueResponse {
	return {
		id: 1,
		asset_id: null,
		asset_name: null,
		account_id: null,
		account_name: null,
		value: 0,
		...overrides
	};
}

describe('SnapshotForm', () => {
	const bankAccount = makeAccount({
		id: 1,
		name: 'Konto Główne',
		category: 'bank',
		current_value: 5000
	});
	const mortgage = makeAccount({
		id: 2,
		name: 'Kredyt Hipoteczny',
		type: 'liability',
		category: 'mortgage',
		current_value: 200000
	});

	it('renders the date snapshot section with date and notes inputs', () => {
		render(SnapshotForm, {
			props: { assets: [], liabilities: [], physicalAssets: [] }
		});
		expect(screen.getByRole('heading', { name: 'Data Snapshot' })).toBeTruthy();
		expect(screen.getByLabelText('Data')).toBeTruthy();
		expect(screen.getByLabelText('Notatki (opcjonalne)')).toBeTruthy();
	});

	it('renders the financial accounts section', () => {
		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});
		expect(screen.getByRole('heading', { name: /Konta finansowe/ })).toBeTruthy();
	});

	it('shows an input for a financial account with positive value in create mode', () => {
		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});
		expect(screen.getByLabelText(/Konto Główne/)).toBeTruthy();
	});

	it('renders the liabilities section when liabilities are provided', () => {
		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [mortgage], physicalAssets: [] }
		});
		expect(screen.getByRole('heading', { name: /Zobowiązania/ })).toBeTruthy();
		expect(screen.getByLabelText(/Kredyt Hipoteczny/)).toBeTruthy();
	});

	it('renders the submit button', () => {
		render(SnapshotForm, {
			props: { assets: [], liabilities: [], physicalAssets: [] }
		});
		expect(screen.getByRole('button', { name: /Zapisz Snapshot/ })).toBeTruthy();
	});

	it('populates date and notes from editingSnapshot', () => {
		const editingSnapshot: SnapshotResponse = {
			id: 7,
			date: '2024-03-31',
			notes: 'Marcowy snapshot',
			values: []
		};
		render(SnapshotForm, {
			props: { editingSnapshot, assets: [], liabilities: [], physicalAssets: [] }
		});
		expect((screen.getByLabelText('Data') as HTMLInputElement).value).toBe('2024-03-31');
		expect((screen.getByLabelText('Notatki (opcjonalne)') as HTMLInputElement).value).toBe(
			'Marcowy snapshot'
		);
	});

	it('removes a financial account field when its remove button is clicked', async () => {
		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});
		expect(screen.getByLabelText(/Konto Główne/)).toBeTruthy();

		await fireEvent.click(screen.getByTitle('Usuń pole'));

		expect(screen.queryByLabelText(/Konto Główne/)).toBeNull();
	});
});

describe('SnapshotForm submit', () => {
	const bankAccount = makeAccount({
		id: 1,
		name: 'Konto Główne',
		category: 'bank',
		current_value: 5000
	});
	const mortgage = makeAccount({
		id: 2,
		name: 'Kredyt Hipoteczny',
		type: 'liability',
		category: 'mortgage',
		current_value: 200000
	});

	beforeEach(() => {
		vi.clearAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('POSTs a new snapshot and navigates home on success', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			json: async () => ({ id: 99 })
		});
		vi.stubGlobal('fetch', fetchMock);

		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [mortgage], physicalAssets: [] }
		});

		await fireEvent.submit(
			screen.getByRole('button', { name: /Zapisz Snapshot/ }).closest('form')!
		);

		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
		const [url, options] = fetchMock.mock.calls[0];
		expect(url).toBe('http://localhost:8000/api/snapshots');
		expect(options.method).toBe('POST');
		const body = JSON.parse(options.body as string);
		expect(body.values).toEqual(
			expect.arrayContaining([
				{ account_id: 1, value: 5000 },
				{ account_id: 2, value: 200000 }
			])
		);
		expect(goto).toHaveBeenCalledWith('/');
	});

	it('shows the API error detail when create fails', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: false,
			json: async () => ({ detail: 'Snapshot już istnieje' })
		});
		vi.stubGlobal('fetch', fetchMock);

		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});

		await fireEvent.submit(
			screen.getByRole('button', { name: /Zapisz Snapshot/ }).closest('form')!
		);

		await waitFor(() => expect(screen.getByText('Snapshot już istnieje')).toBeTruthy());
		expect(goto).not.toHaveBeenCalled();
	});

	it('PUTs an updated snapshot to the snapshot id endpoint in edit mode', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			json: async () => ({ id: 7 })
		});
		vi.stubGlobal('fetch', fetchMock);

		const editingSnapshot: SnapshotResponse = {
			id: 7,
			date: '2024-03-31',
			notes: 'Marcowy snapshot',
			values: [makeSnapshotValue({ account_id: 1, value: 4200 })]
		};

		render(SnapshotForm, {
			props: { editingSnapshot, assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});

		await fireEvent.submit(
			screen.getByRole('button', { name: /Zapisz Snapshot/ }).closest('form')!
		);

		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
		const [url, options] = fetchMock.mock.calls[0];
		expect(url).toBe('http://localhost:8000/api/snapshots/7');
		expect(options.method).toBe('PUT');
		const body = JSON.parse(options.body as string);
		expect(body.date).toBe('2024-03-31');
		expect(body.values).toEqual([{ account_id: 1, value: 4200 }]);
		expect(goto).toHaveBeenCalledWith('/');
	});

	it('shows the API error detail when edit fails', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: false,
			json: async () => ({ detail: 'Nie udało się zaktualizować' })
		});
		vi.stubGlobal('fetch', fetchMock);

		const editingSnapshot: SnapshotResponse = {
			id: 7,
			date: '2024-03-31',
			notes: '',
			values: [makeSnapshotValue({ account_id: 1, value: 4200 })]
		};

		render(SnapshotForm, {
			props: { editingSnapshot, assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});

		await fireEvent.submit(
			screen.getByRole('button', { name: /Zapisz Snapshot/ }).closest('form')!
		);

		await waitFor(() => expect(screen.getByText('Nie udało się zaktualizować')).toBeTruthy());
		expect(goto).not.toHaveBeenCalled();
	});

	it('includes physical asset values in the edit payload', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			json: async () => ({ id: 8 })
		});
		vi.stubGlobal('fetch', fetchMock);

		const apartment = makeAsset({ id: 5, name: 'Mieszkanie', current_value: 500000 });
		const editingSnapshot: SnapshotResponse = {
			id: 8,
			date: '2024-04-30',
			notes: '',
			values: [
				makeSnapshotValue({ id: 1, account_id: 1, value: 4200 }),
				makeSnapshotValue({ id: 2, asset_id: 5, value: 500000 })
			]
		};

		render(SnapshotForm, {
			props: {
				editingSnapshot,
				assets: [bankAccount],
				liabilities: [],
				physicalAssets: [apartment]
			}
		});

		await fireEvent.submit(
			screen.getByRole('button', { name: /Zapisz Snapshot/ }).closest('form')!
		);

		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
		const body = JSON.parse(fetchMock.mock.calls[0][1].body as string);
		expect(body.values).toEqual(
			expect.arrayContaining([
				{ account_id: 1, value: 4200 },
				{ asset_id: 5, value: 500000 }
			])
		);
	});
});

describe('SnapshotForm — sections, hide/show, modals', () => {
	const bankAccount = makeAccount({
		id: 1,
		name: 'Konto Główne',
		category: 'bank',
		current_value: 5000
	});
	const hiddenBank = makeAccount({
		id: 2,
		name: 'Konto Pomocnicze',
		category: 'bank',
		current_value: 0
	});
	const ikeAccount = makeAccount({
		id: 3,
		name: 'IKE Marcin',
		category: 'stock',
		account_wrapper: 'IKE',
		current_value: 12000
	});
	const stockAccount = makeAccount({
		id: 4,
		name: 'Maklerski',
		category: 'stock',
		current_value: 8000
	});
	const apartment = makeAsset({ id: 5, name: 'Mieszkanie', current_value: 500000 });
	const hiddenAsset = makeAsset({ id: 6, name: 'Działka', current_value: 0 });
	const carAccount = makeAccount({
		id: 7,
		name: 'Auto',
		category: 'vehicle',
		current_value: 30000
	});

	beforeEach(() => {
		vi.clearAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('renders the Emerytura section when retirement accounts exist', () => {
		render(SnapshotForm, {
			props: { assets: [ikeAccount], liabilities: [], physicalAssets: [] }
		});
		expect(screen.getByRole('heading', { name: /Emerytura/ })).toBeTruthy();
		expect(screen.getByText('IKE')).toBeTruthy();
	});

	it('renders Inwestycje and Majątek headers', () => {
		render(SnapshotForm, {
			props: {
				assets: [bankAccount, stockAccount, carAccount],
				liabilities: [],
				physicalAssets: [apartment]
			}
		});
		expect(screen.getByRole('heading', { name: /Inwestycje/ })).toBeTruthy();
		expect(screen.getByRole('heading', { name: /Majątek/ })).toBeTruthy();
		expect(screen.getByLabelText(/Auto/)).toBeTruthy();
		expect(screen.getByLabelText(/Mieszkanie/)).toBeTruthy();
	});

	it('hidden accounts (current_value 0) appear under "Pokaż ukryte konta" and can be re-added', async () => {
		render(SnapshotForm, {
			props: { assets: [bankAccount, hiddenBank], liabilities: [], physicalAssets: [] }
		});
		// Hidden one isn't in the visible value list initially.
		expect(screen.queryByLabelText(/Konto Pomocnicze/)).toBeNull();
		await fireEvent.click(screen.getByRole('button', { name: /Konto Pomocnicze/ }));
		expect(screen.getByLabelText(/Konto Pomocnicze/)).toBeTruthy();
	});

	it('hidden physical assets can be re-added from Majątek section', async () => {
		render(SnapshotForm, {
			props: { assets: [], liabilities: [], physicalAssets: [apartment, hiddenAsset] }
		});
		expect(screen.queryByLabelText(/Działka/)).toBeNull();
		await fireEvent.click(screen.getByRole('button', { name: /Działka/ }));
		expect(screen.getByLabelText(/Działka/)).toBeTruthy();
	});

	it('opens the new-account modal when the "+ Dodaj nowe konto" button under financial is clicked', async () => {
		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});
		const addBtns = screen.getAllByRole('button', { name: /Dodaj nowe konto/ });
		await fireEvent.click(addBtns[0]);
		// Modal asks for "Nazwa konta" — its label is sufficient evidence.
		await waitFor(() => expect(screen.getByLabelText(/Nazwa konta/)).toBeTruthy());
	});

	it('blocks new-account create when name is empty', async () => {
		const fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);
		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});
		const addBtns = screen.getAllByRole('button', { name: /Dodaj nowe konto/ });
		await fireEvent.click(addBtns[0]);
		// Click the modal's confirm button — name is empty.
		const createBtn = screen.getByRole('button', { name: 'Utwórz konto' });
		await fireEvent.click(createBtn);
		await waitFor(() => expect(screen.getByText('Nazwa konta jest wymagana')).toBeTruthy());
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('POSTs a new account when the form is submitted', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			json: async () => ({
				id: 99,
				name: 'New Bank',
				type: 'asset',
				category: 'bank',
				owner_user_id: null,
				currency: 'PLN',
				account_wrapper: null,
				purpose: 'general',
				square_meters: null,
				is_active: true,
				receives_contributions: false,
				created_at: '2026-05-20',
				current_value: 0
			})
		});
		vi.stubGlobal('fetch', fetchMock);
		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});
		const addBtns = screen.getAllByRole('button', { name: /Dodaj nowe konto/ });
		await fireEvent.click(addBtns[0]);
		const nameInput = screen.getByLabelText(/^Nazwa konta/) as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'New Bank' } });
		const createBtn = screen.getByRole('button', { name: 'Utwórz konto' });
		await fireEvent.click(createBtn);
		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://localhost:8000/api/accounts');
		expect((init as RequestInit).method).toBe('POST');
		const body = JSON.parse((init as RequestInit).body as string);
		expect(body).toMatchObject({
			name: 'New Bank',
			type: 'asset',
			category: 'bank',
			currency: 'PLN'
		});
	});

	it('shows detail error when create-account API fails', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: false,
			json: async () => ({ detail: 'Already exists' })
		});
		vi.stubGlobal('fetch', fetchMock);
		render(SnapshotForm, {
			props: { assets: [bankAccount], liabilities: [], physicalAssets: [] }
		});
		const addBtns = screen.getAllByRole('button', { name: /Dodaj nowe konto/ });
		await fireEvent.click(addBtns[0]);
		const nameInput = screen.getByLabelText(/^Nazwa konta/) as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'Dup' } });
		const createBtn = screen.getByRole('button', { name: 'Utwórz konto' });
		await fireEvent.click(createBtn);
		await waitFor(() => expect(screen.getByText('Already exists')).toBeTruthy());
	});

	it('opens the new-asset modal and POSTs a new asset on submit', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			json: async () => ({
				id: 42,
				name: 'Nowy Majątek',
				is_active: true,
				created_at: '2026-05-20',
				current_value: 0
			})
		});
		vi.stubGlobal('fetch', fetchMock);
		render(SnapshotForm, {
			props: { assets: [], liabilities: [], physicalAssets: [apartment] }
		});
		await fireEvent.click(screen.getByRole('button', { name: /Dodaj nowy majątek/ }));
		const nameInput = screen.getByLabelText(/Nazwa/) as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'Nowy Majątek' } });
		const createBtn = screen.getByRole('button', { name: 'Utwórz majątek' });
		await fireEvent.click(createBtn);
		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://localhost:8000/api/assets');
		expect((init as RequestInit).method).toBe('POST');
		const body = JSON.parse((init as RequestInit).body as string);
		expect(body).toEqual({ name: 'Nowy Majątek' });
	});

	it('blocks new-asset create when name is empty', async () => {
		const fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);
		render(SnapshotForm, {
			props: { assets: [], liabilities: [], physicalAssets: [apartment] }
		});
		await fireEvent.click(screen.getByRole('button', { name: /Dodaj nowy majątek/ }));
		const createBtn = screen.getByRole('button', { name: 'Utwórz majątek' });
		await fireEvent.click(createBtn);
		await waitFor(() => expect(screen.getByText('Nazwa majątku jest wymagana')).toBeTruthy());
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('Anuluj button navigates to "/"', async () => {
		render(SnapshotForm, {
			props: { assets: [], liabilities: [], physicalAssets: [] }
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Anuluj' }));
		expect(goto).toHaveBeenCalledWith('/');
	});

	describe('quote freshness banner', () => {
		// A far-past quote date is stale regardless of the current clock.
		const staleHolding = {
			security: { name: 'VWCE' },
			quantity: '10',
			latest_quote_date: '2020-01-01'
		};

		afterEach(() => {
			vi.unstubAllGlobals();
		});

		it('shows the stale-quote banner in create mode', () => {
			render(SnapshotForm, {
				props: {
					assets: [bankAccount],
					liabilities: [],
					physicalAssets: [],
					holdings: [staleHolding]
				}
			});
			expect(screen.getByText(/Notowania inwestycji mogą być nieaktualne/)).toBeTruthy();
			expect(screen.getByText(/VWCE/)).toBeTruthy();
		});

		it('suppresses the banner when editing an existing snapshot', () => {
			const editingSnapshot: SnapshotResponse = {
				id: 1,
				date: '2024-03-31',
				notes: null,
				values: []
			};
			render(SnapshotForm, {
				props: {
					editingSnapshot,
					assets: [bankAccount],
					liabilities: [],
					physicalAssets: [],
					holdings: [staleHolding]
				}
			});
			expect(screen.queryByText(/Notowania inwestycji mogą być nieaktualne/)).toBeNull();
		});

		it('refresh button POSTs to refresh-quotes and re-runs load', async () => {
			const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) });
			vi.stubGlobal('fetch', fetchMock);
			render(SnapshotForm, {
				props: {
					assets: [bankAccount],
					liabilities: [],
					physicalAssets: [],
					holdings: [staleHolding]
				}
			});
			await fireEvent.click(screen.getByRole('button', { name: /Aktualizuj ceny/ }));
			await waitFor(() => expect(invalidateAll).toHaveBeenCalled());
			const [url, init] = fetchMock.mock.calls[0];
			expect(url).toBe('http://localhost:8000/api/holdings/refresh-quotes');
			expect(init.method).toBe('POST');
		});
	});
});
