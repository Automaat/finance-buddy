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

export interface AccountPosition {
	account_id: number;
	account_name: string;
	owner_user_id: number;
	quantity: string;
	average_cost: string;
	cost_basis: string;
	market_value: string;
	unrealized_gain: string;
	realized_gain: string;
	average_cost_pln: string | null;
	cost_basis_pln: string | null;
	market_value_pln: string | null;
	unrealized_gain_pln: string | null;
	realized_gain_pln: string | null;
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
	average_cost_pln: string | null;
	cost_basis_pln: string | null;
	market_value_pln: string | null;
	unrealized_gain_pln: string | null;
	realized_gain_pln: string | null;
	latest_quote_rate_pln: string | null;
	accounts: AccountPosition[];
}

export interface AccountOption {
	id: number;
	name: string;
}

export interface DividendRow {
	id: number;
	account_id: number;
	security_id: number;
	pay_date: string;
	gross_amount: string;
	withholding_tax: string;
	net_amount: string;
	currency: string;
	created_at: string;
}

interface AccountRow {
	id: number;
	name: string;
	category: string;
	is_active: boolean;
}

const INVESTMENT_CATEGORIES = new Set(['stock', 'bond', 'fund', 'etf']);

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();
	const [holdingsRes, securitiesRes, accountsRes, dividendsRes] = await Promise.all([
		fetch(`${apiUrl}/api/holdings`),
		fetch(`${apiUrl}/api/holdings/securities`),
		fetch(`${apiUrl}/api/accounts`),
		fetch(`${apiUrl}/api/holdings/dividends`)
	]);
	if (!holdingsRes.ok) throw error(holdingsRes.status, 'Failed to load holdings');
	if (!securitiesRes.ok) throw error(securitiesRes.status, 'Failed to load securities');
	if (!accountsRes.ok) throw error(accountsRes.status, 'Failed to load accounts');
	if (!dividendsRes.ok) throw error(dividendsRes.status, 'Failed to load dividends');
	const holdings = (await holdingsRes.json()) as { holdings: HoldingRow[] };
	const securities = (await securitiesRes.json()) as { securities: SecurityRow[] };
	const accountsPayload = (await accountsRes.json()) as {
		assets: AccountRow[];
		liabilities: AccountRow[];
	};
	const accounts = (accountsPayload.assets ?? [])
		.filter((a) => a.is_active && INVESTMENT_CATEGORIES.has(a.category))
		.map((a) => ({ id: a.id, name: a.name }));
	const dividends = ((await dividendsRes.json()) as { dividends: DividendRow[] }).dividends;
	return {
		holdings: holdings.holdings,
		securities: securities.securities,
		accounts,
		dividends
	};
};
