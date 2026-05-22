import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { goto } from '$app/navigation';
import SnapshotForm from './SnapshotForm.svelte';
import type { Account, Asset, SnapshotResponse, SnapshotValueResponse } from '$lib/types';

vi.mock('$env/dynamic/public', () => ({
	env: {
		PUBLIC_API_URL_BROWSER: 'http://localhost:8000',
		PUBLIC_API_URL: 'http://localhost:8000'
	}
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
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
