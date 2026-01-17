import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
import type { PageLoad } from './$types';

export interface Transaction {
	id: number;
	account_id: number;
	account_name: string;
	amount: number;
	date: string;
	owner: string;
	created_at: string;
}

export interface TransactionsData {
	transactions: Transaction[];
	total_invested: number;
	transaction_count: number;
}

export interface Account {
	id: number;
	name: string;
	category: string;
}

export const load: PageLoad = async ({ fetch, url }) => {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}

		// Build query params from URL
		const accountId = url.searchParams.get('account_id');
		const owner = url.searchParams.get('owner');
		const dateFrom = url.searchParams.get('date_from');
		const dateTo = url.searchParams.get('date_to');

		const params = new URLSearchParams();
		if (accountId) params.set('account_id', accountId);
		if (owner) params.set('owner', owner);
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
			const investmentCategories = new Set(['stock', 'bond', 'fund', 'etf']);

			investmentAccounts = accountsData.assets
				.filter((acc: Account) => investmentCategories.has(acc.category))
				.map((acc: Account) => ({ id: acc.id, name: acc.name, category: acc.category }));
		}

		return {
			transactions: transactionsData,
			accounts: investmentAccounts,
			filters: {
				account_id: accountId,
				owner,
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
