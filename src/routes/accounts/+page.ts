import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
import type { PageLoad } from './$types';

export interface Account {
	id: number;
	name: string;
	type: string;
	category: string;
	owner: string;
	currency: string;
	is_active: boolean;
	created_at: string;
	current_value: number;
}

export interface AccountsData {
	assets: Account[];
	liabilities: Account[];
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}
		const response = await fetch(`${apiUrl}/api/accounts`);

		if (!response.ok) {
			throw error(response.status, 'Failed to load accounts');
		}

		const data: AccountsData = await response.json();
		return data;
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load accounts');
	}
};
