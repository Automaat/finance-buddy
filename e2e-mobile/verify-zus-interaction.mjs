import { chromium, devices } from '@playwright/test';
import { expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5174';

const VIEWPORTS = [
	{ label: 'iphone', device: devices['iPhone 14 Pro'] },
	{ label: 'ipad', device: devices['iPad Mini'] }
];

async function testViewport(browser, vp) {
	const ctx = await browser.newContext({ ...vp.device });
	try {
		const page = await ctx.newPage();
		const consoleErrors = [];
		page.on('pageerror', (err) => consoleErrors.push(err.message));
		page.on('console', (msg) => {
			if (msg.type() === 'error') consoleErrors.push(msg.text().slice(0, 400));
		});
		page.on('requestfailed', (r) =>
			consoleErrors.push(`request failed: ${r.url()} ${r.failure()?.errorText}`)
		);
		page.on('response', async (r) => {
			if (r.url().includes('/api/zus') && r.status() >= 400) {
				const body = await r.text().catch(() => '');
				consoleErrors.push(`zus api ${r.status()}: ${body.slice(0, 400)}`);
			}
		});

		console.log(`\n[${vp.label}] visiting /simulations/zus`);
		await page.goto(`${BASE_URL}/simulations/zus`, { waitUntil: 'networkidle' });

		// Wait for the form heading instead of a fixed sleep — proves the page
		// rendered before we touch fields.
		const formHeading = page.getByRole('heading', { name: 'Parametry' });
		await expect(formHeading).toBeVisible();
		const runButton = page.getByRole('button', { name: /Oblicz emeryturę/i });
		await expect(runButton).toBeVisible();
		console.log(`  [${vp.label}] form + run button: visible`);

		let result;
		try {
			console.log(`  [${vp.label}] filling form fields...`);
			await page.getByLabel('Data urodzenia').fill('1990-06-15');
			await page.getByLabel('Wiek emerytalny').fill('65');
			console.log(`  [${vp.label}] clicking run button...`);
			await runButton.click();

			// Results render under the "Wyniki" heading — wait for it instead
			// of a CSS class that didn't survive the Skeleton-Tailwind redesign.
			const resultsHeading = page.getByRole('heading', { name: 'Wyniki' });
			await expect(resultsHeading).toBeVisible({ timeout: 15000 });
			console.log(`  [${vp.label}] results rendered: OK`);

			const pensionBruttoLabel = page.getByText('Emerytura brutto').first();
			await expect(pensionBruttoLabel).toBeVisible();

			// Yearly projection table lives inside a <details> element behind
			// "Projekcja roczna" — open it before measuring scroll.
			const projectionToggle = page.getByText('Projekcja roczna').first();
			if (await projectionToggle.isVisible()) {
				await projectionToggle.click();
				const projectionTable = page.locator('details table').first();
				await expect(projectionTable).toBeVisible();
				const tableScroll = await projectionTable.evaluate((el) => ({
					scrollWidth: el.scrollWidth,
					clientWidth: el.clientWidth,
					overflow: getComputedStyle(el).overflowX
				}));
				console.log(
					`  [${vp.label}] projection table: scrollW=${tableScroll.scrollWidth} clientW=${tableScroll.clientWidth} overflow=${tableScroll.overflow}`
				);
			}

			const pageOverflow = await page.evaluate(
				() => document.documentElement.scrollWidth - document.documentElement.clientWidth
			);
			console.log(`  [${vp.label}] page overflow x: ${pageOverflow}px`);

			result = { viewport: vp.label, success: true };
		} catch (err) {
			console.log(`  [${vp.label}] ERROR: ${err.message.slice(0, 200)}`);
			result = { viewport: vp.label, success: false, error: err.message };
		}

		if (consoleErrors.length > 0) {
			console.log(`  [${vp.label}] console/page errors:`);
			for (const e of consoleErrors.slice(0, 3)) console.log(`    - ${e}`);
		}

		return result;
	} finally {
		await ctx.close();
	}
}

async function run() {
	const browser = await chromium.launch();
	const results = await Promise.all(VIEWPORTS.map((vp) => testViewport(browser, vp)));
	await browser.close();

	const ok = results.every((r) => r.success);
	console.log(`\n${ok ? 'PASS' : 'FAIL'}: ZUS interactive test`);
	process.exit(ok ? 0 : 1);
}

run().catch((err) => {
	console.error(err);
	process.exit(2);
});
