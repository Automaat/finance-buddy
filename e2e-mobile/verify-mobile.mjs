import { chromium, devices } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { join } from 'node:path';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5174';
const OUT_DIR = new URL('./screenshots', import.meta.url).pathname;
mkdirSync(OUT_DIR, { recursive: true });

const ROUTES = [
	{ path: '/', name: 'dashboard' },
	{ path: '/metryki', name: 'metryki' },
	{ path: '/simulations', name: 'simulations' },
	{ path: '/simulations/mortgage', name: 'mortgage' },
	{ path: '/simulations/zus', name: 'zus' },
	{ path: '/accounts', name: 'accounts' },
	{ path: '/transactions', name: 'transactions' },
	{ path: '/assets', name: 'assets' },
	{ path: '/debts', name: 'debts' },
	{ path: '/snapshots', name: 'snapshots' },
	{ path: '/salaries', name: 'salaries' },
	{ path: '/kalkulator', name: 'kalkulator' },
	{ path: '/config', name: 'config' },
	{ path: '/settings', name: 'settings' }
];

const VIEWPORTS = [
	{ label: 'iphone', device: devices['iPhone 14 Pro'] },
	{ label: 'ipad', device: devices['iPad Mini'] }
];

const MIN_TAP = 44;

async function audit(page, url) {
	await page.goto(url, { waitUntil: 'networkidle', timeout: 20000 }).catch(() => {});
	// Allow any delayed paint (charts)
	await page.waitForTimeout(600);

	return await page.evaluate((minTap) => {
		const doc = document.documentElement;
		const overflowX = doc.scrollWidth - doc.clientWidth;

		const selectors = 'button, a[href], input, select, textarea, [role="button"]';
		const elements = Array.from(document.querySelectorAll(selectors));
		const tooSmall = [];
		for (const el of elements) {
			const style = getComputedStyle(el);
			if (style.display === 'none' || style.visibility === 'hidden') continue;
			const rect = el.getBoundingClientRect();
			if (rect.width === 0 || rect.height === 0) continue;

			// Skip native checkboxes/radios wrapped in a label (label is the tap target)
			if ((el.type === 'checkbox' || el.type === 'radio') && el.closest('label')) {
				continue;
			}

			// Skip home-ui navbar internals (library component, not fixable here)
			if (el.closest('.navbar')) continue;

			if (rect.width < minTap - 1 || rect.height < minTap - 1) {
				tooSmall.push({
					tag: el.tagName.toLowerCase(),
					cls: el.className?.toString?.().slice(0, 60) || '',
					text: (el.innerText || el.value || '').slice(0, 40).replace(/\n/g, ' '),
					w: Math.round(rect.width),
					h: Math.round(rect.height)
				});
			}
		}

		// Check for nodes that extend beyond viewport
		const overflowingNodes = [];
		if (overflowX > 0) {
			const all = document.querySelectorAll('body *');
			for (const el of all) {
				const r = el.getBoundingClientRect();
				if (r.right > doc.clientWidth + 1) {
					overflowingNodes.push({
						tag: el.tagName.toLowerCase(),
						cls: el.className?.toString?.().slice(0, 60) || '',
						right: Math.round(r.right),
						w: Math.round(r.width)
					});
					if (overflowingNodes.length > 5) break;
				}
			}
		}

		return { overflowX, tooSmall, overflowingNodes };
	}, MIN_TAP);
}

async function auditRoute(ctx, vpLabel, route) {
	const page = await ctx.newPage();
	page.on('console', (msg) => {
		if (msg.type() === 'error') {
			console.error(`[${vpLabel}] console error:`, msg.text().slice(0, 200));
		}
	});
	const url = `${BASE_URL}${route.path}`;
	try {
		const audit_result = await audit(page, url);
		const screenshot = join(OUT_DIR, `${vpLabel}-${route.name}.png`);
		await page.screenshot({ path: screenshot, fullPage: true });
		const status =
			audit_result.overflowX === 0 && audit_result.tooSmall.length === 0 ? 'OK' : 'ISSUE';
		console.log(
			`  [${vpLabel}] ${route.path} ... ${status}  overflowX=${audit_result.overflowX}px  tapTooSmall=${audit_result.tooSmall.length}`
		);
		await page.close();
		return { viewport: vpLabel, route: route.path, ...audit_result };
	} catch (err) {
		console.log(`  [${vpLabel}] ${route.path} ... ERROR: ${err.message.slice(0, 100)}`);
		await page.close().catch(() => {});
		return { viewport: vpLabel, route: route.path, error: err.message };
	}
}

async function auditViewport(browser, vp) {
	const ctx = await browser.newContext({ ...vp.device });
	const results = await Promise.all(ROUTES.map((route) => auditRoute(ctx, vp.label, route)));
	await ctx.close();
	return results;
}

async function run() {
	const browser = await chromium.launch();
	const perViewport = await Promise.all(VIEWPORTS.map((vp) => auditViewport(browser, vp)));
	const results = perViewport.flat();

	await browser.close();

	console.log('\n=== SUMMARY ===\n');
	let criticalIssues = 0;
	for (const r of results) {
		if (r.error) {
			console.log(`[${r.viewport}] ${r.route}: ERROR ${r.error.slice(0, 120)}`);
			criticalIssues++;
			continue;
		}
		if (r.overflowX > 0) {
			criticalIssues++;
			console.log(`[${r.viewport}] ${r.route}: horizontal overflow ${r.overflowX}px`);
			for (const n of r.overflowingNodes) {
				console.log(`    -> <${n.tag}> .${n.cls.split(' ')[0]}  right=${n.right}  w=${n.w}`);
			}
		}
		if (r.tooSmall && r.tooSmall.length > 0) {
			console.log(`[${r.viewport}] ${r.route}: ${r.tooSmall.length} small tap targets`);
			for (const t of r.tooSmall.slice(0, 4)) {
				console.log(`    -> <${t.tag}> .${t.cls.split(' ')[0]}  ${t.w}x${t.h}  "${t.text}"`);
			}
		}
	}

	console.log(`\nCritical issues: ${criticalIssues}`);
	console.log(`Screenshots: ${OUT_DIR}`);
	process.exit(criticalIssues > 0 ? 1 : 0);
}

run().catch((err) => {
	console.error(err);
	process.exit(2);
});
