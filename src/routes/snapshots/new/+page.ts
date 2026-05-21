import { error } from '@sveltejs/kit';
import { API_URL_NOT_CONFIGURED_MESSAGE, resolveApiUrl } from '$lib/utils/api';

export async function load({ fetch }) {
	const apiUrl = resolveApiUrl();
	if (!apiUrl) {
		throw error(500, API_URL_NOT_CONFIGURED_MESSAGE);
	}

	try {
		// Fetch accounts, assets, and personas in parallel
		const [accountsResponse, assetsResponse, personasResponse] = await Promise.all([
			fetch(`${apiUrl}/api/accounts`),
			fetch(`${apiUrl}/api/assets`),
			fetch(`${apiUrl}/api/personas`)
		]);

		if (!accountsResponse.ok) {
			throw error(accountsResponse.status, 'Failed to load accounts');
		}
		if (!assetsResponse.ok) {
			throw error(assetsResponse.status, 'Failed to load assets');
		}
		if (!personasResponse.ok) {
			throw error(personasResponse.status, 'Failed to load personas');
		}

		const accountsData = await accountsResponse.json();
		const assetsData = await assetsResponse.json();
		const personas = await personasResponse.json();

		return {
			...accountsData,
			physicalAssets: assetsData.assets,
			personas
		};
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load data');
	}
}
