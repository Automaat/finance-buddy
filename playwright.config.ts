import { defineConfig, devices } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD, JWT_SECRET, STORAGE_STATE } from './e2e/credentials';

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
		// Logs in once; the test projects reuse its storageState.
		{ name: 'setup', testMatch: /auth\.setup\.ts/ },
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'], storageState: STORAGE_STATE },
			testIgnore: /auth\.setup\.ts/,
			dependencies: ['setup']
		},
		{
			name: 'mobile-chrome',
			use: { ...devices['Pixel 7'], storageState: STORAGE_STATE },
			testMatch: /navigation\.spec\.ts$/,
			dependencies: ['setup']
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
						CORS_ORIGINS: `${BASE_URL},http://localhost:${PORT},http://127.0.0.1:${PORT}`,
						FB_JWT_SECRET: JWT_SECRET,
						FB_ADMIN_USERNAME: ADMIN_USERNAME,
						FB_ADMIN_PASSWORD: ADMIN_PASSWORD
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
						// SSR talks straight to the backend; the browser routes
						// through the SvelteKit /api proxy at its own origin.
						PUBLIC_API_URL: API_URL,
						PUBLIC_API_URL_BROWSER: BASE_URL,
						API_PROXY_TARGET: API_URL,
						FB_COOKIE_SECURE: 'false'
					}
				}
			]
});
