import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';

export async function load({ fetch }) {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API base URL is not configured');
		}

		// Fetch accounts, assets, and owners in parallel
		const [accountsResponse, assetsResponse, ownersResponse] = await Promise.all([
			fetch(`${apiUrl}/api/accounts`),
			fetch(`${apiUrl}/api/assets`),
			fetch(`${apiUrl}/api/users`)
		]);

		if (!accountsResponse.ok) {
			throw error(accountsResponse.status, 'Failed to load accounts');
		}
		if (!assetsResponse.ok) {
			throw error(assetsResponse.status, 'Failed to load assets');
		}
		if (!ownersResponse.ok) {
			throw error(ownersResponse.status, 'Failed to load owners');
		}

		const accountsData = await accountsResponse.json();
		const assetsData = await assetsResponse.json();
		const owners = await ownersResponse.json();

		return {
			...accountsData,
			physicalAssets: assetsData.assets,
			owners
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load data');
	}
}
