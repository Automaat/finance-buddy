import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
import type { PageLoad } from './$types';

export interface SnapshotListItem {
	id: number;
	date: string;
	notes: string | null;
	total_net_worth: number;
}

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
	if (!apiUrl) {
		throw error(500, 'API URL is not configured');
	}

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
