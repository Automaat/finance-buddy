import { error } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';
import type { PageLoad } from './$types';
import type { Persona } from '$lib/types/personas';

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
	if (!apiUrl) throw error(500, 'API URL not configured');

	const response = await fetch(`${apiUrl}/api/personas`);
	if (!response.ok) throw error(response.status, 'Failed to load personas');

	const personas: Persona[] = await response.json();
	return { personas };
};
