import { error } from '@sveltejs/kit';
import { API_URL_NOT_CONFIGURED_MESSAGE, resolveApiUrl } from '$lib/utils/api';
import type { PageLoad } from './$types';
import type { Persona } from '$lib/types/personas';

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();
	if (!apiUrl) throw error(500, API_URL_NOT_CONFIGURED_MESSAGE);

	const response = await fetch(`${apiUrl}/api/personas`);
	if (!response.ok) throw error(response.status, 'Failed to load personas');

	const personas: Persona[] = await response.json();
	return { personas };
};
