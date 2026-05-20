import { test, expect } from '@playwright/test';
import { uniqueName } from './utils';

test.describe('goals CRUD', () => {
	test('create, verify, then delete a goal', async ({ page }) => {
		await page.goto('/goals');
		await expect(page.getByRole('heading', { name: 'Cele finansowe' })).toBeVisible();

		const name = uniqueName('e2e-goal');

		await page.getByRole('button', { name: /Nowy cel/ }).click();

		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();

		await dialog.locator('label:has-text("Nazwa") input').fill(name);
		await dialog.locator('label:has-text("Cel (PLN)") input').fill('25000');

		const inOneYear = new Date();
		inOneYear.setFullYear(inOneYear.getFullYear() + 1);
		const isoDate = inOneYear.toISOString().slice(0, 10);
		await dialog.locator('label:has-text("Data celu") input').fill(isoDate);

		await dialog.locator('label:has-text("Obecna kwota") input').fill('5000');
		await dialog.locator('label:has-text("Wkład miesięczny") input').fill('1500');

		await dialog.getByRole('button', { name: 'Zapisz', exact: true }).click();
		await expect(dialog).toBeHidden();

		const card = page.locator('article').filter({ has: page.getByRole('heading', { name }) });
		await expect(card).toBeVisible();
		await expect(card.getByText('25 000', { exact: false })).toBeVisible();

		await card.getByRole('button', { name: 'Usuń' }).click();
		const confirm = page.getByRole('dialog').filter({ hasText: 'Usunąć cel?' });
		await expect(confirm).toBeVisible();
		await confirm.getByRole('button', { name: 'Usuń', exact: true }).click();

		await expect(page.getByRole('heading', { name })).toBeHidden();
	});
});
