import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';

export async function load({ fetch }) {
	try {
		const apiUrl = resolveApiUrl();

		// Fetch accounts, assets, owners, and holdings in parallel. Holdings
		// feed the quote-freshness check; their failure is non-fatal.
		const [accountsResponse, assetsResponse, ownersResponse, holdingsResponse] = await Promise.all([
			fetch(`${apiUrl}/api/accounts`),
			fetch(`${apiUrl}/api/assets`),
			fetch(`${apiUrl}/api/users`),
			fetch(`${apiUrl}/api/holdings`)
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
		const holdings = holdingsResponse.ok ? (await holdingsResponse.json()).holdings : [];

		return {
			...accountsData,
			physicalAssets: assetsData.assets,
			owners,
			holdings
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load data');
	}
}
