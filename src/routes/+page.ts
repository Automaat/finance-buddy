import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';

export async function load({ fetch }) {
	try {
		const response = await fetch(`${env.PUBLIC_API_URL}/api/dashboard`);

		if (!response.ok) {
			throw error(response.status, 'Failed to load dashboard data');
		}

		const data = await response.json();
		return data;
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err; // Re-throw SvelteKit errors
		}
		throw error(500, 'Failed to load dashboard data');
	}
}
