import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Page from './+page.svelte';

vi.mock('$env/dynamic/public', () => ({
	env: {
		PUBLIC_API_URL_BROWSER: 'http://localhost:8000',
		PUBLIC_API_URL: 'http://localhost:8000'
	}
}));

vi.mock('$app/environment', () => ({ browser: false }));

vi.mock('$app/navigation', () => ({
	invalidateAll: vi.fn(),
	goto: vi.fn()
}));

vi.mock('$lib/stores/toast.svelte', () => ({
	toast: { success: vi.fn(), error: vi.fn() }
}));

function pageData(targets: unknown[] = [], owners: unknown[] = []) {
	return {
		data: {
			targets: { targets },
			owners
		}
	};
}

describe('settings/allocation page', () => {
	it('renders the headline and scope picker', () => {
		render(Page, pageData());
		expect(screen.getByText('Cele alokacji')).toBeTruthy();
		expect(screen.getAllByText('Wspólne (gospodarstwo)').length).toBeGreaterThan(0);
	});

	it('shows empty-state row when no targets configured', () => {
		render(Page, pageData());
		expect(screen.getByText(/Brak celów/)).toBeTruthy();
	});

	it('seeds draft with existing household targets', () => {
		render(
			Page,
			pageData([
				{
					id: 1,
					category: 'stock',
					owner_user_id: null,
					target_pct: 60,
					created_at: '2026-01-01T00:00:00'
				},
				{
					id: 2,
					category: 'bond',
					owner_user_id: null,
					target_pct: 40,
					created_at: '2026-01-01T00:00:00'
				}
			])
		);
		expect(screen.getByText('100.00%')).toBeTruthy();
	});

	it('blocks save when sum != 100', async () => {
		render(
			Page,
			pageData([
				{
					id: 1,
					category: 'stock',
					owner_user_id: null,
					target_pct: 50,
					created_at: '2026-01-01T00:00:00'
				}
			])
		);
		const saveButton = screen.getByRole('button', { name: /Zapisz/ });
		expect((saveButton as HTMLButtonElement).disabled).toBe(true);
	});

	it('disables add button when all categories used', () => {
		const allCats = [
			'bank',
			'saving_account',
			'stock',
			'bond',
			'gold',
			'real_estate',
			'ppk',
			'fund',
			'etf',
			'vehicle'
		];
		const targets = allCats.map((cat, i) => ({
			id: i + 1,
			category: cat,
			owner_user_id: null,
			target_pct: 10,
			created_at: '2026-01-01T00:00:00'
		}));
		render(Page, pageData(targets));
		const addButton = screen.getByRole('button', { name: /Dodaj kategorię/ });
		expect((addButton as HTMLButtonElement).disabled).toBe(true);
	});

	it('lists owner names in scope picker', () => {
		render(
			Page,
			pageData(
				[],
				[
					{ id: 1, name: 'Marcin' },
					{ id: 2, name: 'Ewa' }
				]
			)
		);
		expect(screen.getByRole('option', { name: 'Marcin' })).toBeTruthy();
		expect(screen.getByRole('option', { name: 'Ewa' })).toBeTruthy();
	});

	it('adds a new row when add button is clicked', async () => {
		render(Page, pageData());
		const addButton = screen.getByRole('button', { name: /Dodaj kategorię/ });
		await fireEvent.click(addButton);
		expect(screen.queryByText(/Brak celów/)).toBeNull();
	});
});
