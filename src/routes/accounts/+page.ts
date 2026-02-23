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
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}
		const [accountsResponse, personasResponse] = await Promise.all([
			fetch(`${apiUrl}/api/accounts`),
			fetch(`${apiUrl}/api/personas`)
		]);

		if (!accountsResponse.ok) {
			throw error(accountsResponse.status, 'Failed to load accounts');
		}
		if (!personasResponse.ok) {
			throw error(personasResponse.status, 'Failed to load personas');
		}

		const data: AccountsData = await accountsResponse.json();
		const personas: Persona[] = await personasResponse.json();
		return { ...data, personas };
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load accounts');
	}
};
