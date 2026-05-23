import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export interface SnapshotListItem {
	id: number;
	date: string;
	notes: string | null;
	total_net_worth: number;
}

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();

	const snapshots = (async () => {
		const response = await fetch(`${apiUrl}/api/snapshots`);
		if (!response.ok) {
			throw error(response.status, 'Failed to load snapshots');
		}
		return (await response.json()) as SnapshotListItem[];
	})();

	return {
		snapshots
	};
};
