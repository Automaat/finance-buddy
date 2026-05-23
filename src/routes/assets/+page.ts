import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export interface Asset {
	id: number;
	name: string;
	is_active: boolean;
	created_at: string;
	current_value: number;
}

export interface AssetsData {
	assets: Asset[];
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = resolveApiUrl();
		const response = await fetch(`${apiUrl}/api/assets`);

		if (!response.ok) {
			throw error(response.status, 'Failed to load assets');
		}

		const data: AssetsData = await response.json();
		return data;
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load assets');
	}
};
