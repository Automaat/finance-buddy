import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export interface Debt {
	id: number;
	account_id: number;
	account_name: string;
	account_owner_user_id: number | null;
	name: string;
	debt_type: string;
	start_date: string;
	initial_amount: number;
	interest_rate: number;
	currency: string;
	notes: string | null;
	is_active: boolean;
	created_at: string;
	latest_balance: number | null;
	latest_balance_date: string | null;
	total_paid: number;
	interest_paid: number;
}

export interface DebtPayment {
	id: number;
	account_id: number;
	account_name: string;
	amount: number;
	date: string;
	owner_user_id: number | null;
	created_at: string;
}

export interface DebtsListResponse {
	debts: Debt[];
	total_count: number;
	total_initial_amount: number;
	active_debts_count: number;
}

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();
	const [debtsResponse, ownersResponse] = await Promise.all([
		fetch(`${apiUrl}/api/debts`),
		fetch(`${apiUrl}/api/users`)
	]);
	if (!debtsResponse.ok) {
		throw error(debtsResponse.status, 'Failed to load debts');
	}
	const debts = await debtsResponse.json();
	const owners = ownersResponse.ok ? await ownersResponse.json() : [];
	return { ...debts, owners };
};
