import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
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
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}
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
