import { test, expect } from '@playwright/test';

test.describe('dashboard', () => {
	test('renders net worth title and charts', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByRole('heading', { name: /Wartość Netto/i }).first()).toBeVisible();
		const canvases = page.locator('canvas');
		await expect(canvases.first()).toBeVisible();
	});

	test('opens retirement limits modal when stats are present', async ({ page }) => {
		await page.goto('/');
		const trigger = page.getByRole('button', { name: 'Konfiguruj limity' });
		if (await trigger.count()) {
			await trigger.first().click();
			await expect(page.getByRole('dialog')).toBeVisible();
			await page.getByRole('dialog').getByRole('button', { name: 'Anuluj' }).click();
			await expect(page.getByRole('dialog')).toBeHidden();
		} else {
			test.info().annotations.push({
				type: 'skipped',
				description: 'no retirement stats seeded'
			});
		}
	});
});
