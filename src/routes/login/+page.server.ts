import { fail, redirect } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { Actions } from './$types';

const cookieSecure = env.FB_COOKIE_SECURE === 'true';

export const actions: Actions = {
	default: async ({ request, cookies, fetch }) => {
		const form = await request.formData();
		const username = String(form.get('username') ?? '').trim();
		const password = String(form.get('password') ?? '');
		const rememberMe = form.get('remember_me') === 'on';

		if (!username || !password) {
			return fail(400, { error: 'Podaj nazwę użytkownika i hasło', username });
		}

		const response = await fetch('/api/auth/login', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ username, password, remember_me: rememberMe })
		});
		if (response.status === 401) {
			return fail(401, { error: 'Nieprawidłowa nazwa użytkownika lub hasło', username });
		}
		if (!response.ok) {
			// A non-401 failure is a backend/network problem, not bad credentials.
			return fail(response.status, {
				error: 'Logowanie chwilowo niedostępne — spróbuj ponownie później',
				username
			});
		}

		const { token } = (await response.json()) as { token: string };
		cookies.set('fb_token', token, {
			path: '/',
			httpOnly: true,
			sameSite: 'lax',
			secure: cookieSecure,
			// "Remember me" persists for 5 days; otherwise a session cookie.
			maxAge: rememberMe ? 5 * 24 * 60 * 60 : undefined
		});
		redirect(303, '/');
	}
};
