import { error } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) throw error(500, 'API URL not configured');

		const response = await fetch(`${apiUrl}/api/simulations/prefill`);
		if (!response.ok) throw error(response.status, 'Failed to load prefill data');

		return await response.json();
	} catch (err) {
		if (err instanceof Error && 'status' in err) throw err;
		throw error(500, 'Failed to load simulation data');
	}
};
