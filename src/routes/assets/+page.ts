import { error } from '@sveltejs/kit';
import { API_URL_NOT_CONFIGURED_MESSAGE, resolveApiUrl } from '$lib/utils/api';
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
	const apiUrl = resolveApiUrl();
	if (!apiUrl) {
		throw error(500, API_URL_NOT_CONFIGURED_MESSAGE);
	}

	try {
		const response = await fetch(`${apiUrl}/api/assets`);

		if (!response.ok) {
			throw error(response.status, 'Failed to load assets');
		}

		const data: AssetsData = await response.json();
		return data;
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load assets');
	}
};
