import { defineConfig, devices } from '@playwright/test';

const PORT = Number(process.env.E2E_FRONTEND_PORT ?? 4173);
const BASE_URL = process.env.E2E_BASE_URL ?? `http://127.0.0.1:${PORT}`;
const API_URL = process.env.E2E_API_URL ?? 'http://127.0.0.1:8000';
const DATABASE_URL =
	process.env.DATABASE_URL ?? 'postgresql://finance:password@localhost:5432/finance';

const skipWebServer = process.env.E2E_SKIP_WEBSERVER === '1';

export default defineConfig({
	testDir: './e2e',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 2 : 0,
	workers: 1,
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
		{ name: 'mobile-chrome', use: { ...devices['Pixel 7'] } }
	],
	webServer: skipWebServer
		? undefined
		: [
				{
					command: 'uv run uvicorn app.main:app --host 127.0.0.1 --port 8000 --log-level warning',
					cwd: 'backend',
					url: 'http://127.0.0.1:8000/health',
					reuseExistingServer: !process.env.CI,
					timeout: 120_000,
					stdout: 'pipe',
					stderr: 'pipe',
					env: {
						DATABASE_URL,
						APP_PASSWORD: process.env.APP_PASSWORD ?? 'test',
						CORS_ORIGINS: `${BASE_URL},http://localhost:${PORT},http://127.0.0.1:${PORT}`,
						SEED_DEV_DATA: process.env.SEED_DEV_DATA ?? 'true'
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
