import { error, redirect } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	const { user } = await parent();
	if (!user?.isAdmin) {
		redirect(303, '/');
	}

	const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
	if (!apiUrl) {
		throw error(500, 'API URL is not configured');
	}

	const response = await fetch(`${apiUrl}/api/auth/users`);
	if (!response.ok) {
		throw error(response.status, 'Nie udało się pobrać użytkowników');
	}
	return { users: await response.json() };
};
