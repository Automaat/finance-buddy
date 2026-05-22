// Shared E2E auth config — consumed by playwright.config.ts (to launch the
// backend + frontend with matching settings) and by auth.setup.ts (to log in).
export const ADMIN_USERNAME = process.env.FB_ADMIN_USERNAME ?? 'e2e-admin';
export const ADMIN_PASSWORD = process.env.FB_ADMIN_PASSWORD ?? 'e2e-admin-password';
export const JWT_SECRET = process.env.FB_JWT_SECRET ?? 'e2e-test-jwt-secret';

// Where auth.setup.ts writes the logged-in session for the test projects.
export const STORAGE_STATE = 'e2e/.auth/state.json';
