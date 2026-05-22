import { test, expect } from '@playwright/test';
import { INVESTMENT_CATEGORIES } from '../src/lib/constants';
import { openDialog } from './utils';

const apiUrl = process.env.E2E_API_URL ?? 'http://127.0.0.1:8000';

interface ApiAccount {
	id: number;
	name: string;
	category: string;
	owner_user_id: number | null;
	account_wrapper: string | null;
}

async function pickInvestmentAccount(
	request: import('@playwright/test').APIRequestContext
): Promise<ApiAccount> {
	const res = await request.get(`${apiUrl}/api/accounts`);
	expect(res.status()).toBe(200);
	const { assets } = (await res.json()) as { assets: ApiAccount[] };
	const candidate = assets.find(
		(a) => INVESTMENT_CATEGORIES.has(a.category) || a.account_wrapper !== null
	);
	if (!candidate) throw new Error('no investment-eligible account seeded');
	return candidate;
}

test.describe('transactions', () => {
	test('list page renders header and create button', async ({ page }) => {
		await page.goto('/transactions');
		await expect(page.getByRole('heading', { name: 'Wszystkie transakcje' })).toBeVisible();
		await expect(page.getByRole('button', { name: /Nowa Transakcja/ })).toBeVisible();
	});

	test('create transaction via UI then delete via API', async ({ page, request }) => {
		const account = await pickInvestmentAccount(request);

		await page.goto('/transactions');
		await openDialog(page, /Nowa Transakcja/);

		const dialog = page.getByRole('dialog').filter({ hasText: 'Nowa Transakcja' });
		await expect(dialog).toBeVisible();

		await dialog.locator('label:has-text("Konto") select').selectOption(String(account.id));
		await dialog.locator('label:has-text("Kwota") input').fill('1234.56');

		const today = new Date().toISOString().slice(0, 10);
		await dialog.locator('label:has-text("Data zakupu") input').fill(today);

		await dialog.getByRole('button', { name: /Dodaj transakcję/ }).click();
		await expect(dialog).toBeHidden();

		const listRes = await request.get(`${apiUrl}/api/accounts/${account.id}/transactions`);
		expect(listRes.status()).toBe(200);
		const { transactions } = await listRes.json();
		const created = transactions.find(
			(t: { amount: number; date: string }) => t.amount === 1234.56 && t.date === today
		);
		expect(created, 'created transaction should be present').toBeDefined();

		const del = await request.delete(
			`${apiUrl}/api/accounts/${account.id}/transactions/${created.id}`
		);
		expect(del.status()).toBe(204);
	});

	test('GET /api/transactions returns shape', async ({ request }) => {
		const res = await request.get(`${apiUrl}/api/transactions`);
		expect(res.status()).toBe(200);
		const body = await res.json();
		expect(body).toHaveProperty('transactions');
		expect(Array.isArray(body.transactions)).toBe(true);
	});
});
