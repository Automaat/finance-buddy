import { browser } from '$app/environment';
import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import type { PageLoad } from './$types';

export interface Debt {
	id: number;
	account_id: number;
	account_name: string;
	account_owner: string;
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
	owner: string;
	created_at: string;
}

export interface DebtsListResponse {
	debts: Debt[];
	total_count: number;
	total_initial_amount: number;
	active_debts_count: number;
}

export const load: PageLoad = async ({ fetch }): Promise<DebtsListResponse> => {
	const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
	const response = await fetch(`${apiUrl}/api/debts`);
	if (!response.ok) {
		throw error(response.status, 'Failed to load debts');
	}
	return response.json();
};
