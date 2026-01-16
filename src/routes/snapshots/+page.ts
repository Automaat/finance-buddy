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
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}

		const response = await fetch(`${apiUrl}/api/snapshots`);

		if (!response.ok) {
			throw error(response.status, 'Failed to load snapshots');
		}

		const snapshots: SnapshotListItem[] = await response.json();

		return {
			snapshots
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load snapshots');
	}
};
