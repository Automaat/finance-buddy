import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { SnapshotResponse } from '$lib/types';

export async function load({ params, fetch }) {
	const apiUrl = resolveApiUrl();

	const [snapshotRes, accountsRes, assetsRes, ownersRes] = await Promise.all([
		fetch(`${apiUrl}/api/snapshots/${params.id}`),
		fetch(`${apiUrl}/api/accounts`),
		fetch(`${apiUrl}/api/assets`),
		fetch(`${apiUrl}/api/users`)
	]);

	if (!snapshotRes.ok) {
		throw error(snapshotRes.status, 'Snapshot not found');
	}

	if (!accountsRes.ok || !assetsRes.ok) {
		throw error(500, 'Failed to load accounts or assets');
	}

	if (!ownersRes.ok) {
		throw error(500, 'Failed to load owners');
	}

	const snapshot: SnapshotResponse = await snapshotRes.json();
	const accountsData = await accountsRes.json();
	const assetsData = await assetsRes.json();
	const owners = await ownersRes.json();

	const assets = accountsData.assets;
	const liabilities = accountsData.liabilities;
	const physicalAssets = assetsData.assets;

	return {
		snapshot,
		assets,
		liabilities,
		physicalAssets,
		owners
	};
}
