import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';
import type { Transaction, TransactionsData } from '$lib/types/transactions';
import { INVESTMENT_CATEGORIES } from '$lib/constants';

export type { Transaction, TransactionsData };

export interface Account {
	id: number;
	name: string;
	category: string;
}

export const load: PageLoad = async ({ fetch, url }) => {
	try {
		const apiUrl = resolveApiUrl();

		// Build query params from URL
		const accountId = url.searchParams.get('account_id');
		const ownerUserId = url.searchParams.get('owner_user_id');
		const dateFrom = url.searchParams.get('date_from');
		const dateTo = url.searchParams.get('date_to');

		const params = new URLSearchParams();
		if (accountId) params.set('account_id', accountId);
		if (ownerUserId) params.set('owner_user_id', ownerUserId);
		if (dateFrom) params.set('date_from', dateFrom);
		if (dateTo) params.set('date_to', dateTo);

		// Fetch transactions with filters
		const transactionsResponse = await fetch(`${apiUrl}/api/transactions?${params.toString()}`);

		if (!transactionsResponse.ok) {
			throw error(transactionsResponse.status, 'Failed to load transactions');
		}

		const transactionsData: TransactionsData = await transactionsResponse.json();

		// Fetch accounts for filter dropdown
		const accountsResponse = await fetch(`${apiUrl}/api/accounts`);
		let investmentAccounts: Account[] = [];

		if (accountsResponse.ok) {
			const accountsData = await accountsResponse.json();

			investmentAccounts = accountsData.assets
				.filter((acc: Account) => INVESTMENT_CATEGORIES.has(acc.category))
				.map((acc: Account) => ({ id: acc.id, name: acc.name, category: acc.category }));
		}

		const ownersResponse = await fetch(`${apiUrl}/api/users`);
		const owners = ownersResponse.ok ? await ownersResponse.json() : [];

		return {
			transactions: transactionsData,
			accounts: investmentAccounts,
			owners,
			filters: {
				account_id: accountId,
				owner_user_id: ownerUserId,
				date_from: dateFrom,
				date_to: dateTo
			}
		};
	} catch (err) {
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load transactions');
	}
};
