import { test, expect } from '@playwright/test';

function uniqueName(prefix: string): string {
	const stamp = Date.now().toString(36) + Math.random().toString(36).slice(2, 6);
	return `${prefix}-${stamp}`;
}

test.describe('accounts CRUD', () => {
	test('create asset account then soft-delete it', async ({ page }) => {
		await page.goto('/accounts');
		await expect(page.getByRole('heading', { name: 'Konta' })).toBeVisible();

		const name = uniqueName('e2e-bank');

		await page.getByRole('button', { name: /Nowe Konto/ }).click();

		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();

		await dialog.locator('label:has-text("Nazwa") input').fill(name);
		await dialog.locator('label:has-text("Typ") select').selectOption('asset');
		await dialog.locator('label:has-text("Kategoria") select').selectOption('bank');

		await dialog.getByRole('button', { name: /Utwórz konto/ }).click();
		await expect(dialog).toBeHidden();

		const row = page.getByRole('row', { name: new RegExp(name) });
		await expect(row).toBeVisible();

		await row.getByRole('button', { name: 'Usuń' }).click();

		const confirm = page.getByRole('dialog').filter({ hasText: 'Potwierdzenie usunięcia' });
		await expect(confirm).toBeVisible();
		await confirm.getByRole('button', { name: 'Usuń', exact: true }).click();

		await expect(page.getByRole('row', { name: new RegExp(name) })).toBeHidden();
	});
});
