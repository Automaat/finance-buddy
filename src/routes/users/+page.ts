import { error, redirect } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	const { user } = await parent();
	if (!user?.isAdmin) {
		redirect(303, '/');
	}

	const apiUrl = resolveApiUrl();

	const response = await fetch(`${apiUrl}/api/auth/users`);
	if (!response.ok) {
		throw error(response.status, 'Nie udało się pobrać użytkowników');
	}
	return { users: await response.json() };
};
