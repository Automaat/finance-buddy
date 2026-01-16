import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';

export async function load({ fetch }) {
	try {
		const response = await fetch(`${env.PUBLIC_API_URL}/api/accounts`);

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
