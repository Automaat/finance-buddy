import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
import type { PageLoad } from './$types';
import type { Transaction, TransactionsData } from '$lib/types/transactions';
import type { Persona } from '$lib/types/personas';

export interface Account {
	id: number;
	name: string;
	type: string;
	category: string;
	owner: string;
	currency: string;
	account_wrapper: string | null;
	purpose: string;
	is_active: boolean;
	receives_contributions: boolean;
	created_at: string;
	current_value: number;
	square_meters: number | null;
}

export interface AccountsData {
	assets: Account[];
	liabilities: Account[];
}

export type { Transaction, TransactionsData };

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
	if (!apiUrl) {
		throw error(500, 'API URL is not configured');
	}

	const personas = (async (): Promise<Persona[]> => {
		const res = await fetch(`${apiUrl}/api/personas`);
		if (!res.ok) {
			throw error(res.status, 'Failed to load personas');
		}
		return (await res.json()) as Persona[];
	})();

	const accountsData = (async () => {
		const response = await fetch(`${apiUrl}/api/accounts`);
		if (!response.ok) {
			throw error(response.status, 'Failed to load accounts');
		}
		return (await response.json()) as AccountsData;
	})();

	return {
		accountsData,
		personas
	};
};
