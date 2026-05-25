import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';
import type { Transaction, TransactionsData } from '$lib/types/transactions';
import type { OwnerOption } from '$lib/types/owners';

export interface Account {
	id: number;
	name: string;
	type: string;
	category: string;
	owner_user_id: number | null;
	currency: string;
	account_wrapper: string | null;
	purpose: string;
	is_active: boolean;
	receives_contributions: boolean;
	excluded_from_fire: boolean;
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
	const apiUrl = resolveApiUrl();

	const owners = (async (): Promise<OwnerOption[]> => {
		const res = await fetch(`${apiUrl}/api/users`);
		if (!res.ok) {
			throw error(res.status, 'Failed to load owners');
		}
		return (await res.json()) as OwnerOption[];
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
		owners
	};
};
