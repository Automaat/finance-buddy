import { defineConfig, devices } from '@playwright/test';

const PORT = Number(process.env.E2E_FRONTEND_PORT ?? 4173);
const BASE_URL = process.env.E2E_BASE_URL ?? `http://127.0.0.1:${PORT}`;
const API_URL = process.env.E2E_API_URL ?? 'http://127.0.0.1:8000';
const DATABASE_URL =
	process.env.E2E_DATABASE_URL ??
	process.env.DATABASE_URL ??
	'postgresql://finance:password@localhost:5433/finance';

const skipWebServer = process.env.E2E_SKIP_WEBSERVER === '1';

export default defineConfig({
	testDir: './e2e',
	fullyParallel: true,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	workers: process.env.CI ? 4 : undefined,
	reporter: process.env.CI
		? [['github'], ['html', { open: 'never' }], ['list']]
		: [['html', { open: 'never' }], ['list']],
	timeout: 60_000,
	expect: { timeout: 10_000 },
	use: {
		baseURL: BASE_URL,
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure',
		video: 'retain-on-failure',
		actionTimeout: 15_000,
		navigationTimeout: 30_000
	},
	projects: [
		{ name: 'chromium', use: { ...devices['Desktop Chrome'] } },
		{
			name: 'mobile-chrome',
			use: { ...devices['Pixel 7'] },
			testMatch: /navigation\.spec\.ts$/
		}
	],
	webServer: skipWebServer
		? undefined
		: [
				{
					// backend-go applies internal/db/schema.sql to an empty database
					// itself; for a local run seed fixtures separately if needed.
					command: 'go run ./cmd/api',
					cwd: 'backend-go',
					url: 'http://127.0.0.1:8000/health',
					reuseExistingServer: !process.env.CI,
					timeout: 120_000,
					stdout: 'pipe',
					stderr: 'pipe',
					env: {
						DATABASE_URL,
						FB_ADDR: '127.0.0.1:8000',
						CORS_ORIGINS: `${BASE_URL},http://localhost:${PORT},http://127.0.0.1:${PORT}`
					}
				},
				{
					command: 'npm run build && node build',
					url: BASE_URL,
					reuseExistingServer: !process.env.CI,
					timeout: 180_000,
					stdout: 'pipe',
					stderr: 'pipe',
					env: {
						NODE_ENV: 'production',
						PORT: String(PORT),
						HOST: '127.0.0.1',
						ORIGIN: BASE_URL,
						PUBLIC_API_URL: API_URL,
						PUBLIC_API_URL_BROWSER: API_URL
					}
				}
			]
});
