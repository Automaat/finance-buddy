import { chromium, devices } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5174';

async function run() {
	const browser = await chromium.launch();
	const results = [];

	for (const vp of [
		{ label: 'iphone', device: devices['iPhone 14 Pro'] },
		{ label: 'ipad', device: devices['iPad Mini'] }
	]) {
		const ctx = await browser.newContext({ ...vp.device });
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
		await page.waitForTimeout(500);

		const formVisible = await page.locator('.form-section').isVisible();
		const runButton = page.locator('button.primary-button');
		const runButtonVisible = await runButton.isVisible();

		console.log(`  form visible: ${formVisible}`);
		console.log(`  run button visible: ${runButtonVisible}`);

		if (runButtonVisible) {
			console.log('  filling form fields...');
			await page.locator('input[type="date"]').first().fill('1990-06-15');
			const retirementAgeInput = page.locator('input[type="number"]').nth(0);
			// Set a valid retirement age (first numeric input is retirement age in this form)
			await retirementAgeInput.fill('65');
			await page.waitForTimeout(200);
			console.log('  clicking run button...');
			await runButton.click();

			// Wait for results
			try {
				await page.waitForSelector('.summary-cards', { timeout: 15000 });
				console.log('  results rendered: OK');

				const summaryCards = await page.locator('.summary-card').count();
				console.log(`  summary cards: ${summaryCards}`);

				const chartHeight = await page
					.locator('.chart-container')
					.evaluate((el) => el.offsetHeight);
				console.log(`  chart height: ${chartHeight}px`);

				// Check if table still scrolls nicely on mobile
				const tableWrapper = page.locator('.projection-table').first();
				const hasTable = (await tableWrapper.count()) > 0;
				if (hasTable) {
					const tableScroll = await tableWrapper.evaluate((el) => ({
						scrollWidth: el.scrollWidth,
						clientWidth: el.clientWidth,
						overflow: getComputedStyle(el).overflowX
					}));
					console.log(
						`  projection table: scrollW=${tableScroll.scrollWidth} clientW=${tableScroll.clientWidth} overflow=${tableScroll.overflow}`
					);
				}

				// Verify no page-level horizontal overflow
				const pageOverflow = await page.evaluate(
					() => document.documentElement.scrollWidth - document.documentElement.clientWidth
				);
				console.log(`  page overflow x: ${pageOverflow}px`);

				results.push({ viewport: vp.label, success: true });
			} catch (err) {
				console.log(`  ERROR waiting for results: ${err.message}`);
				results.push({ viewport: vp.label, success: false, error: err.message });
			}
		}

		if (consoleErrors.length > 0) {
			console.log('  console/page errors:');
			for (const e of consoleErrors.slice(0, 3)) console.log(`    - ${e}`);
		}

		await ctx.close();
	}

	await browser.close();

	const ok = results.every((r) => r.success);
	console.log(`\n${ok ? 'PASS' : 'FAIL'}: ZUS interactive test`);
	process.exit(ok ? 0 : 1);
}

run().catch((err) => {
	console.error(err);
	process.exit(2);
});
