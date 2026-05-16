import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SnapshotForm from './SnapshotForm.svelte';
import type { Account, Asset, SnapshotResponse } from '$lib/types';

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
		owner: 'Marcin',
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
