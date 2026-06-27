import { test, expect } from '@playwright/test';

const routes: { path: string; heading: RegExp }[] = [
	{ path: '/', heading: /Pulpit/i },
	{ path: '/metryki', heading: /Metryki/i },
	{ path: '/simulations', heading: /Symulacje/i },
	{ path: '/accounts', heading: /Konta/i },
	{ path: '/transactions', heading: /Transakcje/i },
	{ path: '/assets', heading: /Majątek/i },
	{ path: '/debts', heading: /Zobowiązania/i },
	{ path: '/goals', heading: /Cele finansowe/i },
	{ path: '/snapshots', heading: /Migawki/i },
	{ path: '/salaries', heading: /Historia wynagrodzeń/i },
	{ path: '/settings/config', heading: /Konfiguracja/i },
	{ path: '/settings/users', heading: /Użytkownicy/i }
];

test.describe('smoke @smoke', () => {
	for (const route of routes) {
		test(`loads ${route.path} without console/page errors`, async ({ page }) => {
			const consoleErrors: string[] = [];
			const pageErrors: string[] = [];

			page.on('console', (msg) => {
				if (msg.type() === 'error') consoleErrors.push(msg.text());
			});
			page.on('pageerror', (err) => pageErrors.push(err.message));

			const response = await page.goto(route.path, { waitUntil: 'domcontentloaded' });
			expect(response, `no response for ${route.path}`).not.toBeNull();
			expect(response!.status(), `bad status on ${route.path}`).toBeLessThan(400);

			await expect(page.locator('h1, h2').filter({ hasText: route.heading }).first()).toBeVisible();

			expect(pageErrors, `page errors on ${route.path}: ${pageErrors.join(' | ')}`).toEqual([]);
			const fatalConsole = consoleErrors.filter(
				(line) => !/favicon|404 .*\.(?:ico|png|jpg|svg|webp)/i.test(line)
			);
			expect(fatalConsole, `console errors on ${route.path}: ${fatalConsole.join(' | ')}`).toEqual(
				[]
			);
		});
	}

	test('legacy /kalkulator redirects to /simulations/kalkulator', async ({ page }) => {
		const response = await page.goto('/kalkulator');
		expect(response!.status()).toBeLessThan(400);
		await expect(page).toHaveURL(/\/simulations\/kalkulator$/);
	});
});
