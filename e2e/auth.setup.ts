import { test as setup } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname } from 'node:path';
import { ADMIN_USERNAME, ADMIN_PASSWORD, STORAGE_STATE } from './credentials';

const API_URL = process.env.E2E_API_URL ?? 'http://127.0.0.1:8000';

// Logs in once as admin and writes a storageState the test projects reuse.
setup('authenticate', async ({ request }) => {
	const response = await request.post(`${API_URL}/api/auth/login`, {
		data: { username: ADMIN_USERNAME, password: ADMIN_PASSWORD }
	});
	if (!response.ok()) {
		throw new Error(`E2E login failed: ${response.status()} ${await response.text()}`);
	}
	const { token } = (await response.json()) as { token: string };

	// A single JWT cookie scoped to the bare host — cookies are not
	// port-specific, so it rides both the frontend (page navigations) and the
	// backend (direct API requests in *.spec.ts).
	mkdirSync(dirname(STORAGE_STATE), { recursive: true });
	writeFileSync(
		STORAGE_STATE,
		JSON.stringify({
			cookies: [
				{
					name: 'fb_token',
					value: token,
					domain: '127.0.0.1',
					path: '/',
					expires: -1,
					httpOnly: true,
					secure: false,
					sameSite: 'Lax'
				}
			],
			origins: []
		})
	);
});
