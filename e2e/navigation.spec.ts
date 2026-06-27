import { test, expect } from '@playwright/test';

const SIDEBAR_LINKS: { label: string; path: string }[] = [
	{ label: 'Metryki', path: '/metryki' },
	{ label: 'Konta', path: '/accounts' },
	{ label: 'Transakcje', path: '/transactions' },
	{ label: 'Majątek', path: '/assets' },
	{ label: 'Zobowiązania', path: '/debts' },
	{ label: 'Cele', path: '/goals' },
	{ label: 'Migawki', path: '/snapshots' },
	{ label: 'Wynagrodzenia', path: '/salaries' },
	{ label: 'Ustawienia', path: '/settings/config' }
];

test.describe('navigation', () => {
	for (const link of SIDEBAR_LINKS) {
		test(`desktop sidebar link navigates to ${link.path}`, async ({ page, isMobile }) => {
			test.skip(isMobile, 'desktop-only sidebar');
			await page.goto('/');
			await page
				.locator('aside')
				.getByRole('link', { name: new RegExp(`^${link.label}$`) })
				.click();
			await expect(page).toHaveURL(new RegExp(`${link.path}$`));
		});
	}

	test('mobile bottom nav is visible and scrollable', async ({ page, isMobile }) => {
		test.skip(!isMobile, 'mobile-only nav');
		await page.goto('/');
		const nav = page.getByRole('navigation', { name: 'Mobile navigation' });
		await expect(nav).toBeVisible();
		await expect(nav.getByRole('link', { name: 'Dashboard' })).toBeVisible();
		await nav.getByRole('link', { name: 'Konta' }).click();
		await expect(page).toHaveURL(/\/accounts$/);
	});
});
