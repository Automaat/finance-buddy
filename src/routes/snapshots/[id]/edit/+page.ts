import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import type { SnapshotResponse } from '$lib/types';

export async function load({ params, fetch }) {
	const API_URL = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';

	const [snapshotRes, accountsRes, assetsRes, personasRes] = await Promise.all([
		fetch(`${API_URL}/api/snapshots/${params.id}`),
		fetch(`${API_URL}/api/accounts`),
		fetch(`${API_URL}/api/assets`),
		fetch(`${API_URL}/api/personas`)
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
	const allAccounts = await accountsRes.json();
	const allAssets = await assetsRes.json();
	const personas = await personasRes.json();

	const assets = allAccounts.filter((a: any) => a.type === 'asset');
	const liabilities = allAccounts.filter((a: any) => a.type === 'liability');
	const physicalAssets = allAssets;

	return {
		snapshot,
		assets,
		liabilities,
		physicalAssets,
		personas
	};
}
