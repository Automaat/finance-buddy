import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';

export async function load({ fetch }) {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		const response = await fetch(`${apiUrl}/api/accounts`);

		if (!response.ok) {
			throw error(response.status, 'Failed to load accounts');
		}

		const data = await response.json();
		return data;
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load accounts');
	}
}
