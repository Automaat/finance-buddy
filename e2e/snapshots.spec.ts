import { test, expect } from '@playwright/test';

const apiUrl = process.env.E2E_API_URL ?? 'http://127.0.0.1:8000';

test.describe('snapshots', () => {
	test('list page shows headers and create button', async ({ page }) => {
		await page.goto('/snapshots');
		await expect(page.getByRole('heading', { name: 'Snapshots', exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: /Nowy Snapshot/ })).toBeVisible();
	});

	test('new snapshot form renders with prefilled date and submit button', async ({ page }) => {
		await page.goto('/snapshots/new');
		await expect(page.getByRole('heading', { name: 'Nowy Snapshot' })).toBeVisible();

		const dateInput = page.locator('input#date');
		await expect(dateInput).toBeVisible();
		const value = await dateInput.inputValue();
		expect(value).toMatch(/^\d{4}-\d{2}-\d{2}$/);

		await expect(page.getByRole('button', { name: /Zapisz Snapshot/ })).toBeVisible();
	});

	test('GET /api/snapshots returns list', async ({ request }) => {
		const res = await request.get(`${apiUrl}/api/snapshots`);
		expect(res.status()).toBe(200);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
		if (list.length > 0) {
			expect(list[0]).toHaveProperty('date');
			expect(list[0]).toHaveProperty('total_net_worth');
		}
	});
});
