import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export interface SecurityRow {
	id: number;
	symbol: string;
	isin: string | null;
	name: string;
	asset_type: string;
	currency: string;
	created_at: string;
}

export interface HoldingRow {
	security: SecurityRow;
	quantity: string;
	average_cost: string;
	cost_basis: string;
	latest_quote: string | null;
	latest_quote_date: string | null;
	market_value: string;
	unrealized_gain: string;
	realized_gain: string;
}

export interface AccountOption {
	id: number;
	name: string;
}

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();
	const [holdingsRes, securitiesRes, accountsRes] = await Promise.all([
		fetch(`${apiUrl}/api/holdings`),
		fetch(`${apiUrl}/api/holdings/securities`),
		fetch(`${apiUrl}/api/accounts`)
	]);
	if (!holdingsRes.ok) throw error(holdingsRes.status, 'Failed to load holdings');
	if (!securitiesRes.ok) throw error(securitiesRes.status, 'Failed to load securities');
	if (!accountsRes.ok) throw error(accountsRes.status, 'Failed to load accounts');
	const holdings = (await holdingsRes.json()) as { holdings: HoldingRow[] };
	const securities = (await securitiesRes.json()) as { securities: SecurityRow[] };
	const accountsData = (await accountsRes.json()) as { accounts: AccountOption[] };
	return {
		holdings: holdings.holdings,
		securities: securities.securities,
		accounts: accountsData.accounts.map((a) => ({ id: a.id, name: a.name }))
	};
};
