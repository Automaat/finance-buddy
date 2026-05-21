import { error } from '@sveltejs/kit';
import { API_URL_NOT_CONFIGURED_MESSAGE, resolveApiUrl } from '$lib/utils/api';
import type { SnapshotResponse } from '$lib/types';

export async function load({ params, fetch }) {
	const apiUrl = resolveApiUrl();
	if (!apiUrl) {
		throw error(500, API_URL_NOT_CONFIGURED_MESSAGE);
	}

	const [snapshotRes, accountsRes, assetsRes, personasRes] = await Promise.all([
		fetch(`${apiUrl}/api/snapshots/${params.id}`),
		fetch(`${apiUrl}/api/accounts`),
		fetch(`${apiUrl}/api/assets`),
		fetch(`${apiUrl}/api/personas`)
	]);

	if (!snapshotRes.ok) {
		throw error(snapshotRes.status, 'Snapshot not found');
	}

	if (!accountsRes.ok || !assetsRes.ok) {
		throw error(500, 'Failed to load accounts or assets');
	}

	if (!personasRes.ok) {
		throw error(500, 'Failed to load personas');
	}

	const snapshot: SnapshotResponse = await snapshotRes.json();
	const accountsData = await accountsRes.json();
	const assetsData = await assetsRes.json();
	const personas = await personasRes.json();

	const assets = accountsData.assets;
	const liabilities = accountsData.liabilities;
	const physicalAssets = assetsData.assets;

	return {
		snapshot,
		assets,
		liabilities,
		physicalAssets,
		personas
	};
}
